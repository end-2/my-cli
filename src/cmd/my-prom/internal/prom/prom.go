package prom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/pkg/cliutil"
)

const (
	DefaultAPIBaseURL = "http://localhost:9090/"
	DefaultUserAgent  = "my-cli/my-prom"
	DefaultTimeout    = 15 * time.Second
)

var labelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Request struct {
	Kind       string   `json:"kind"`
	Query      string   `json:"query,omitempty"`
	Time       string   `json:"time,omitempty"`
	Start      string   `json:"start,omitempty"`
	End        string   `json:"end,omitempty"`
	Step       string   `json:"step,omitempty"`
	Timeout    string   `json:"timeout,omitempty"`
	Limit      *int     `json:"limit,omitempty"`
	Matchers   []string `json:"matchers,omitempty"`
	Label      string   `json:"label,omitempty"`
	HTTPMethod string   `json:"http_method,omitempty"`
	BaseURL    string   `json:"base_url,omitempty"`
	Alias      string   `json:"alias,omitempty"`
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
	r.Query = strings.TrimSpace(r.Query)
	r.Time = strings.TrimSpace(r.Time)
	r.Start = strings.TrimSpace(r.Start)
	r.End = strings.TrimSpace(r.End)
	r.Step = strings.TrimSpace(r.Step)
	r.Timeout = strings.TrimSpace(r.Timeout)
	r.Label = strings.TrimSpace(r.Label)
	r.HTTPMethod = strings.ToUpper(strings.TrimSpace(r.HTTPMethod))
	r.BaseURL = strings.TrimSpace(r.BaseURL)
	r.Alias = strings.TrimSpace(r.Alias)
	r.Matchers = normalizeMatchers(r.Matchers)

	if r.Limit != nil && *r.Limit < 0 {
		return errors.New("json input field \"limit\" must be greater than or equal to zero")
	}

	if r.HTTPMethod != "" && r.HTTPMethod != http.MethodGet && r.HTTPMethod != http.MethodPost {
		return errors.New("json input field \"http_method\" must be either \"GET\" or \"POST\"")
	}

	switch r.Kind {
	case "query":
		if r.Query == "" {
			return errors.New("json input field \"query\" is required for kind \"query\"")
		}

		if r.Start != "" {
			return errors.New("json input field \"start\" is not allowed for kind \"query\"")
		}

		if r.End != "" {
			return errors.New("json input field \"end\" is not allowed for kind \"query\"")
		}

		if r.Step != "" {
			return errors.New("json input field \"step\" is not allowed for kind \"query\"")
		}

		if len(r.Matchers) > 0 {
			return errors.New("json input field \"matchers\" is not allowed for kind \"query\"")
		}

		if r.Label != "" {
			return errors.New("json input field \"label\" is not allowed for kind \"query\"")
		}
	case "query_range":
		if r.Query == "" {
			return errors.New("json input field \"query\" is required for kind \"query_range\"")
		}

		if r.Start == "" {
			return errors.New("json input field \"start\" is required for kind \"query_range\"")
		}

		if r.End == "" {
			return errors.New("json input field \"end\" is required for kind \"query_range\"")
		}

		if r.Step == "" {
			return errors.New("json input field \"step\" is required for kind \"query_range\"")
		}

		if r.Time != "" {
			return errors.New("json input field \"time\" is not allowed for kind \"query_range\"")
		}

		if len(r.Matchers) > 0 {
			return errors.New("json input field \"matchers\" is not allowed for kind \"query_range\"")
		}

		if r.Label != "" {
			return errors.New("json input field \"label\" is not allowed for kind \"query_range\"")
		}
	case "series":
		if len(r.Matchers) == 0 {
			return errors.New("json input field \"matchers\" must contain at least one matcher for kind \"series\"")
		}

		if r.Query != "" {
			return errors.New("json input field \"query\" is not allowed for kind \"series\"")
		}

		if r.Time != "" {
			return errors.New("json input field \"time\" is not allowed for kind \"series\"")
		}

		if r.Step != "" {
			return errors.New("json input field \"step\" is not allowed for kind \"series\"")
		}

		if r.Timeout != "" {
			return errors.New("json input field \"timeout\" is not allowed for kind \"series\"")
		}

		if r.Label != "" {
			return errors.New("json input field \"label\" is not allowed for kind \"series\"")
		}
	case "label_names":
		if r.Query != "" {
			return errors.New("json input field \"query\" is not allowed for kind \"label_names\"")
		}

		if r.Time != "" {
			return errors.New("json input field \"time\" is not allowed for kind \"label_names\"")
		}

		if r.Step != "" {
			return errors.New("json input field \"step\" is not allowed for kind \"label_names\"")
		}

		if r.Timeout != "" {
			return errors.New("json input field \"timeout\" is not allowed for kind \"label_names\"")
		}

		if r.Label != "" {
			return errors.New("json input field \"label\" is not allowed for kind \"label_names\"")
		}
	case "label_values":
		if r.Label == "" {
			return errors.New("json input field \"label\" is required for kind \"label_values\"")
		}

		if !labelNamePattern.MatchString(r.Label) {
			return errors.New("json input field \"label\" must be a Prometheus label name like \"job\" or \"__name__\"")
		}

		if r.Query != "" {
			return errors.New("json input field \"query\" is not allowed for kind \"label_values\"")
		}

		if r.Time != "" {
			return errors.New("json input field \"time\" is not allowed for kind \"label_values\"")
		}

		if r.Step != "" {
			return errors.New("json input field \"step\" is not allowed for kind \"label_values\"")
		}

		if r.Timeout != "" {
			return errors.New("json input field \"timeout\" is not allowed for kind \"label_values\"")
		}

		if r.HTTPMethod == http.MethodPost {
			return errors.New("json input field \"http_method\" must be \"GET\" for kind \"label_values\"")
		}
	default:
		return fmt.Errorf(
			"json input field \"kind\" must be one of %q, %q, %q, %q, or %q",
			"query",
			"query_range",
			"series",
			"label_names",
			"label_values",
		)
	}

	return nil
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "instant", "instant-query", "instant_query":
		return "query"
	case "range", "range-query", "range_query":
		return "query_range"
	case "label-names", "labels":
		return "label_names"
	case "label-values":
		return "label_values"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func normalizeMatchers(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	matchers := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		matchers = append(matchers, trimmed)
	}

	if len(matchers) == 0 {
		return nil
	}

	return matchers
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
		return "", errors.New("parse prometheus api base url: empty value")
	}

	baseURL, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse prometheus api base url: %w", err)
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
		return nil, fmt.Errorf("parse prometheus api base url: %w", err)
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
	path, err := requestPath(input)
	if err != nil {
		return RequestPlan{}, err
	}

	endpoint, err := c.baseURL.Parse(path)
	if err != nil {
		return RequestPlan{}, fmt.Errorf("build prometheus api url: %w", err)
	}

	params := encodeRequestParams(input)
	httpMethod := requestHTTPMethod(input)
	plan := RequestPlan{
		URL:        endpoint,
		Path:       endpoint.RequestURI(),
		HTTPMethod: httpMethod,
	}

	switch httpMethod {
	case http.MethodGet:
		endpoint.RawQuery = params.Encode()
		plan.URL = endpoint
		plan.Path = endpoint.RequestURI()
	case http.MethodPost:
		if input.Kind == "label_values" {
			return RequestPlan{}, fmt.Errorf("unsupported http method %q for kind %q", httpMethod, input.Kind)
		}

		plan.Body = []byte(params.Encode())
		plan.ContentType = "application/x-www-form-urlencoded"
	default:
		return RequestPlan{}, fmt.Errorf("unsupported http method %q", httpMethod)
	}

	return plan, nil
}

