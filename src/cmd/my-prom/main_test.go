package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	promapp "github.com/end-2/my-cli/src/cmd/my-prom/internal/app"
	"github.com/end-2/my-cli/src/cmd/my-prom/internal/prom"
)

func TestRootCommandPrintsVersion(t *testing.T) {
	originalVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() {
		Version = originalVersion
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(strings.NewReader(""), &stdout, &stderr, []string{"--version"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got != "1.2.3\n" {
		t.Fatalf("stdout = %q, want %q", got, "1.2.3\n")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(strings.NewReader(""), &stdout, &stderr, []string{"--help"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	output := stdout.String()
	for _, expected := range []string{
		"Query Prometheus HTTP API endpoints with one JSON request",
		"--dry-run",
		"--version",
		"my-prom.yaml",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("stdout = %q, want to contain %q", output, expected)
		}
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsDryRunPlanUsingConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{"--dry-run", `{"kind":"query","query":"up","time":"2026-03-13T12:00:00Z"}`},
		promapp.Dependencies{
			LoadConfig: func(request prom.Request) (prom.ClientConfig, error) {
				return prom.ClientConfig{
					BaseURL: "https://prom.example.com/",
					Token:   "configured-token",
					Timeout: time.Second,
				}, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if got := output["mode"]; got != "dry-run" {
		t.Fatalf("mode = %v, want %q", got, "dry-run")
	}

	httpOutput, ok := output["http"].(map[string]any)
	if !ok {
		t.Fatalf("http = %T, want map[string]any", output["http"])
	}

	if got := httpOutput["url"]; got != "https://prom.example.com/api/v1/query?query=up&time=2026-03-13T12%3A00%3A00Z" {
		t.Fatalf("http.url = %v, want custom configured URL", got)
	}

	if got := httpOutput["auth"]; got != "token" {
		t.Fatalf("http.auth = %v, want %q", got, "token")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesInstantQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/api/v1/query" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/query")
		}

		if got := r.URL.Query().Get("query"); got != "up" {
			t.Fatalf("query = %q, want %q", got, "up")
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer test-token")
		}

		if got := r.Header.Get("User-Agent"); got != prom.DefaultUserAgent {
			t.Fatalf("User-Agent = %q, want %q", got, prom.DefaultUserAgent)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":{
				"resultType":"vector",
				"result":[
					{
						"metric":{"__name__":"up","job":"node","instance":"node-1"},
						"value":[1710283200.5,"1"]
					}
				]
			},
			"warnings":["partial response"]
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"instant","query":"up"}`},
		testDependencies(server.URL, server.Client(), "test-token"),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind       string `json:"kind"`
		ResultType string `json:"result_type"`
		Count      int    `json:"count"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  struct {
				Value string `json:"value"`
			} `json:"value"`
		} `json:"result"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "query" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "query")
	}

	if output.ResultType != "vector" {
		t.Fatalf("ResultType = %q, want %q", output.ResultType, "vector")
	}

	if output.Count != 1 {
		t.Fatalf("Count = %d, want %d", output.Count, 1)
	}

	if output.Result[0].Metric["instance"] != "node-1" {
		t.Fatalf("Metric.instance = %q, want %q", output.Result[0].Metric["instance"], "node-1")
	}

	if output.Result[0].Value.Value != "1" {
		t.Fatalf("Value = %q, want %q", output.Result[0].Value.Value, "1")
	}

	if output.Warnings[0] != "partial response" {
		t.Fatalf("Warnings = %v, want partial response", output.Warnings)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesRangeQueryWithPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}

		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/query_range")
		}

		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q, want form content type", got)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm returned error: %v", err)
		}

		if got := r.PostForm.Get("step"); got != "5m" {
			t.Fatalf("step = %q, want %q", got, "5m")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":{
				"resultType":"matrix",
				"result":[
					{
						"metric":{"__name__":"up","job":"node"},
						"values":[
							[1710283200,"1"],
							[1710283500,"0"]
						]
					}
				]
			}
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(`{"kind":"range","query":"up","start":"2026-03-13T00:00:00Z","end":"2026-03-13T00:10:00Z","step":"5m","http_method":"POST"}`),
		&stdout,
		&stderr,
		nil,
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind       string `json:"kind"`
		ResultType string `json:"result_type"`
		Result     []struct {
			Values []struct {
				Value string `json:"value"`
			} `json:"values"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "query_range" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "query_range")
	}

	if output.ResultType != "matrix" {
		t.Fatalf("ResultType = %q, want %q", output.ResultType, "matrix")
	}

	if len(output.Result[0].Values) != 2 {
		t.Fatalf("len(Values) = %d, want %d", len(output.Result[0].Values), 2)
	}

	if output.Result[0].Values[1].Value != "0" {
		t.Fatalf("Values[1].Value = %q, want %q", output.Result[0].Values[1].Value, "0")
	}
}

func TestRootCommandFetchesLabelValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/label/__name__/values" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/label/__name__/values")
		}

		if got := r.URL.Query()["match[]"]; len(got) != 1 || got[0] != `{job="api"}` {
			t.Fatalf("match[] = %v, want %q", got, `{job="api"}`)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":["go_goroutines","http_requests_total","up"],
			"infos":["trimmed by backend limit"]
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"label_values","label":"__name__","matchers":["{job=\"api\"}"],"limit":2}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind   string   `json:"kind"`
		Label  string   `json:"label"`
		Count  int      `json:"count"`
		Values []string `json:"values"`
		Infos  []string `json:"infos"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "label_values" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "label_values")
	}

	if output.Label != "__name__" {
		t.Fatalf("Label = %q, want %q", output.Label, "__name__")
	}

	if output.Count != 2 {
		t.Fatalf("Count = %d, want %d", output.Count, 2)
	}

	if len(output.Values) != 2 {
		t.Fatalf("len(Values) = %d, want %d", len(output.Values), 2)
	}

	if output.Infos[0] != "trimmed by backend limit" {
		t.Fatalf("Infos = %v, want trimmed info", output.Infos)
	}
}

func TestRootCommandReturnsPrometheusAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"error",
			"errorType":"bad_data",
			"error":"parse error"
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"query","query":"{"}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err == nil {
		t.Fatal("executeWithDependencies returned nil error, want prometheus api error")
	}

	if !strings.Contains(err.Error(), "bad_data: parse error") {
		t.Fatalf("error = %v, want prometheus error", err)
	}
}

func executeWithDependencies(stdin *strings.Reader, stdout, stderr *bytes.Buffer, args []string, deps promapp.Dependencies) error {
	return promapp.ExecuteWithDependencies(stdin, stdout, stderr, args, Version, deps)
}

func testDependencies(baseURL string, httpClient *http.Client, token string) promapp.Dependencies {
	return promapp.Dependencies{
		LoadConfig: func(request prom.Request) (prom.ClientConfig, error) {
			return prom.ClientConfig{
				BaseURL: baseURL,
				Token:   token,
				Timeout: time.Second,
			}, nil
		},
		HTTPClient: httpClient,
	}
}
