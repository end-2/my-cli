package github

import (
	"strings"
	"testing"
)

func TestParseRequestCommitHistoryAlias(t *testing.T) {
	request, err := ParseRequest(`{"kind":"commit-history","owner":"cli","repo":"cli","ref":"release/1.0","limit":10}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.Kind != "commit_history" {
		t.Fatalf("request.Kind = %q, want %q", request.Kind, "commit_history")
	}

	if request.Limit == nil {
		t.Fatal("request.Limit = nil, want 10")
	}

	if *request.Limit != 10 {
		t.Fatalf("*request.Limit = %d, want %d", *request.Limit, 10)
	}
}

func TestParseRequestIssueListAlias(t *testing.T) {
	request, err := ParseRequest(`{"kind":"issue-list","owner":"cli","repo":"cli","limit":10}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.Kind != "issue_list" {
		t.Fatalf("request.Kind = %q, want %q", request.Kind, "issue_list")
	}
}

func TestParseRequestPullRequestListAlias(t *testing.T) {
	request, err := ParseRequest(`{"kind":"pr-list","owner":"cli","repo":"cli","limit":10}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.Kind != "pull_request_list" {
		t.Fatalf("request.Kind = %q, want %q", request.Kind, "pull_request_list")
	}
}

func TestParseRequestAcceptsConfigSelectors(t *testing.T) {
	request, err := ParseRequest(`{"kind":"issue","owner":"cli","repo":"cli","number":123,"base_url":" https://ghe.example.com/api/v3 ","alias":" example-ghe "}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.BaseURL != "https://ghe.example.com/api/v3" {
		t.Fatalf("request.BaseURL = %q, want trimmed base URL", request.BaseURL)
	}

	if request.Alias != "example-ghe" {
		t.Fatalf("request.Alias = %q, want trimmed alias", request.Alias)
	}
}

func TestParseRequestRejectsCommitHistoryLimitAboveMax(t *testing.T) {
	_, err := ParseRequest(`{"kind":"commit_history","owner":"cli","repo":"cli","ref":"main","limit":101}`)
	if err == nil {
		t.Fatal("ParseRequest returned nil error, want limit validation error")
	}

	if !strings.Contains(err.Error(), `"limit" must be between 1 and 100`) {
		t.Fatalf("error = %v, want limit validation error", err)
	}
}

func TestParseRequestRejectsIssueListNumber(t *testing.T) {
	_, err := ParseRequest(`{"kind":"issue_list","owner":"cli","repo":"cli","number":1}`)
	if err == nil {
		t.Fatal("ParseRequest returned nil error, want validation error")
	}

	if !strings.Contains(err.Error(), `"number" is not allowed`) {
		t.Fatalf("error = %v, want number validation error", err)
	}
}

func TestBuildRequestCommitHistory(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:  "commit_history",
		Owner: "cli",
		Repo:  "cli",
		Ref:   "release/1.0",
		Limit: intPtr(5),
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if got := plan.URL.String(); got != "https://api.github.com/repos/cli/cli/commits?per_page=5&sha=release%2F1.0" {
		t.Fatalf("plan.URL.String() = %q, want commit history URL", got)
	}
}

func TestBuildRequestIssueList(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:  "issue_list",
		Owner: "cli",
		Repo:  "cli",
		Limit: intPtr(5),
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if got := plan.URL.String(); got != "https://api.github.com/repos/cli/cli/issues?per_page=5" {
		t.Fatalf("plan.URL.String() = %q, want issue list URL", got)
	}
}

func TestBuildRequestPullRequestListUsesDefaultLimit(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:  "pull_request_list",
		Owner: "cli",
		Repo:  "cli",
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if got := plan.URL.String(); got != "https://api.github.com/repos/cli/cli/pulls?per_page=30" {
		t.Fatalf("plan.URL.String() = %q, want default pull request list URL", got)
	}
}

func TestBuildRequestCommitHistoryUsesDefaultLimit(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:  "commit_history",
		Owner: "cli",
		Repo:  "cli",
		Ref:   "main",
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if got := plan.URL.String(); got != "https://api.github.com/repos/cli/cli/commits?per_page=30&sha=main" {
		t.Fatalf("plan.URL.String() = %q, want default limit URL", got)
	}
}

func TestNormalizeBaseURLAppendsTrailingSlash(t *testing.T) {
	baseURL, err := NormalizeBaseURL("https://ghe.example.com/api/v3")
	if err != nil {
		t.Fatalf("NormalizeBaseURL returned error: %v", err)
	}

	if baseURL != "https://ghe.example.com/api/v3/" {
		t.Fatalf("baseURL = %q, want trailing slash", baseURL)
	}
}

func intPtr(value int) *int {
	return &value
}