func requestPath(input Request) (string, error) {
	switch input.Kind {
	case "query":
		return "api/v1/query", nil
	case "query_range":
		return "api/v1/query_range", nil
	case "series":
		return "api/v1/series", nil
	case "label_names":
		return "api/v1/labels", nil
	case "label_values":
		return fmt.Sprintf("api/v1/label/%s/values", url.PathEscape(input.Label)), nil
	default:
		return "", fmt.Errorf("unsupported kind %q", input.Kind)
	}
}

func requestHTTPMethod(input Request) string {
	if input.HTTPMethod != "" {
		return input.HTTPMethod
	}

	return http.MethodGet
}

func encodeRequestParams(input Request) url.Values {
	values := url.Values{}

	switch input.Kind {
	case "query":
		values.Set("query", input.Query)
		setOptional(values, "time", input.Time)
		setOptional(values, "timeout", input.Timeout)
		setOptionalInt(values, "limit", input.Limit)
	case "query_range":
		values.Set("query", input.Query)
		values.Set("start", input.Start)
		values.Set("end", input.End)
		values.Set("step", input.Step)
		setOptional(values, "timeout", input.Timeout)
		setOptionalInt(values, "limit", input.Limit)
	case "series", "label_names", "label_values":
		for _, matcher := range input.Matchers {
			values.Add("match[]", matcher)
		}

		setOptional(values, "start", input.Start)
		setOptional(values, "end", input.End)
		setOptionalInt(values, "limit", input.Limit)
	}

	return values
}

