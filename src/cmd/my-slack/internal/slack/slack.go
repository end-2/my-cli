package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/pkg/cliutil"
)

const (
	DefaultAPIBaseURL = "https://slack.com/api/"
	DefaultUserAgent  = "my-cli/my-slack"
	DefaultTimeout    = 15 * time.Second
	DefaultListLimit  = 100
	MaxListLimit      = 1000
)

var methodPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-zA-Z0-9]*)+$`)

type Request struct {
	Kind       string         `json:"kind"`
	Method     string         `json:"method"`
	Args       map[string]any `json:"args,omitempty"`
	Limit      *int           `json:"limit,omitempty"`
	Cursor     string         `json:"cursor,omitempty"`
	ListField  string         `json:"list_field,omitempty"`
	HTTPMethod string         `json:"http_method,omitempty"`
	BaseURL    string         `json:"base_url,omitempty"`
	Alias      string         `json:"alias,omitempty"`
}

func ParseRequest(raw string) (Request, error) {
	request, err := cliutil.DecodeStrictJSON[Request](raw)
	if err != nil {
		return Request{}, err
	}

	if err := request.normalize(); err != nil {
		return Request{}, err
	}

	return request, nil
}

func (r *Request) normalize() error {
	r.Kind = normalizeKind(r.Kind)
	r.Method = strings.TrimSpace(r.Method)
	r.Cursor = strings.TrimSpace(r.Cursor)
	r.ListField = strings.TrimSpace(r.ListField)
	r.HTTPMethod = strings.ToUpper(strings.TrimSpace(r.HTTPMethod))
	r.BaseURL = strings.TrimSpace(r.BaseURL)
	r.Alias = strings.TrimSpace(r.Alias)

	if r.Method == "" {
		return errors.New("json input field \"method\" is required")
	}

	if !methodPattern.MatchString(r.Method) {
		return errors.New("json input field \"method\" must be a Slack Web API method like \"conversations.list\"")
	}

	if r.HTTPMethod != "" && r.HTTPMethod != http.MethodGet && r.HTTPMethod != http.MethodPost {
		return errors.New("json input field \"http_method\" must be either \"GET\" or \"POST\"")
	}

	switch r.Kind {
	case "create", "read", "update", "delete":
		if r.Limit != nil {
			return fmt.Errorf("json input field \"limit\" is not allowed for kind %q", r.Kind)
		}

		if r.Cursor != "" {
			return fmt.Errorf("json input field \"cursor\" is not allowed for kind %q", r.Kind)
		}

		if r.ListField != "" {
			return fmt.Errorf("json input field \"list_field\" is not allowed for kind %q", r.Kind)
		}
	case "list":
		limitFromArgs, hasLimitInArgs, err := extractOptionalIntArg(r.Args, "limit")
		if err != nil {
			return fmt.Errorf("json input field \"args.limit\": %w", err)
		}

		if hasLimitInArgs {
			if r.Limit != nil && *r.Limit != limitFromArgs {
				return errors.New("json input fields \"limit\" and \"args.limit\" must match when both are provided")
			}

			if r.Limit == nil {
				r.Limit = &limitFromArgs
			}
		}

		if r.Limit != nil && (*r.Limit < 1 || *r.Limit > MaxListLimit) {
			return fmt.Errorf("json input field \"limit\" must be between 1 and %d for kind %q", MaxListLimit, r.Kind)
		}

		cursorFromArgs, hasCursorInArgs, err := extractOptionalStringArg(r.Args, "cursor")
		if err != nil {
			return fmt.Errorf("json input field \"args.cursor\": %w", err)
		}

		if hasCursorInArgs {
			if r.Cursor != "" && r.Cursor != cursorFromArgs {
				return errors.New("json input fields \"cursor\" and \"args.cursor\" must match when both are provided")
			}

			if r.Cursor == "" {
				r.Cursor = cursorFromArgs
			}
		}
	default:
		return fmt.Errorf(
			"json input field \"kind\" must be one of %q, %q, %q, %q, or %q",
			"create",
			"read",
			"update",
			"delete",
			"list",
		)
	}

	return nil
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "post":
		return "create"
	case "get":
		return "read"
	case "put", "patch":
		return "update"
	case "remove":
		return "delete"
	case "ls":
		return "list"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

type ClientConfig struct {
	BaseURL   string
	Token     string
	Timeout   time.Duration
	UserAgent string
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL:   DefaultAPIBaseURL,
		Timeout:   DefaultTimeout,
		UserAgent: DefaultUserAgent,
	}
}

func NormalizeBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("parse slack api base url: empty value")
	}

	baseURL, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse slack api base url: %w", err)
	}

	switch {
	case baseURL.Path == "":
		baseURL.Path = "/"
	case !strings.HasSuffix(baseURL.Path, "/"):
		baseURL.Path += "/"
	}

	return baseURL.String(), nil
}

func (c ClientConfig) withDefaults() ClientConfig {
	config := DefaultClientConfig()

	if value := strings.TrimSpace(c.BaseURL); value != "" {
		config.BaseURL = value
	}

	if value := strings.TrimSpace(c.Token); value != "" {
		config.Token = value
	}

	if value := strings.TrimSpace(c.UserAgent); value != "" {
		config.UserAgent = value
	}

	if c.Timeout > 0 {
		config.Timeout = c.Timeout
	}

	return config
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
	userAgent  string
}

func NewClient(config ClientConfig, httpClient *http.Client) (*Client, error) {
	config = config.withDefaults()

	normalizedBaseURL, err := NormalizeBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(normalizedBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse slack api base url: %w", err)
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		token:      config.Token,
		userAgent:  config.UserAgent,
	}, nil
}

func (c *Client) AuthMode() string {
	if c.token != "" {
		return "token"
	}

	return "none"
}

type RequestPlan struct {
	URL         *url.URL
	Path        string
	HTTPMethod  string
	ContentType string
	Body        []byte
}

func (c *Client) BuildRequest(input Request) (RequestPlan, error) {
	endpoint, err := c.baseURL.Parse(input.Method)
	if err != nil {
		return RequestPlan{}, fmt.Errorf("build slack api url: %w", err)
	}

	args := normalizedArgs(input)
	httpMethod := requestHTTPMethod(input)

	plan := RequestPlan{
		URL:        endpoint,
		Path:       endpoint.RequestURI(),
		HTTPMethod: httpMethod,
	}

	switch httpMethod {
	case http.MethodGet:
		query := endpoint.Query()
		if err := encodeArgsToQuery(query, args); err != nil {
			return RequestPlan{}, err
		}

		endpoint.RawQuery = query.Encode()
		plan.URL = endpoint
		plan.Path = endpoint.RequestURI()
	case http.MethodPost:
		body, err := json.Marshal(args)
		if err != nil {
			return RequestPlan{}, fmt.Errorf("encode slack api request body: %w", err)
		}

		plan.Body = body
		plan.ContentType = "application/json; charset=utf-8"
	default:
		return RequestPlan{}, fmt.Errorf("unsupported http method %q", httpMethod)
	}

	return plan, nil
}

func (c *Client) Execute(plan RequestPlan, input Request) (any, error) {
	if input.Kind == "list" {
		return c.executeList(input)
	}

	payload, err := c.executePlan(plan)
	if err != nil {
		return nil, err
	}

	return responseEnvelope{
		Kind:     input.Kind,
		Method:   input.Method,
		Response: payload,
	}, nil
}

func (c *Client) executeList(input Request) (listEnvelope, error) {
	limit := listLimit(input)
	items := make([]any, 0, limit)
	current := input

	var listField string
	var mergedResponse map[string]any
	var nextCursor string

	for len(items) < limit {
		plan, err := c.BuildRequest(current)
		if err != nil {
			return listEnvelope{}, err
		}

		payload, err := c.executePlan(plan)
		if err != nil {
			return listEnvelope{}, err
		}

		if listField == "" {
			if current.ListField != "" {
				listField = current.ListField
			} else {
				listField, err = inferListField(payload)
				if err != nil {
					return listEnvelope{}, err
				}
			}
		}

		pageItems, err := extractListItems(payload, listField)
		if err != nil {
			return listEnvelope{}, err
		}

		if mergedResponse == nil {
			mergedResponse = cloneMap(payload)
		}

		remaining := limit - len(items)
		if len(pageItems) > remaining {
			pageItems = pageItems[:remaining]
		}

		items = append(items, cloneSlice(pageItems)...)
		nextCursor = nextCursorFromPayload(payload)

		if len(pageItems) == 0 || nextCursor == "" {
			break
		}

		current.Cursor = nextCursor
	}

	if mergedResponse == nil {
		mergedResponse = map[string]any{"ok": true}
	}

	mergedResponse[listField] = items
	mergedResponse = withNextCursor(mergedResponse, nextCursor)

	return listEnvelope{
		Kind:   "list",
		Method: input.Method,
		List: listOutput{
			Field:      listField,
			Limit:      limit,
			Count:      len(items),
			NextCursor: nextCursor,
			Items:      items,
		},
		Response: mergedResponse,
	}, nil
}

func (c *Client) executePlan(plan RequestPlan) (map[string]any, error) {
	resp, err := c.do(plan)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return decodeAPIResponse(resp, plan.Path)
}

func (c *Client) do(plan RequestPlan) (*http.Response, error) {
	var body io.Reader
	if len(plan.Body) > 0 {
		body = bytes.NewReader(plan.Body)
	}

	req, err := http.NewRequest(plan.HTTPMethod, plan.URL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create slack api request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if plan.ContentType != "" {
		req.Header.Set("Content-Type", plan.ContentType)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call slack api %s: %w", plan.Path, err)
	}

	return resp, nil
}

type responseEnvelope struct {
	Kind     string         `json:"kind"`
	Method   string         `json:"method"`
	Response map[string]any `json:"response"`
}

type listEnvelope struct {
	Kind     string         `json:"kind"`
	Method   string         `json:"method"`
	List     listOutput     `json:"list"`
	Response map[string]any `json:"response"`
}

type listOutput struct {
	Field      string `json:"field"`
	Limit      int    `json:"limit"`
	Count      int    `json:"count"`
	NextCursor string `json:"next_cursor,omitempty"`
	Items      []any  `json:"items"`
}

func decodeAPIResponse(resp *http.Response, path string) (map[string]any, error) {
	var payload map[string]any

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()

	if err := decoder.Decode(&payload); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("slack api %s returned %s", path, resp.Status)
		}

		return nil, fmt.Errorf("decode slack api response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeAPIError(resp.StatusCode, resp.Status, path, payload)
	}

	if okValue, exists := payload["ok"]; exists {
		ok, okType := okValue.(bool)
		if okType && !ok {
			return nil, decodeAPIError(resp.StatusCode, resp.Status, path, payload)
		}
	}

	return payload, nil
}

func decodeAPIError(statusCode int, status, path string, payload map[string]any) error {
	message := stringValue(payload["error"])
	if message == "" {
		message = stringValue(payload["message"])
	}

	if message == "" {
		if statusCode >= http.StatusBadRequest {
			return fmt.Errorf("slack api %s returned %s", path, status)
		}

		return fmt.Errorf("slack api %s returned ok=false", path)
	}

	if needed := stringValue(payload["needed"]); needed != "" {
		return fmt.Errorf("slack api %s returned %s: %s (needed: %s)", path, statusOrOK(statusCode, status), message, needed)
	}

	return fmt.Errorf("slack api %s returned %s: %s", path, statusOrOK(statusCode, status), message)
}

func statusOrOK(statusCode int, status string) string {
	if statusCode == 0 || status == "" {
		return "ok=false"
	}

	if statusCode < http.StatusBadRequest {
		return "ok=false"
	}

	return status
}

func requestHTTPMethod(input Request) string {
	if input.HTTPMethod != "" {
		return input.HTTPMethod
	}

	switch input.Kind {
	case "create", "update", "delete":
		return http.MethodPost
	default:
		return http.MethodGet
	}
}

func normalizedArgs(input Request) map[string]any {
	args := cloneMap(input.Args)
	if args == nil {
		args = map[string]any{}
	}

	if input.Kind == "list" {
		args["limit"] = listLimit(input)

		if input.Cursor != "" {
			args["cursor"] = input.Cursor
		} else {
			delete(args, "cursor")
		}
	}

	return args
}

func encodeArgsToQuery(query url.Values, args map[string]any) error {
	for key, value := range args {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}

		switch typed := value.(type) {
		case string:
			query.Set(key, typed)
		case bool:
			query.Set(key, strconv.FormatBool(typed))
		case int:
			query.Set(key, strconv.Itoa(typed))
		case int8, int16, int32, int64:
			query.Set(key, fmt.Sprintf("%d", typed))
		case uint, uint8, uint16, uint32, uint64:
			query.Set(key, fmt.Sprintf("%d", typed))
		case float32, float64:
			query.Set(key, trimFloatString(fmt.Sprintf("%v", typed)))
		case json.Number:
			query.Set(key, typed.String())
		case []string:
			query.Set(key, strings.Join(typed, ","))
		default:
			raw, err := json.Marshal(typed)
			if err != nil {
				return fmt.Errorf("encode slack api query parameter %q: %w", key, err)
			}

			query.Set(key, string(raw))
		}
	}

	return nil
}

func inferListField(payload map[string]any) (string, error) {
	fields := make([]string, 0, len(payload))

	for key, value := range payload {
		if isReservedListField(key) {
			continue
		}

		if _, ok := value.([]any); ok {
			fields = append(fields, key)
		}
	}

	switch len(fields) {
	case 0:
		return "", errors.New("could not infer list field from Slack response; provide json input field \"list_field\"")
	case 1:
		return fields[0], nil
	default:
		slices.Sort(fields)
		return "", fmt.Errorf("could not infer list field from Slack response; candidates: %s; provide json input field \"list_field\"", strings.Join(fields, ", "))
	}
}

func isReservedListField(field string) bool {
	switch field {
	case "ok", "warning", "warnings", "response_metadata", "needed", "provided":
		return true
	default:
		return false
	}
}

func extractListItems(payload map[string]any, field string) ([]any, error) {
	value, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("slack response field %q not found; provide a valid json input field \"list_field\"", field)
	}

	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("slack response field %q is not an array", field)
	}

	return items, nil
}

func nextCursorFromPayload(payload map[string]any) string {
	metadata, ok := payload["response_metadata"].(map[string]any)
	if !ok {
		return ""
	}

	return strings.TrimSpace(stringValue(metadata["next_cursor"]))
}

func withNextCursor(payload map[string]any, nextCursor string) map[string]any {
	metadata, ok := payload["response_metadata"].(map[string]any)
	if !ok {
		metadata = map[string]any{}
	}

	metadata["next_cursor"] = nextCursor
	payload["response_metadata"] = metadata

	return payload
}

func listLimit(input Request) int {
	if input.Limit == nil {
		return DefaultListLimit
	}

	return *input.Limit
}

func extractOptionalIntArg(args map[string]any, key string) (int, bool, error) {
	if len(args) == 0 {
		return 0, false, nil
	}

	value, ok := args[key]
	if !ok || value == nil {
		return 0, false, nil
	}

	switch typed := value.(type) {
	case int:
		return typed, true, nil
	case int8:
		return int(typed), true, nil
	case int16:
		return int(typed), true, nil
	case int32:
		return int(typed), true, nil
	case int64:
		return int(typed), true, nil
	case uint:
		return int(typed), true, nil
	case uint8:
		return int(typed), true, nil
	case uint16:
		return int(typed), true, nil
	case uint32:
		return int(typed), true, nil
	case uint64:
		return int(typed), true, nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, true, errors.New("must be a whole number")
		}

		return int(typed), true, nil
	case float32:
		if typed != float32(int(typed)) {
			return 0, true, errors.New("must be a whole number")
		}

		return int(typed), true, nil
	case json.Number:
		number, err := typed.Int64()
		if err != nil {
			return 0, true, errors.New("must be an integer")
		}

		return int(number), true, nil
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, true, errors.New("must be an integer")
		}

		return number, true, nil
	default:
		return 0, true, errors.New("must be an integer")
	}
}

func extractOptionalStringArg(args map[string]any, key string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, nil
	}

	value, ok := args[key]
	if !ok || value == nil {
		return "", false, nil
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true, nil
	default:
		return "", true, errors.New("must be a string")
	}
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			output[key] = cloneMap(typed)
		case []any:
			output[key] = cloneSlice(typed)
		default:
			output[key] = typed
		}
	}

	return output
}

func cloneSlice(input []any) []any {
	if input == nil {
		return nil
	}

	output := make([]any, 0, len(input))
	for _, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			output = append(output, cloneMap(typed))
		case []any:
			output = append(output, cloneSlice(typed))
		default:
			output = append(output, typed)
		}
	}

	return output
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func trimFloatString(value string) string {
	if strings.HasSuffix(value, ".0") {
		return strings.TrimSuffix(value, ".0")
	}

	return value
}
