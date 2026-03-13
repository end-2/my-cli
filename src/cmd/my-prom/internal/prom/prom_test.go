package prom

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseRequestNormalizesAliases(t *testing.T) {
	request, err := ParseRequest(`{
		"kind":"range",
		"query":"up",
		"start":"2026-03-13T00:00:00Z",
		"end":"2026-03-13T01:00:00Z",
		"step":"5m"
	}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.Kind != "query_range" {
		t.Fatalf("Kind = %q, want %q", request.Kind, "query_range")
	}
}

func TestParseRequestRejectsInvalidFieldsForQuery(t *testing.T) {
	_, err := ParseRequest(`{"kind":"query","query":"up","matchers":["up"]}`)
	if err == nil {
		t.Fatal("ParseRequest returned nil error, want validation error")
	}

	if !strings.Contains(err.Error(), `"matchers" is not allowed`) {
		t.Fatalf("error = %v, want matchers validation error", err)
	}
}

func TestBuildRequestEncodesPostBodyForRangeQuery(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "https://prom.example.com/"}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	limit := 2
	plan, err := client.BuildRequest(Request{
		Kind:       "query_range",
		Query:      `rate(http_requests_total[5m])`,
		Start:      "2026-03-13T00:00:00Z",
		End:        "2026-03-13T01:00:00Z",
		Step:       "5m",
		Timeout:    "30s",
		Limit:      &limit,
		HTTPMethod: http.MethodPost,
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if plan.URL.String() != "https://prom.example.com/api/v1/query_range" {
		t.Fatalf("URL = %q, want POST endpoint without query string", plan.URL.String())
	}

	if plan.ContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("ContentType = %q, want form encoding", plan.ContentType)
	}

	values, err := url.ParseQuery(string(plan.Body))
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}

	if got := values.Get("query"); got != `rate(http_requests_total[5m])` {
		t.Fatalf("query = %q, want range query", got)
	}

	if got := values.Get("limit"); got != "2" {
		t.Fatalf("limit = %q, want %q", got, "2")
	}
}

func TestClientExecuteFetchesLabelNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/labels" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/labels")
		}

		if got := r.URL.Query().Get("start"); got != "2026-03-13T00:00:00Z" {
			t.Fatalf("start = %q, want %q", got, "2026-03-13T00:00:00Z")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"status":"success",
			"data":["__name__","instance","job"],
			"warnings":["partial response"]
		}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		BaseURL: server.URL,
		Timeout: time.Second,
	}, server.Client())
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	limit := 2
	request := Request{
		Kind:  "label_names",
		Start: "2026-03-13T00:00:00Z",
		Limit: &limit,
	}

	plan, err := client.BuildRequest(request)
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	output, err := client.Execute(plan, request)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	payload := output.(labelNamesEnvelope)
	if payload.Count != 2 {
		t.Fatalf("Count = %d, want %d", payload.Count, 2)
	}

	if len(payload.Labels) != 2 {
		t.Fatalf("len(Labels) = %d, want %d", len(payload.Labels), 2)
	}

	if payload.Warnings[0] != "partial response" {
		t.Fatalf("Warnings = %v, want partial response", payload.Warnings)
	}
}

func TestDecodeAPIErrorReturnsPrometheusMessage(t *testing.T) {
	response := httptest.NewRecorder()
	response.WriteHeader(http.StatusBadRequest)
	_, _ = io.WriteString(response, `{
		"status":"error",
		"errorType":"bad_data",
		"error":"invalid parameter \"query\": 1:1: parse error"
	}`)

	_, err := decodeAPIEnvelope(response.Result(), "/api/v1/query")
	if err == nil {
		t.Fatal("decodeAPIEnvelope returned nil error, want api error")
	}

	if !strings.Contains(err.Error(), "bad_data: invalid parameter") {
		t.Fatalf("error = %v, want prometheus error details", err)
	}
}

func TestNormalizeSamplePointFormatsTimestamp(t *testing.T) {
	point, err := normalizeSamplePoint([]any{json.Number("1710283200.5"), "1"})
	if err != nil {
		t.Fatalf("normalizeSamplePoint returned error: %v", err)
	}

	if point.Timestamp != "2024-03-12T22:40:00.5Z" {
		t.Fatalf("Timestamp = %q, want RFC3339 time", point.Timestamp)
	}

	if point.TimestampUnix != 1710283200.5 {
		t.Fatalf("TimestampUnix = %v, want %v", point.TimestampUnix, 1710283200.5)
	}
}