func setOptional(values url.Values, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	values.Set(key, value)
}

func setOptionalInt(values url.Values, key string, value *int) {
	if value == nil {
		return
	}

	values.Set(key, strconv.Itoa(*value))
}

func (c *Client) Execute(plan RequestPlan, input Request) (any, error) {
	envelope, err := c.executePlan(plan)
	if err != nil {
		return nil, err
	}

	switch input.Kind {
	case "query", "query_range":
		return decodeQueryEnvelope(input, envelope)
	case "series":
		return decodeSeriesEnvelope(input, envelope)
	case "label_names":
		return decodeLabelNamesEnvelope(input, envelope)
	case "label_values":
		return decodeLabelValuesEnvelope(input, envelope)
	default:
		return nil, fmt.Errorf("unsupported kind %q", input.Kind)
	}
}

func (c *Client) executePlan(plan RequestPlan) (apiEnvelope, error) {
	resp, err := c.do(plan)
	if err != nil {
		return apiEnvelope{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return decodeAPIEnvelope(resp, plan.Path)
}

func (c *Client) do(plan RequestPlan) (*http.Response, error) {
	var body io.Reader
	if len(plan.Body) > 0 {
		body = bytes.NewReader(plan.Body)
	}

	req, err := http.NewRequest(plan.HTTPMethod, plan.URL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create prometheus api request: %w", err)
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
		return nil, fmt.Errorf("call prometheus api %s: %w", plan.Path, err)
	}

	return resp, nil
}

type apiEnvelope struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings"`
	Infos     []string        `json:"infos"`
}

func decodeAPIEnvelope(resp *http.Response, path string) (apiEnvelope, error) {
	var envelope apiEnvelope
	decoder := json.NewDecoder(resp.Body)

	if err := decoder.Decode(&envelope); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return apiEnvelope{}, fmt.Errorf("prometheus api %s returned %s", path, resp.Status)
		}

		return apiEnvelope{}, fmt.Errorf("decode prometheus api response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest || strings.EqualFold(envelope.Status, "error") {
		return apiEnvelope{}, decodeAPIError(resp.StatusCode, resp.Status, path, envelope)
	}

	return envelope, nil
}

func decodeAPIError(statusCode int, status, path string, envelope apiEnvelope) error {
	if strings.TrimSpace(envelope.Error) == "" {
		if statusCode >= http.StatusBadRequest {
			return fmt.Errorf("prometheus api %s returned %s", path, status)
		}

		return fmt.Errorf("prometheus api %s returned status=error", path)
	}

	message := envelope.Error
	if strings.TrimSpace(envelope.ErrorType) != "" {
		message = envelope.ErrorType + ": " + envelope.Error
	}

	return fmt.Errorf("prometheus api %s returned %s: %s", path, statusOrError(statusCode, status), message)
}

