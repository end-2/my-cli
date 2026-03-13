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
		"Query GitHub issues, pull requests, issue lists, pull request lists, commits, and commit history",
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
			LoadConfig: func(request github.Request) (github.ClientConfig, error) {
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

func TestRootCommandPassesConfigSelectorsToLoader(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	var loadedRequest github.Request

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{"--dry-run", `{"kind":"issue","owner":"cli","repo":"cli","number":123,"base_url":"https://ghe.example.com/api/v3","alias":"example-ghe"}`},
		ghapp.Dependencies{
			LoadConfig: func(request github.Request) (github.ClientConfig, error) {
				loadedRequest = request

				return github.ClientConfig{
					BaseURL: "https://ghe.example.com/api/v3",
					Token:   "configured-token",
					Timeout: time.Second,
				}, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	if loadedRequest.BaseURL != "https://ghe.example.com/api/v3" {
		t.Fatalf("loadedRequest.BaseURL = %q, want request base URL", loadedRequest.BaseURL)
	}

	if loadedRequest.Alias != "example-ghe" {
		t.Fatalf("loadedRequest.Alias = %q, want request alias", loadedRequest.Alias)
	}

	var output struct {
		HTTP struct {
			URL string `json:"url"`
		} `json:"http"`
		Request struct {
			BaseURL string `json:"base_url"`
			Alias   string `json:"alias"`
		} `json:"request"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.HTTP.URL != "https://ghe.example.com/api/v3/repos/cli/cli/issues/123" {
		t.Fatalf("HTTP.URL = %q, want request-selected URL", output.HTTP.URL)
	}

	if output.Request.BaseURL != "https://ghe.example.com/api/v3" {
		t.Fatalf("Request.BaseURL = %q, want %q", output.Request.BaseURL, "https://ghe.example.com/api/v3")
	}

	if output.Request.Alias != "example-ghe" {
		t.Fatalf("Request.Alias = %q, want %q", output.Request.Alias, "example-ghe")
	}
}

func TestRootCommandPrintsDryRunPlanForCommitHistory(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{"--dry-run", `{"kind":"commit_history","owner":"cli","repo":"cli","ref":"release/1.0","limit":2}`},
		testDependencies("https://api.github.com", http.DefaultClient, ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		HTTP struct {
			URL string `json:"url"`
		} `json:"http"`
		Request struct {
			Kind  string `json:"kind"`
			Ref   string `json:"ref"`
			Limit int    `json:"limit"`
		} `json:"request"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.HTTP.URL != "https://api.github.com/repos/cli/cli/commits?per_page=2&sha=release%2F1.0" {
		t.Fatalf("HTTP.URL = %q, want commit history URL", output.HTTP.URL)
	}

	if output.Request.Kind != "commit_history" {
		t.Fatalf("Request.Kind = %q, want %q", output.Request.Kind, "commit_history")
	}

	if output.Request.Ref != "release/1.0" {
		t.Fatalf("Request.Ref = %q, want %q", output.Request.Ref, "release/1.0")
	}

	if output.Request.Limit != 2 {
		t.Fatalf("Request.Limit = %d, want %d", output.Request.Limit, 2)
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

func TestRootCommandFetchesIssueList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/cli/cli/issues" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/repos/cli/cli/issues")
		}

		if got := r.URL.Query().Get("per_page"); got != "2" {
			t.Fatalf("per_page = %q, want %q", got, "2")
		}

		switch r.URL.Query().Get("page") {
		case "1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"url":"https://api.github.com/repos/cli/cli/issues/999",
					"html_url":"https://github.com/cli/cli/pull/999",
					"number":999,
					"title":"Pull request item",
					"state":"open",
					"body":"PR body",
					"comments":1,
					"user":{"login":"octocat"},
					"assignees":[],
					"labels":[],
					"created_at":"2026-03-10T12:00:00Z",
					"updated_at":"2026-03-11T12:00:00Z",
					"closed_at":null,
					"pull_request":{"url":"https://api.github.com/repos/cli/cli/pulls/999"}
				},
				{
					"url":"https://api.github.com/repos/cli/cli/issues/123",
					"html_url":"https://github.com/cli/cli/issues/123",
					"number":123,
					"title":"First issue",
					"state":"open",
					"body":"Issue body",
					"comments":4,
					"user":{"login":"octocat"},
					"assignees":[{"login":"hubot"}],
					"labels":[{"name":"bug"}],
					"created_at":"2026-03-10T12:00:00Z",
					"updated_at":"2026-03-11T12:00:00Z",
					"closed_at":null
				}
			]`))
		case "2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"url":"https://api.github.com/repos/cli/cli/issues/122",
					"html_url":"https://github.com/cli/cli/issues/122",
					"number":122,
					"title":"Second issue",
					"state":"open",
					"body":"Another issue body",
					"comments":2,
					"user":{"login":"hubot"},
					"assignees":[],
					"labels":[{"name":"docs"}],
					"created_at":"2026-03-09T12:00:00Z",
					"updated_at":"2026-03-10T12:00:00Z",
					"closed_at":null
				}
			]`))
		default:
			t.Fatalf("page = %q, want %q or %q", r.URL.Query().Get("page"), "1", "2")
		}
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"issue_list","owner":"cli","repo":"cli","limit":2}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind      string `json:"kind"`
		IssueList struct {
			Limit  int `json:"limit"`
			Issues []struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"issues"`
		} `json:"issue_list"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "issue_list" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "issue_list")
	}

	if output.IssueList.Limit != 2 {
		t.Fatalf("IssueList.Limit = %d, want %d", output.IssueList.Limit, 2)
	}

	if len(output.IssueList.Issues) != 2 {
		t.Fatalf("len(IssueList.Issues) = %d, want %d", len(output.IssueList.Issues), 2)
	}

	if output.IssueList.Issues[0].Number != 123 {
		t.Fatalf("IssueList.Issues[0].Number = %d, want %d", output.IssueList.Issues[0].Number, 123)
	}

	if output.IssueList.Issues[1].Title != "Second issue" {
		t.Fatalf("IssueList.Issues[1].Title = %q, want %q", output.IssueList.Issues[1].Title, "Second issue")
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

func TestRootCommandFetchesPullRequestListWithAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/cli/cli/pulls" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/repos/cli/cli/pulls")
		}

		if got := r.URL.Query().Get("per_page"); got != "2" {
			t.Fatalf("per_page = %q, want %q", got, "2")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"url":"https://api.github.com/repos/cli/cli/pulls/456",
				"html_url":"https://github.com/cli/cli/pull/456",
				"number":456,
				"title":"PR title",
				"state":"open",
				"body":"PR body",
				"draft":false,
				"user":{"login":"monalisa"},
				"base":{"ref":"main","sha":"base-sha"},
				"head":{"ref":"feature","sha":"head-sha"},
				"created_at":"2026-03-10T12:00:00Z",
				"updated_at":"2026-03-11T12:00:00Z",
				"merged_at":null
			},
			{
				"url":"https://api.github.com/repos/cli/cli/pulls/455",
				"html_url":"https://github.com/cli/cli/pull/455",
				"number":455,
				"title":"Older PR",
				"state":"open",
				"body":"Older PR body",
				"draft":true,
				"user":{"login":"hubot"},
				"base":{"ref":"main","sha":"older-base"},
				"head":{"ref":"feature-2","sha":"older-head"},
				"created_at":"2026-03-09T12:00:00Z",
				"updated_at":"2026-03-10T12:00:00Z",
				"merged_at":null
			}
		]`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"pr-list","owner":"cli","repo":"cli","limit":2}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind            string `json:"kind"`
		PullRequestList struct {
			Limit        int `json:"limit"`
			PullRequests []struct {
				Number     int    `json:"number"`
				HeadBranch string `json:"head_branch"`
				Draft      bool   `json:"draft"`
			} `json:"pull_requests"`
		} `json:"pull_request_list"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "pull_request_list" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "pull_request_list")
	}

	if output.PullRequestList.Limit != 2 {
		t.Fatalf("PullRequestList.Limit = %d, want %d", output.PullRequestList.Limit, 2)
	}

	if len(output.PullRequestList.PullRequests) != 2 {
		t.Fatalf("len(PullRequestList.PullRequests) = %d, want %d", len(output.PullRequestList.PullRequests), 2)
	}

	if output.PullRequestList.PullRequests[0].HeadBranch != "feature" {
		t.Fatalf("PullRequestList.PullRequests[0].HeadBranch = %q, want %q", output.PullRequestList.PullRequests[0].HeadBranch, "feature")
	}

	if !output.PullRequestList.PullRequests[1].Draft {
		t.Fatal("PullRequestList.PullRequests[1].Draft = false, want true")
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
			"parents":[{"sha":"parent1"},{"sha":"parent2"}],
			"stats":{"additions":12,"deletions":3,"total":15},
			"files":[
				{
					"filename":"README.md",
					"status":"modified",
					"additions":10,
					"deletions":2,
					"changes":12,
					"patch":"@@ -1 +1 @@\\n-old\\n+new"
				},
				{
					"filename":"docs/old.md",
					"previous_filename":"docs/legacy.md",
					"status":"renamed",
					"additions":2,
					"deletions":1,
					"changes":3
				}
			]
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
			Stats   struct {
				Additions int `json:"additions"`
				Deletions int `json:"deletions"`
				Total     int `json:"total"`
			} `json:"stats"`
			Files []struct {
				Filename         string `json:"filename"`
				Status           string `json:"status"`
				Changes          int    `json:"changes"`
				PreviousFilename string `json:"previous_filename"`
				Patch            string `json:"patch"`
			} `json:"files"`
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

	if output.Commit.Stats.Total != 15 {
		t.Fatalf("Commit.Stats.Total = %d, want %d", output.Commit.Stats.Total, 15)
	}

	if len(output.Commit.Files) != 2 {
		t.Fatalf("len(Commit.Files) = %d, want %d", len(output.Commit.Files), 2)
	}

	if output.Commit.Files[0].Filename != "README.md" {
		t.Fatalf("Commit.Files[0].Filename = %q, want %q", output.Commit.Files[0].Filename, "README.md")
	}

	if output.Commit.Files[0].Patch == "" {
		t.Fatal("Commit.Files[0].Patch is empty, want patch content")
	}

	if output.Commit.Files[1].PreviousFilename != "docs/legacy.md" {
		t.Fatalf("Commit.Files[1].PreviousFilename = %q, want %q", output.Commit.Files[1].PreviousFilename, "docs/legacy.md")
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandFetchesCommitHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/cli/cli/commits" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/repos/cli/cli/commits")
		}

		if got := r.URL.Query().Get("sha"); got != "release/1.0" {
			t.Fatalf("sha = %q, want %q", got, "release/1.0")
		}

		if got := r.URL.Query().Get("per_page"); got != "2" {
			t.Fatalf("per_page = %q, want %q", got, "2")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"sha":"abc123",
				"url":"https://api.github.com/repos/cli/cli/commits/abc123",
				"html_url":"https://github.com/cli/cli/commit/abc123",
				"author":{"login":"octocat"},
				"committer":{"login":"github-actions[bot]"},
				"commit":{
					"author":{"name":"Octo Cat","email":"octo@example.com","date":"2026-03-10T12:00:00Z"},
					"committer":{"name":"GitHub Actions","email":"bot@example.com","date":"2026-03-10T12:01:00Z"},
					"message":"First commit"
				},
				"parents":[{"sha":"parent1"}]
			},
			{
				"sha":"def456",
				"url":"https://api.github.com/repos/cli/cli/commits/def456",
				"html_url":"https://github.com/cli/cli/commit/def456",
				"author":null,
				"committer":null,
				"commit":{
					"author":{"name":"Mona Lisa","email":"mona@example.com","date":"2026-03-09T10:00:00Z"},
					"committer":{"name":"Mona Lisa","email":"mona@example.com","date":"2026-03-09T10:05:00Z"},
					"message":"Second commit"
				},
				"parents":[{"sha":"parent2"},{"sha":"parent3"}]
			}
		]`))
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeWithDependencies(
		strings.NewReader(""),
		&stdout,
		&stderr,
		[]string{`{"kind":"commit_history","owner":"cli","repo":"cli","ref":"release/1.0","limit":2}`},
		testDependencies(server.URL, server.Client(), ""),
	)
	if err != nil {
		t.Fatalf("executeWithDependencies returned error: %v", err)
	}

	var output struct {
		Kind          string `json:"kind"`
		CommitHistory struct {
			Ref     string `json:"ref"`
			Limit   int    `json:"limit"`
			Commits []struct {
				SHA       string   `json:"sha"`
				Message   string   `json:"message"`
				Parents   []string `json:"parents"`
				Committer struct {
					Login string `json:"login"`
				} `json:"committer"`
			} `json:"commits"`
		} `json:"commit_history"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}

	if output.Kind != "commit_history" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "commit_history")
	}

	if output.CommitHistory.Ref != "release/1.0" {
		t.Fatalf("CommitHistory.Ref = %q, want %q", output.CommitHistory.Ref, "release/1.0")
	}

	if output.CommitHistory.Limit != 2 {
		t.Fatalf("CommitHistory.Limit = %d, want %d", output.CommitHistory.Limit, 2)
	}

	if len(output.CommitHistory.Commits) != 2 {
		t.Fatalf("len(CommitHistory.Commits) = %d, want %d", len(output.CommitHistory.Commits), 2)
	}

	if output.CommitHistory.Commits[0].SHA != "abc123" {
		t.Fatalf("CommitHistory.Commits[0].SHA = %q, want %q", output.CommitHistory.Commits[0].SHA, "abc123")
	}

	if output.CommitHistory.Commits[0].Committer.Login != "github-actions[bot]" {
		t.Fatalf("CommitHistory.Commits[0].Committer.Login = %q, want %q", output.CommitHistory.Commits[0].Committer.Login, "github-actions[bot]")
	}

	if output.CommitHistory.Commits[1].Message != "Second commit" {
		t.Fatalf("CommitHistory.Commits[1].Message = %q, want %q", output.CommitHistory.Commits[1].Message, "Second commit")
	}

	if len(output.CommitHistory.Commits[1].Parents) != 2 {
		t.Fatalf("CommitHistory.Commits[1].Parents = %v, want 2 parents", output.CommitHistory.Commits[1].Parents)
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
		LoadConfig: func(request github.Request) (github.ClientConfig, error) {
			return github.ClientConfig{
				BaseURL: baseURL,
				Token:   token,
				Timeout: time.Second,
			}, nil
		},
		HTTPClient: httpClient,
	}
}
