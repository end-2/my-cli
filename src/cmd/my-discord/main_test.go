package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	discordapp "github.com/end-2/my-cli/src/cmd/my-discord/internal/app"
	"github.com/end-2/my-cli/src/cmd/my-discord/internal/discord"
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
		"Call Discord REST API routes with one JSON request",
		"--dry-run",
		"--version",
		"my-discord.yaml",
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
		[]string{"--dry-run", `{"kind":"read","path":"/channels/123"}`},
		discordapp.Dependencies{
			LoadConfig: func(request discord.Request) (discord.ClientConfig, error) {
				return discord.ClientConfig{
					BaseURL: "https://discord.example.com/api/v10/",
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

	httpOutput, ok := output["http"].(map[string]any)
	if !ok {
		t.Fatalf("http = %T, want map[string]any", output["http"])
	}

	if got := httpOutput["url"]; got != "https://discord.example.com/api/v10/channels/123" {
		t.Fatalf("http.url = %v, want custom configured URL", got)
	}

	if got := httpOutput["method"]; got != "GET" {
		t.Fatalf("http.method = %v, want %q", got, "GET")
	}

	if got := httpOutput["auth"]; got != "bot_token" {
		t.Fatalf("http.auth = %v, want %q", got, "bot_token")
	}
}

func TestRootCommandFetchesReadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/api/v10/channels/123" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v10/channels/123")
		}

		if got := r.Header.Get("Authorization"); got != "Bot test-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bot test-token")
		}

		if got := r.Header.Get("User-Agent"); got != discord.DefaultUserAgent {
			t.Fatalf("User-Agent = %q, want %q", got, discord.DefaultUserAgent)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","name":"eng-platform","type":0}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"read","path":"/channels/123"}`},
		testDependencies(server.URL+"/api/v10/", server.Client(), "test-token"),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind       string `json:"kind"`
		Path       string `json:"path"`
		HTTPMethod string `json:"http_method"`
		Response   struct {
			Name string `json:"name"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "read" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "read")
	}

	if output.Path != "/channels/123" {
		t.Fatalf("Path = %q, want %q", output.Path, "/channels/123")
	}

	if output.HTTPMethod != "GET" {
		t.Fatalf("HTTPMethod = %q, want %q", output.HTTPMethod, "GET")
	}

	if output.Response.Name != "eng-platform" {
		t.Fatalf("Response.Name = %q, want %q", output.Response.Name, "eng-platform")
	}
}

func TestRootCommandFetchesCreateRequestFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}

		if r.URL.Path != "/api/v10/channels/123/messages" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v10/channels/123/messages")
		}

		if got := r.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want JSON content type", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}

		if body["content"] != "hello from my-discord" {
			t.Fatalf("body[content] = %v, want message content", body["content"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"456","content":"hello from my-discord"}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(`{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from my-discord"}}`),
		&stdout,
		&stderr,
		nil,
		testDependencies(server.URL+"/api/v10/", server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Response struct {
			ID string `json:"id"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Response.ID != "456" {
		t.Fatalf("Response.ID = %q, want %q", output.Response.ID, "456")
	}
}

func TestRootCommandFetchesListWithAfterPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v10/guilds/G1/members" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v10/guilds/G1/members")
		}

		switch r.URL.Query().Get("after") {
		case "0":
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Fatalf("limit = %q, want %q", got, "2")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"user":{"id":"100","username":"alpha"}},
				{"user":{"id":"101","username":"beta"}}
			]`))
		case "101":
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Fatalf("limit = %q, want %q", got, "2")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"user":{"id":"102","username":"gamma"}}
			]`))
		default:
			t.Fatalf("after = %q, want %q or %q", r.URL.Query().Get("after"), "0", "101")
		}
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"list","path":"/guilds/G1/members","limit":3,"page_limit":2,"after":"0","cursor_field":"user.id"}`},
		testDependencies(server.URL+"/api/v10/", server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind string `json:"kind"`
		List struct {
			CursorField string `json:"cursor_field"`
			Pagination  string `json:"pagination"`
			Limit       int    `json:"limit"`
			Count       int    `json:"count"`
			Items       []struct {
				User struct {
					ID string `json:"id"`
				} `json:"user"`
			} `json:"items"`
		} `json:"list"`
		Response []struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "list" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "list")
	}

	if output.List.CursorField != "user.id" {
		t.Fatalf("List.CursorField = %q, want %q", output.List.CursorField, "user.id")
	}

	if output.List.Pagination != "after" {
		t.Fatalf("List.Pagination = %q, want %q", output.List.Pagination, "after")
	}

	if output.List.Limit != 3 || output.List.Count != 3 {
		t.Fatalf("List limit/count = %d/%d, want 3/3", output.List.Limit, output.List.Count)
	}

	if len(output.Response) != 3 {
		t.Fatalf("len(Response) = %d, want %d", len(output.Response), 3)
	}

	if output.Response[2].User.ID != "102" {
		t.Fatalf("Response[2].User.ID = %q, want %q", output.Response[2].User.ID, "102")
	}
}

func TestRootCommandSendsAuditLogReasonAndHandlesNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodDelete)
		}

		if got := r.Header.Get("X-Audit-Log-Reason"); got != "cleanup%20old%20message" {
			t.Fatalf("X-Audit-Log-Reason = %q, want encoded reason", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"delete","path":"/channels/123/messages/456","reason":"cleanup old message"}`},
		testDependencies(server.URL+"/api/v10/", server.Client(), "test-token"),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), `"response": {}`) {
		t.Fatalf("stdout = %q, want empty object response", stdout.String())
	}
}

func TestRootCommandReturnsDiscordAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Unknown Channel","code":10003}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"read","path":"/channels/999"}`},
		testDependencies(server.URL+"/api/v10/", server.Client(), ""),
	)
	if err == nil {
		t.Fatal("executeWithDependencies returned nil error, want discord api error")
	}

	if !strings.Contains(err.Error(), "Unknown Channel") {
		t.Fatalf("error = %v, want Unknown Channel", err)
	}
}

func executeWithDependencies(stdin *strings.Reader, stdout, stderr *bytes.Buffer, args []string, deps discordapp.Dependencies) error {
	return discordapp.ExecuteWithDependencies(stdin, stdout, stderr, args, Version, deps)
}

func testDependencies(baseURL string, httpClient *http.Client, token string) discordapp.Dependencies {
	return discordapp.Dependencies{
		LoadConfig: func(request discord.Request) (discord.ClientConfig, error) {
			return discord.ClientConfig{
				BaseURL: baseURL,
				Token:   token,
				Timeout: time.Second,
			}, nil
		},
		HTTPClient: httpClient,
	}
}