func statusOrError(statusCode int, status string) string {
	if statusCode == 0 || status == "" || statusCode < http.StatusBadRequest {
		return "status=error"
	}

	return status
}

type queryEnvelope struct {
	Kind       string   `json:"kind"`
	Query      string   `json:"query"`
	Time       string   `json:"time,omitempty"`
	Start      string   `json:"start,omitempty"`
	End        string   `json:"end,omitempty"`
	Step       string   `json:"step,omitempty"`
	Timeout    string   `json:"timeout,omitempty"`
	Limit      *int     `json:"limit,omitempty"`
	ResultType string   `json:"result_type"`
	Count      int      `json:"count"`
	Result     any      `json:"result"`
	Warnings   []string `json:"warnings,omitempty"`
	Infos      []string `json:"infos,omitempty"`
}

type seriesEnvelope struct {
	Kind     string              `json:"kind"`
	Matchers []string            `json:"matchers"`
	Start    string              `json:"start,omitempty"`
	End      string              `json:"end,omitempty"`
	Limit    *int                `json:"limit,omitempty"`
	Count    int                 `json:"count"`
	Series   []map[string]string `json:"series"`
	Warnings []string            `json:"warnings,omitempty"`
	Infos    []string            `json:"infos,omitempty"`
}

type labelNamesEnvelope struct {
	Kind     string   `json:"kind"`
	Matchers []string `json:"matchers,omitempty"`
	Start    string   `json:"start,omitempty"`
	End      string   `json:"end,omitempty"`
	Limit    *int     `json:"limit,omitempty"`
	Count    int      `json:"count"`
	Labels   []string `json:"labels"`
	Warnings []string `json:"warnings,omitempty"`
	Infos    []string `json:"infos,omitempty"`
}

type labelValuesEnvelope struct {
	Kind     string   `json:"kind"`
	Label    string   `json:"label"`
	Matchers []string `json:"matchers,omitempty"`
	Start    string   `json:"start,omitempty"`
	End      string   `json:"end,omitempty"`
	Limit    *int     `json:"limit,omitempty"`
	Count    int      `json:"count"`
	Values   []string `json:"values"`
	Warnings []string `json:"warnings,omitempty"`
	Infos    []string `json:"infos,omitempty"`
}

type queryAPIData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

type instantSeriesAPI struct {
	Metric    map[string]string `json:"metric"`
	Value     []any             `json:"value,omitempty"`
	Histogram []any             `json:"histogram,omitempty"`
}

type rangeSeriesAPI struct {
	Metric     map[string]string `json:"metric"`
	Values     [][]any           `json:"values,omitempty"`
	Histograms [][]any           `json:"histograms,omitempty"`
}

type instantSeriesOutput struct {
	Metric    map[string]string     `json:"metric,omitempty"`
	Value     *samplePointOutput    `json:"value,omitempty"`
	Histogram *histogramPointOutput `json:"histogram,omitempty"`
}

type rangeSeriesOutput struct {
	Metric     map[string]string      `json:"metric,omitempty"`
	Values     []samplePointOutput    `json:"values,omitempty"`
	Histograms []histogramPointOutput `json:"histograms,omitempty"`
}

type samplePointOutput struct {
	Timestamp     string  `json:"timestamp"`
	TimestampUnix float64 `json:"timestamp_unix"`
	Value         string  `json:"value"`
}

type histogramPointOutput struct {
	Timestamp     string  `json:"timestamp"`
	TimestampUnix float64 `json:"timestamp_unix"`
	Histogram     any     `json:"histogram"`
}

