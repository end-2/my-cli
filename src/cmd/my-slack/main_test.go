package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	slackapp "github.com/end-2/my-cli/src/cmd/my-slack/internal/app"
	"github.com/end-2/my-cli/src/cmd/my-slack/internal/slack"
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
		"Call Slack Web API methods with one JSON request",
		"--dry-run",
		"--version",
		"my-slack.yaml",
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
		[]string{"--dry-run", `{"kind":"read","method":"conversations.info","args":{"channel":"C123"}}`},
		slackapp.Dependencies{
			LoadConfig: func(request slack.Request) (slack.ClientConfig, error) {
				return slack.ClientConfig{
					BaseURL: "https://slack.example.com/api/",
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

	if got := httpOutput["url"]; got != "https://slack.example.com/api/conversations.info?channel=C123" {
		t.Fatalf("http.url = %v, want custom configured URL", got)
	}

	if got := httpOutput["auth"]; got != "token" {
		t.Fatalf("http.auth = %v, want %q", got, "token")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesReadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/conversations.info" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/conversations.info")
		}

		if got := r.URL.Query().Get("channel"); got != "C123" {
			t.Fatalf("channel = %q, want %q", got, "C123")
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer test-token")
		}

		if got := r.Header.Get("User-Agent"); got != slack.DefaultUserAgent {
			t.Fatalf("User-Agent = %q, want %q", got, slack.DefaultUserAgent)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"channel": {
				"id": "C123",
				"name": "eng-platform",
				"is_archived": false
			}
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"read","method":"conversations.info","args":{"channel":"C123"}}`},
		testDependencies(server.URL, server.Client(), "test-token"),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind     string `json:"kind"`
		Method   string `json:"method"`
		Response struct {
			Channel struct {
				Name string `json:"name"`
			} `json:"channel"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "read" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "read")
	}

	if output.Method != "conversations.info" {
		t.Fatalf("Method = %q, want %q", output.Method, "conversations.info")
	}

	if output.Response.Channel.Name != "eng-platform" {
		t.Fatalf("Response.Channel.Name = %q, want %q", output.Response.Channel.Name, "eng-platform")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesCreateRequestFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}

		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/chat.postMessage")
		}

		if got := r.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want JSON content type", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}

		if body["channel"] != "C123" {
			t.Fatalf("body[channel] = %v, want %q", body["channel"], "C123")
		}

		if body["text"] != "hello from my-slack" {
			t.Fatalf("body[text] = %v, want message text", body["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"channel": "C123",
			"ts": "1710000000.000100",
			"message": {
				"text": "hello from my-slack"
			}
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(`{"kind":"create","method":"chat.postMessage","args":{"channel":"C123","text":"hello from my-slack"}}`),
		&stdout,
		&stderr,
		nil,
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Response struct {
			TS string `json:"ts"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Response.TS != "1710000000.000100" {
		t.Fatalf("Response.TS = %q, want message ts", output.Response.TS)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesListWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.list" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/conversations.list")
		}

		if got := r.URL.Query().Get("limit"); got != "3" {
			t.Fatalf("limit = %q, want %q", got, "3")
		}

		switch r.URL.Query().Get("cursor") {
		case "":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"ok": true,
				"channels": [
					{"id":"C1","name":"eng-platform"},
					{"id":"C2","name":"eng-data"}
				],
				"response_metadata": {
					"next_cursor": "cursor-2"
				}
			}`))
		case "cursor-2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"ok": true,
				"channels": [
					{"id":"C3","name":"eng-web"}
				],
				"response_metadata": {
					"next_cursor": ""
				}
			}`))
		default:
			t.Fatalf("cursor = %q, want empty or %q", r.URL.Query().Get("cursor"), "cursor-2")
		}
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"list","method":"conversations.list","limit":3}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind string `json:"kind"`
		List struct {
			Field string `json:"field"`
			Limit int    `json:"limit"`
			Count int    `json:"count"`
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"list"`
		Response struct {
			Channels []struct {
				ID string `json:"id"`
			} `json:"channels"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "list" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "list")
	}

	if output.List.Field != "channels" {
		t.Fatalf("List.Field = %q, want %q", output.List.Field, "channels")
	}

	if output.List.Limit != 3 {
		t.Fatalf("List.Limit = %d, want %d", output.List.Limit, 3)
	}

	if output.List.Count != 3 {
		t.Fatalf("List.Count = %d, want %d", output.List.Count, 3)
	}

	if len(output.List.Items) != 3 {
		t.Fatalf("len(List.Items) = %d, want %d", len(output.List.Items), 3)
	}

	if output.List.Items[2].Name != "eng-web" {
		t.Fatalf("List.Items[2].Name = %q, want %q", output.List.Items[2].Name, "eng-web")
	}

	if len(output.Response.Channels) != 3 {
		t.Fatalf("len(Response.Channels) = %d, want %d", len(output.Response.Channels), 3)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandReturnsSlackAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"read","method":"conversations.info","args":{"channel":"C999"}}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err == nil {
		t.Fatal("executeWithDependencies returned nil error, want slack api error")
	}

	if !strings.Contains(err.Error(), "channel_not_found") {
		t.Fatalf("error = %v, want channel_not_found", err)
	}
}

func executeWithDependencies(stdin *strings.Reader, stdout, stderr *bytes.Buffer, args []string, deps slackapp.Dependencies) error {
	return slackapp.ExecuteWithDependencies(stdin, stdout, stderr, args, Version, deps)
}

func testDependencies(baseURL string, httpClient *http.Client, token string) slackapp.Dependencies {
	return slackapp.Dependencies{
		LoadConfig: func(request slack.Request) (slack.ClientConfig, error) {
			return slack.ClientConfig{
				BaseURL: baseURL,
				Token:   token,
				Timeout: time.Second,
			}, nil
		},
		HTTPClient: httpClient,
	}
}
