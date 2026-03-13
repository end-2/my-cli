package discord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/pkg/cliutil"
)

const (
	DefaultAPIBaseURL = "https://discord.com/api/v10/"
	DefaultUserAgent  = "DiscordBot (https://github.com/end-2/my-cli, 1.0) my-cli/my-discord"
	DefaultTimeout    = 15 * time.Second
	DefaultListLimit  = 100
	MaxListLimit      = 1000
	DefaultPageLimit  = 100
	MaxPageLimit      = 1000
	DefaultTokenType  = "Bot"
)

type Request struct {
	Kind        string         `json:"kind"`
	Path        string         `json:"path"`
	Query       map[string]any `json:"query,omitempty"`
	Body        map[string]any `json:"body,omitempty"`
	Limit       *int           `json:"limit,omitempty"`
	PageLimit   *int           `json:"page_limit,omitempty"`
	Before      string         `json:"before,omitempty"`
	After       string         `json:"after,omitempty"`
	ListField   string         `json:"list_field,omitempty"`
	CursorField string         `json:"cursor_field,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	HTTPMethod  string         `json:"http_method,omitempty"`
	BaseURL     string         `json:"base_url,omitempty"`
	Alias       string         `json:"alias,omitempty"`
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
	r.Path = strings.TrimSpace(r.Path)
	r.Before = strings.TrimSpace(r.Before)
	r.After = strings.TrimSpace(r.After)
	r.ListField = strings.TrimSpace(r.ListField)
	r.CursorField = strings.TrimSpace(r.CursorField)
	r.Reason = strings.TrimSpace(r.Reason)
	r.HTTPMethod = strings.ToUpper(strings.TrimSpace(r.HTTPMethod))
	r.BaseURL = strings.TrimSpace(r.BaseURL)
	r.Alias = strings.TrimSpace(r.Alias)

	if r.Path == "" {
		return errors.New("json input field \"path\" is required")
	}

	parsedPath, err := url.Parse(r.Path)
	if err != nil {
		return fmt.Errorf("parse json input field \"path\": %w", err)
	}

	if parsedPath.IsAbs() || parsedPath.Host != "" {
		return errors.New("json input field \"path\" must be relative to the Discord API base URL")
	}

	r.Path = "/" + strings.TrimLeft(r.Path, "/")

	if r.HTTPMethod != "" && !isSupportedHTTPMethod(r.HTTPMethod) {
		return errors.New("json input field \"http_method\" must be one of \"GET\", \"POST\", \"PUT\", \"PATCH\", or \"DELETE\"")
	}

	switch r.Kind {
	case "create", "read", "update", "delete":
		if r.Limit != nil {
			return fmt.Errorf("json input field \"limit\" is not allowed for kind %q", r.Kind)
		}

		if r.PageLimit != nil {
			return fmt.Errorf("json input field \"page_limit\" is not allowed for kind %q", r.Kind)
		}

		if r.Before != "" {
			return fmt.Errorf("json input field \"before\" is not allowed for kind %q", r.Kind)
		}

		if r.After != "" {
			return fmt.Errorf("json input field \"after\" is not allowed for kind %q", r.Kind)
		}

		if r.ListField != "" {
			return fmt.Errorf("json input field \"list_field\" is not allowed for kind %q", r.Kind)
		}

		if r.CursorField != "" {
			return fmt.Errorf("json input field \"cursor_field\" is not allowed for kind %q", r.Kind)
		}
	case "list":
		if r.HTTPMethod != "" && r.HTTPMethod != http.MethodGet {
			return errors.New("json input field \"http_method\" must be \"GET\" for kind \"list\"")
		}

		if r.Limit != nil && (*r.Limit < 1 || *r.Limit > MaxListLimit) {
			return fmt.Errorf("json input field \"limit\" must be between 1 and %d for kind %q", MaxListLimit, r.Kind)
		}

		if r.PageLimit != nil && (*r.PageLimit < 1 || *r.PageLimit > MaxPageLimit) {
			return fmt.Errorf("json input field \"page_limit\" must be between 1 and %d for kind %q", MaxPageLimit, r.Kind)
		}

		if r.Before != "" && r.After != "" {
			return errors.New("json input fields \"before\" and \"after\" cannot be used together")
		}

		if err := validateListQuery(r.Query); err != nil {
			return err
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

	if requestHTTPMethod(*r) == http.MethodGet && len(r.Body) > 0 {
		return errors.New("json input field \"body\" is not allowed for GET requests")
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

func validateListQuery(query map[string]any) error {
	for key := range query {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "limit", "before", "after":
			return fmt.Errorf("json input field \"query.%s\" is reserved for kind %q; use the top-level field instead", key, "list")
		}
	}

	return nil
}

func isSupportedHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type ClientConfig struct {
	BaseURL   string
	Token     string
	TokenType string
	Timeout   time.Duration
	UserAgent string
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL:   DefaultAPIBaseURL,
		TokenType: DefaultTokenType,
		Timeout:   DefaultTimeout,
		UserAgent: DefaultUserAgent,
	}
}

func NormalizeBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("parse discord api base url: empty value")
	}

	baseURL, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse discord api base url: %w", err)
	}

	switch {
	case baseURL.Path == "":
		baseURL.Path = "/"
	case !strings.HasSuffix(baseURL.Path, "/"):
		baseURL.Path += "/"
	}

	return baseURL.String(), nil
}

func NormalizeTokenType(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "bot":
		return DefaultTokenType, nil
	case "bearer":
		return "Bearer", nil
	default:
		return "", errors.New("must be either \"Bot\" or \"Bearer\"")
	}
}

func (c ClientConfig) withDefaults() (ClientConfig, error) {
	config := DefaultClientConfig()

	if value := strings.TrimSpace(c.BaseURL); value != "" {
		config.BaseURL = value
	}

	if value := strings.TrimSpace(c.Token); value != "" {
		config.Token = value
	}

	if value := strings.TrimSpace(c.TokenType); value != "" {
		tokenType, err := NormalizeTokenType(value)
		if err != nil {
			return ClientConfig{}, err
		}

		config.TokenType = tokenType
	}

	if value := strings.TrimSpace(c.UserAgent); value != "" {
		config.UserAgent = value
	}

	if c.Timeout > 0 {
		config.Timeout = c.Timeout
	}

	return config, nil
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
	tokenType  string
	userAgent  string
}

func NewClient(config ClientConfig, httpClient *http.Client) (*Client, error) {
	configWithDefaults, err := config.withDefaults()
	if err != nil {
		return nil, err
	}

	normalizedBaseURL, err := NormalizeBaseURL(configWithDefaults.BaseURL)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(normalizedBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse discord api base url: %w", err)
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: configWithDefaults.Timeout}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		token:      configWithDefaults.Token,
		tokenType:  configWithDefaults.TokenType,
		userAgent:  configWithDefaults.UserAgent,
	}, nil
}

func (c *Client) AuthMode() string {
	if c.token == "" {
		return "none"
	}

	return strings.ToLower(c.tokenType) + "_token"
}

type RequestPlan struct {
	URL            *url.URL
	Path           string
	HTTPMethod     string
	ContentType    string
	Body           []byte
	AuditLogReason string
}

func (c *Client) BuildRequest(input Request) (RequestPlan, error) {
	endpoint, err := c.baseURL.Parse(strings.TrimLeft(input.Path, "/"))
	if err != nil {
		return RequestPlan{}, fmt.Errorf("build discord api url: %w", err)
	}

	query := endpoint.Query()
	if err := encodeArgsToQuery(query, input.Query); err != nil {
		return RequestPlan{}, err
	}

	if input.Kind == "list" {
		query.Set("limit", strconv.Itoa(requestPageLimit(input)))
		if input.Before != "" {
			query.Set("before", input.Before)
		}
		if input.After != "" {
			query.Set("after", input.After)
		}
	}

	endpoint.RawQuery = query.Encode()

	plan := RequestPlan{
		URL:        endpoint,
		Path:       endpoint.RequestURI(),
		HTTPMethod: requestHTTPMethod(input),
	}

	if input.Reason != "" {
		plan.AuditLogReason = encodeAuditLogReason(input.Reason)
	}

	if plan.HTTPMethod != http.MethodGet && len(input.Body) > 0 {
		body, err := json.Marshal(input.Body)
		if err != nil {
			return RequestPlan{}, fmt.Errorf("encode discord api request body: %w", err)
		}

		plan.Body = body
		plan.ContentType = "application/json; charset=utf-8"
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
		Kind:       input.Kind,
		Path:       input.Path,
		HTTPMethod: plan.HTTPMethod,
		Response:   payload,
	}, nil
}

func (c *Client) executeList(input Request) (listEnvelope, error) {
	limit := listLimit(input)
	pageLimit := requestPageLimit(input)
	cursorField := listCursorField(input)
	current := input
	items := make([]any, 0, limit)

	var firstPayload any
	var listField string
	var nextCursor string

	for len(items) < limit {
		pageRequest := current
		pageRequest.PageLimit = intPtr(pageLimit)

		plan, err := c.BuildRequest(pageRequest)
		if err != nil {
			return listEnvelope{}, err
		}

		payload, err := c.executePlan(plan)
		if err != nil {
			return listEnvelope{}, err
		}

		pageItems, resolvedField, err := extractListItems(payload, current.ListField)
		if err != nil {
			return listEnvelope{}, err
		}

		if firstPayload == nil {
			firstPayload = cloneAny(payload)
		}

		if listField == "" {
			listField = resolvedField
		}

		rawPageCount := len(pageItems)
		remaining := limit - len(items)
		if len(pageItems) > remaining {
			pageItems = pageItems[:remaining]
		}

		items = append(items, cloneSlice(pageItems)...)

		if len(pageItems) == 0 {
			nextCursor = ""
			break
		}

		nextCursor, err = extractCursorValue(pageItems[len(pageItems)-1], cursorField)
		if err != nil {
			return listEnvelope{}, err
		}

		pageHasMore := listHasMore(payload, rawPageCount, pageLimit)
		moreAvailable := pageHasMore || rawPageCount > len(pageItems)

		if len(items) >= limit {
			if !moreAvailable {
				nextCursor = ""
			}
			break
		}

		if !moreAvailable {
			nextCursor = ""
			break
		}

		current.Before = ""
		current.After = ""
		if paginationMode(input) == "after" {
			current.After = nextCursor
		} else {
			current.Before = nextCursor
		}
	}

	if firstPayload == nil {
		firstPayload = []any{}
	}

	return listEnvelope{
		Kind:       "list",
		Path:       input.Path,
		HTTPMethod: http.MethodGet,
		List: listOutput{
			Field:       listField,
			CursorField: cursorField,
			Pagination:  paginationMode(input),
			Limit:       limit,
			Count:       len(items),
			NextCursor:  nextCursor,
			Items:       items,
		},
		Response: mergeListResponse(firstPayload, listField, items, nextCursor != ""),
	}, nil
}

func (c *Client) executePlan(plan RequestPlan) (any, error) {
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
		return nil, fmt.Errorf("create discord api request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if plan.ContentType != "" {
		req.Header.Set("Content-Type", plan.ContentType)
	}

	if plan.AuditLogReason != "" {
		req.Header.Set("X-Audit-Log-Reason", plan.AuditLogReason)
	}

	if c.token != "" {
		req.Header.Set("Authorization", c.tokenType+" "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call discord api %s: %w", plan.Path, err)
	}

	return resp, nil
}

type responseEnvelope struct {
	Kind       string `json:"kind"`
	Path       string `json:"path"`
	HTTPMethod string `json:"http_method"`
	Response   any    `json:"response"`
}

type listEnvelope struct {
	Kind       string     `json:"kind"`
	Path       string     `json:"path"`
	HTTPMethod string     `json:"http_method"`
	List       listOutput `json:"list"`
	Response   any        `json:"response"`
}

type listOutput struct {
	Field       string `json:"field,omitempty"`
	CursorField string `json:"cursor_field"`
	Pagination  string `json:"pagination"`
	Limit       int    `json:"limit"`
	Count       int    `json:"count"`
	NextCursor  string `json:"next_cursor,omitempty"`
	Items       []any  `json:"items"`
}

func decodeAPIResponse(resp *http.Response, path string) (any, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read discord api response: %w", err)
	}

	if len(bytes.TrimSpace(body)) == 0 {
		if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("discord api %s returned %s", path, resp.Status)
		}

		return map[string]any{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var payload any
	if err := decoder.Decode(&payload); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("discord api %s returned %s", path, resp.Status)
		}

		return nil, fmt.Errorf("decode discord api response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeAPIError(resp.StatusCode, resp.Status, path, payload)
	}

	return payload, nil
}

func decodeAPIError(statusCode int, status, path string, payload any) error {
	message := stringValue(nestedValue(payload, "message"))
	code := stringValueOrNumber(nestedValue(payload, "code"))

	if message == "" {
		return fmt.Errorf("discord api %s returned %s", path, status)
	}

	if code != "" {
		return fmt.Errorf("discord api %s returned %s: %s (code: %s)", path, statusOrUnknown(statusCode, status), message, code)
	}

	return fmt.Errorf("discord api %s returned %s: %s", path, statusOrUnknown(statusCode, status), message)
}

func statusOrUnknown(statusCode int, status string) string {
	if statusCode == 0 || status == "" {
		return "error"
	}

	return status
}

func requestHTTPMethod(input Request) string {
	if input.HTTPMethod != "" {
		return input.HTTPMethod
	}

	switch input.Kind {
	case "create":
		return http.MethodPost
	case "update":
		return http.MethodPatch
	case "delete":
		return http.MethodDelete
	default:
		return http.MethodGet
	}
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
			for _, item := range typed {
				query.Add(key, item)
			}
		default:
			raw, err := json.Marshal(typed)
			if err != nil {
				return fmt.Errorf("encode discord api query parameter %q: %w", key, err)
			}

			query.Set(key, string(raw))
		}
	}

	return nil
}

func listLimit(input Request) int {
	if input.Limit == nil {
		return DefaultListLimit
	}

	return *input.Limit
}

func requestPageLimit(input Request) int {
	if input.PageLimit != nil {
		return *input.PageLimit
	}

	if input.Limit != nil && *input.Limit < DefaultPageLimit {
		return *input.Limit
	}

	return DefaultPageLimit
}

func paginationMode(input Request) string {
	if input.After != "" {
		return "after"
	}

	return "before"
}

func listCursorField(input Request) string {
	if input.CursorField != "" {
		return input.CursorField
	}

	return "id"
}

func extractListItems(payload any, requestedField string) ([]any, string, error) {
	switch typed := payload.(type) {
	case []any:
		return typed, "", nil
	case map[string]any:
		field := requestedField
		if field == "" {
			inferredField, err := inferListField(typed)
			if err != nil {
				return nil, "", err
			}

			field = inferredField
		}

		value, ok := typed[field]
		if !ok {
			return nil, "", fmt.Errorf("discord response field %q not found; provide a valid json input field \"list_field\"", field)
		}

		items, ok := value.([]any)
		if !ok {
			return nil, "", fmt.Errorf("discord response field %q is not an array", field)
		}

		return items, field, nil
	default:
		return nil, "", errors.New("discord list responses must be either an array or an object that contains an array field")
	}
}

func inferListField(payload map[string]any) (string, error) {
	fields := make([]string, 0, len(payload))

	for key, value := range payload {
		if _, ok := value.([]any); ok {
			fields = append(fields, key)
		}
	}

	switch len(fields) {
	case 0:
		return "", errors.New("could not infer list field from Discord response; provide json input field \"list_field\"")
	case 1:
		return fields[0], nil
	default:
		slices.Sort(fields)
		return "", fmt.Errorf("could not infer list field from Discord response; candidates: %s; provide json input field \"list_field\"", strings.Join(fields, ", "))
	}
}

func listHasMore(payload any, rawPageCount, pageLimit int) bool {
	if rawPageCount == 0 {
		return false
	}

	if typed, ok := payload.(map[string]any); ok {
		if value, exists := typed["has_more"]; exists {
			hasMore, ok := value.(bool)
			if ok {
				return hasMore
			}
		}
	}

	return rawPageCount >= pageLimit
}

func extractCursorValue(item any, field string) (string, error) {
	current := item

	for _, segment := range strings.Split(field, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return "", errors.New("json input field \"cursor_field\" must not contain empty path segments")
		}

		obj, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("discord list cursor field %q is not reachable on the current item", field)
		}

		next, ok := obj[segment]
		if !ok {
			return "", fmt.Errorf("discord list cursor field %q was not found on the current item", field)
		}

		current = next
	}

	value := stringValueOrNumber(current)
	if value == "" {
		return "", fmt.Errorf("discord list cursor field %q must resolve to a string or number", field)
	}

	return value, nil
}

func mergeListResponse(payload any, field string, items []any, hasMore bool) any {
	switch typed := cloneAny(payload).(type) {
	case []any:
		return cloneSlice(items)
	case map[string]any:
		typed[field] = cloneSlice(items)
		if _, exists := typed["has_more"]; exists {
			typed["has_more"] = hasMore
		}
		return typed
	default:
		return cloneSlice(items)
	}
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		return cloneSlice(typed)
	default:
		return typed
	}
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = cloneAny(value)
	}

	return output
}

func cloneSlice(input []any) []any {
	if input == nil {
		return nil
	}

	output := make([]any, 0, len(input))
	for _, value := range input {
		output = append(output, cloneAny(value))
	}

	return output
}

func nestedValue(payload any, path ...string) any {
	current := payload
	for _, segment := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}

		next, ok := obj[segment]
		if !ok {
			return nil
		}

		current = next
	}

	return current
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func stringValueOrNumber(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case int:
		return strconv.Itoa(typed)
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", typed)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", typed)
	case float32, float64:
		return trimFloatString(fmt.Sprintf("%v", typed))
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

func encodeAuditLogReason(reason string) string {
	return strings.ReplaceAll(url.QueryEscape(strings.TrimSpace(reason)), "+", "%20")
}

func intPtr(value int) *int {
	return &value
}