func decodeQueryEnvelope(input Request, envelope apiEnvelope) (queryEnvelope, error) {
	data, err := decodeUseNumber[queryAPIData](envelope.Data)
	if err != nil {
		return queryEnvelope{}, fmt.Errorf("decode prometheus query data: %w", err)
	}

	output := queryEnvelope{
		Kind:       input.Kind,
		Query:      input.Query,
		Time:       input.Time,
		Start:      input.Start,
		End:        input.End,
		Step:       input.Step,
		Timeout:    input.Timeout,
		Limit:      input.Limit,
		ResultType: data.ResultType,
		Warnings:   envelope.Warnings,
		Infos:      envelope.Infos,
	}

	switch data.ResultType {
	case "vector":
		items, err := decodeUseNumber[[]instantSeriesAPI](data.Result)
		if err != nil {
			return queryEnvelope{}, fmt.Errorf("decode prometheus vector result: %w", err)
		}

		result, err := normalizeInstantSeries(items, input.Limit)
		if err != nil {
			return queryEnvelope{}, err
		}

		output.Count = len(result)
		output.Result = result
	case "matrix":
		items, err := decodeUseNumber[[]rangeSeriesAPI](data.Result)
		if err != nil {
			return queryEnvelope{}, fmt.Errorf("decode prometheus matrix result: %w", err)
		}

		result, err := normalizeRangeSeries(items, input.Limit)
		if err != nil {
			return queryEnvelope{}, err
		}

		output.Count = len(result)
		output.Result = result
	case "scalar", "string":
		item, err := decodeUseNumber[[]any](data.Result)
		if err != nil {
			return queryEnvelope{}, fmt.Errorf("decode prometheus %s result: %w", data.ResultType, err)
		}

		point, err := normalizeSamplePoint(item)
		if err != nil {
			return queryEnvelope{}, fmt.Errorf("decode prometheus %s result: %w", data.ResultType, err)
		}

		output.Count = 1
		output.Result = point
	default:
		return queryEnvelope{}, fmt.Errorf("unsupported prometheus result type %q", data.ResultType)
	}

	return output, nil
}

func decodeSeriesEnvelope(input Request, envelope apiEnvelope) (seriesEnvelope, error) {
	items, err := decodeUseNumber[[]map[string]string](envelope.Data)
	if err != nil {
		return seriesEnvelope{}, fmt.Errorf("decode prometheus series response: %w", err)
	}

	items = trimSlice(items, input.Limit)

	return seriesEnvelope{
		Kind:     "series",
		Matchers: input.Matchers,
		Start:    input.Start,
		End:      input.End,
		Limit:    input.Limit,
		Count:    len(items),
		Series:   items,
		Warnings: envelope.Warnings,
		Infos:    envelope.Infos,
	}, nil
}

func decodeLabelNamesEnvelope(input Request, envelope apiEnvelope) (labelNamesEnvelope, error) {
	items, err := decodeUseNumber[[]string](envelope.Data)
	if err != nil {
		return labelNamesEnvelope{}, fmt.Errorf("decode prometheus label names response: %w", err)
	}

	items = trimSlice(items, input.Limit)

	return labelNamesEnvelope{
		Kind:     "label_names",
		Matchers: input.Matchers,
		Start:    input.Start,
		End:      input.End,
		Limit:    input.Limit,
		Count:    len(items),
		Labels:   items,
		Warnings: envelope.Warnings,
		Infos:    envelope.Infos,
	}, nil
}

func decodeLabelValuesEnvelope(input Request, envelope apiEnvelope) (labelValuesEnvelope, error) {
	items, err := decodeUseNumber[[]string](envelope.Data)
	if err != nil {
		return labelValuesEnvelope{}, fmt.Errorf("decode prometheus label values response: %w", err)
	}

	items = trimSlice(items, input.Limit)

	return labelValuesEnvelope{
		Kind:     "label_values",
		Label:    input.Label,
		Matchers: input.Matchers,
		Start:    input.Start,
		End:      input.End,
		Limit:    input.Limit,
		Count:    len(items),
		Values:   items,
		Warnings: envelope.Warnings,
		Infos:    envelope.Infos,
	}, nil
}

