package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ghapp "github.com/end-2/my-cli/src/cmd/my-github/internal/app"
	"github.com/end-2/my-cli/src/cmd/my-github/internal/github"
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

func TestRootCommandPrintsVersionWithSingleDashLongFlag(t *testing.T) {
	originalVersion := Version
	Version = "9.9.9"
	t.Cleanup(func() {
		Version = originalVersion
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(strings.NewReader(""), &stdout, &stderr, []string{"-version"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got != "9.9.9\n" {
		t.Fatalf("stdout = %q, want %q", got, "9.9.9\n")
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
		"Query GitHub issues, pull requests, and commits",
		"--dry-run",
		"--version",
		"my-github.yaml",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("stdout = %q, want to contain %q", output, expected)
		}
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandPrintsHelpWithSingleDashLongFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := execute(strings.NewReader(""), &stdout, &stderr, []string{"-help"}); err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	if got := stdout.String(); got == "" {
		t.Fatal("stdout is empty, want help output")
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
		[]string{"--dry-run", `{"kind":"issue","owner":"cli","repo":"cli","number":123}`},
		ghapp.Dependencies{
			LoadConfig: func() (github.ClientConfig, error) {
				return github.ClientConfig{
					BaseURL: "https://example.github.local/api/v3",
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

	if got := httpOutput["url"]; got != "https://example.github.local/api/v3/repos/cli/cli/issues/123" {
		t.Fatalf("http.url = %v, want custom configured URL", got)
	}

	if got := httpOutput["auth"]; got != "token" {
		t.Fatalf("http.auth = %v, want %q", got, "token")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/repos/cli/cli/issues/123" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/repos/cli/cli/issues/123")
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer test-token")
		}

		if got := r.Header.Get("X-GitHub-Api-Version"); got != github.APIVersion {
			t.Fatalf("X-GitHub-Api-Version = %q, want %q", got, github.APIVersion)
		}

		if got := r.Header.Get("User-Agent"); got != github.DefaultUserAgent {
			t.Fatalf("User-Agent = %q, want %q", got, github.DefaultUserAgent)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"url":"https://api.github.com/repos/cli/cli/issues/123",
			"html_url":"https://github.com/cli/cli/issues/123",
			"number":123,
			"title":"Issue title",
			"state":"open",
			"body":"Issue body",
			"comments":4,
			"user":{"login":"octocat"},
			"assignees":[{"login":"hubot"}],
			"labels":[{"name":"bug"},{"name":"good first issue"}],
			"created_at":"2026-03-10T12:00:00Z",
			"updated_at":"2026-03-11T12:00:00Z",
			"closed_at":null
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"issue","owner":"cli","repo":"cli","number":123}`},
		testDependencies(server.URL, server.Client(), "test-token"),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind  string `json:"kind"`
		Issue struct {
			Title  string   `json:"title"`
			Labels []string `json:"labels"`
		} `json:"issue"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "issue" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "issue")
	}

	if output.Issue.Title != "Issue title" {
		t.Fatalf("Issue.Title = %q, want %q", output.Issue.Title, "Issue title")
	}

	if len(output.Issue.Labels) != 2 {
		t.Fatalf("Issue.Labels = %v, want 2 labels", output.Issue.Labels)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesPullRequestWithAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/cli/cli/pulls/456" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/repos/cli/cli/pulls/456")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"url":"https://api.github.com/repos/cli/cli/pulls/456",
			"html_url":"https://github.com/cli/cli/pull/456",
			"number":456,
			"title":"PR title",
			"state":"open",
			"body":"PR body",
			"draft":false,
			"merged":false,
			"user":{"login":"monalisa"},
			"base":{"ref":"main","sha":"base-sha"},
			"head":{"ref":"feature","sha":"head-sha"},
			"created_at":"2026-03-10T12:00:00Z",
			"updated_at":"2026-03-11T12:00:00Z",
			"merged_at":null
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"pr","owner":"cli","repo":"cli","number":456}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind        string `json:"kind"`
		PullRequest struct {
			HeadBranch string `json:"head_branch"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "pull_request" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "pull_request")
	}

	if output.PullRequest.HeadBranch != "feature" {
		t.Fatalf("PullRequest.HeadBranch = %q, want %q", output.PullRequest.HeadBranch, "feature")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesCommitFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/cli/cli/commits/trunk" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/repos/cli/cli/commits/trunk")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sha":"abc123",
			"url":"https://api.github.com/repos/cli/cli/commits/abc123",
			"html_url":"https://github.com/cli/cli/commit/abc123",
			"author":{"login":"octocat"},
			"commit":{
				"author":{"name":"Octo Cat","email":"octo@example.com","date":"2026-03-10T12:00:00Z"},
				"committer":{"name":"Octo Bot","email":"bot@example.com","date":"2026-03-10T12:01:00Z"},
				"message":"Commit message"
			},
			"parents":[{"sha":"parent1"},{"sha":"parent2"}]
		}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(`{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}`),
		&stdout,
		&stderr,
		nil,
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Commit struct {
			SHA    string `json:"sha"`
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Parents []string `json:"parents"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Commit.SHA != "abc123" {
		t.Fatalf("Commit.SHA = %q, want %q", output.Commit.SHA, "abc123")
	}

	if output.Commit.Author.Login != "octocat" {
		t.Fatalf("Commit.Author.Login = %q, want %q", output.Commit.Author.Login, "octocat")
	}

	if len(output.Commit.Parents) != 2 {
		t.Fatalf("Commit.Parents = %v, want 2 parents", output.Commit.Parents)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandReturnsValidationErrorForInvalidJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"issue","owner":"cli","repo":"cli"}`},
		testDependencies("https://api.github.com", http.DefaultClient, ""),
	)
	if err == nil {
		t.Fatal("executeWithDependencies returned nil error, want validation error")
	}

	if !strings.Contains(err.Error(), `"number" must be greater than zero`) {
		t.Fatalf("error = %v, want number validation error", err)
	}
}

func TestRootCommandReturnsGitHubAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found","documentation_url":"https://docs.github.com/rest"}`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"issue","owner":"cli","repo":"cli","number":999}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err == nil {
		t.Fatal("executeWithDependencies returned nil error, want github api error")
	}

	if !strings.Contains(err.Error(), "404 Not Found: Not Found") {
		t.Fatalf("error = %v, want 404 error", err)
	}
}

func executeWithDependencies(stdin *strings.Reader, stdout, stderr *bytes.Buffer, args []string, deps ghapp.Dependencies) error {
	return ghapp.ExecuteWithDependencies(stdin, stdout, stderr, args, Version, deps)
}

func testDependencies(baseURL string, httpClient *http.Client, token string) ghapp.Dependencies {
	return ghapp.Dependencies{
		LoadConfig: func() (github.ClientConfig, error) {
			return github.ClientConfig{
				BaseURL: baseURL,
				Token:   token,
				Timeout: time.Second,
			}, nil
		},
		HTTPClient: httpClient,
	}
}