func normalizeInstantSeries(items []instantSeriesAPI, limit *int) ([]instantSeriesOutput, error) {
	items = trimSlice(items, limit)
	result := make([]instantSeriesOutput, 0, len(items))

	for _, item := range items {
		output := instantSeriesOutput{
			Metric: item.Metric,
		}

		if len(item.Value) > 0 {
			point, err := normalizeSamplePoint(item.Value)
			if err != nil {
				return nil, err
			}

			output.Value = &point
		}

		if len(item.Histogram) > 0 {
			point, err := normalizeHistogramPoint(item.Histogram)
			if err != nil {
				return nil, err
			}

			output.Histogram = &point
		}

		result = append(result, output)
	}

	return result, nil
}

func normalizeRangeSeries(items []rangeSeriesAPI, limit *int) ([]rangeSeriesOutput, error) {
	items = trimSlice(items, limit)
	result := make([]rangeSeriesOutput, 0, len(items))

	for _, item := range items {
		output := rangeSeriesOutput{
			Metric: item.Metric,
		}

		if len(item.Values) > 0 {
			values := make([]samplePointOutput, 0, len(item.Values))
			for _, value := range item.Values {
				point, err := normalizeSamplePoint(value)
				if err != nil {
					return nil, err
				}

				values = append(values, point)
			}

			output.Values = values
		}

		if len(item.Histograms) > 0 {
			values := make([]histogramPointOutput, 0, len(item.Histograms))
			for _, value := range item.Histograms {
				point, err := normalizeHistogramPoint(value)
				if err != nil {
					return nil, err
				}

				values = append(values, point)
			}

			output.Histograms = values
		}

		result = append(result, output)
	}

	return result, nil
}

func normalizeSamplePoint(raw []any) (samplePointOutput, error) {
	if len(raw) != 2 {
		return samplePointOutput{}, errors.New("sample value must contain exactly two items")
	}

	timestamp, err := parseTimestamp(raw[0])
	if err != nil {
		return samplePointOutput{}, err
	}

	return samplePointOutput{
		Timestamp:     timestamp.RFC3339,
		TimestampUnix: timestamp.Unix,
		Value:         stringifyScalar(raw[1]),
	}, nil
}

func normalizeHistogramPoint(raw []any) (histogramPointOutput, error) {
	if len(raw) != 2 {
		return histogramPointOutput{}, errors.New("histogram value must contain exactly two items")
	}

	timestamp, err := parseTimestamp(raw[0])
	if err != nil {
		return histogramPointOutput{}, err
	}

	return histogramPointOutput{
		Timestamp:     timestamp.RFC3339,
		TimestampUnix: timestamp.Unix,
		Histogram:     raw[1],
	}, nil
}

type parsedTimestamp struct {
	Unix    float64
	RFC3339 string
}

func parseTimestamp(raw any) (parsedTimestamp, error) {
	value, err := parseFloat(raw)
	if err != nil {
		return parsedTimestamp{}, fmt.Errorf("parse sample timestamp: %w", err)
	}

	sec, frac := math.Modf(value)
	nsec := int64(math.Round(frac * float64(time.Second)))
	if nsec >= int64(time.Second) {
		sec++
		nsec -= int64(time.Second)
	}

	instant := time.Unix(int64(sec), nsec).UTC()

	return parsedTimestamp{
		Unix:    value,
		RFC3339: instant.Format(time.RFC3339Nano),
	}, nil
}

func parseFloat(raw any) (float64, error) {
	switch value := raw.(type) {
	case json.Number:
		return value.Float64()
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case string:
		return strconv.ParseFloat(value, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric value type %T", raw)
	}
}

func stringifyScalar(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	default:
		return fmt.Sprintf("%v", raw)
	}
}

func trimSlice[T any](items []T, limit *int) []T {
	if limit == nil || *limit == 0 || len(items) <= *limit {
		return items
	}

	return items[:*limit]
}

func decodeUseNumber[T any](raw json.RawMessage) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	if err := decoder.Decode(&value); err != nil {
		return value, err
	}

	return value, nil
}
