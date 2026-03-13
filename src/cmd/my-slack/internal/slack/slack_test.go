package slack

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseRequestListAliasAndArgsLimit(t *testing.T) {
	request, err := ParseRequest(`{"kind":"ls","method":"users.list","args":{"limit":"50","cursor":" next-1 "},"list_field":"members"}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.Kind != "list" {
		t.Fatalf("request.Kind = %q, want %q", request.Kind, "list")
	}

	if request.Limit == nil {
		t.Fatal("request.Limit = nil, want 50")
	}

	if *request.Limit != 50 {
		t.Fatalf("*request.Limit = %d, want %d", *request.Limit, 50)
	}

	if request.Cursor != "next-1" {
		t.Fatalf("request.Cursor = %q, want %q", request.Cursor, "next-1")
	}
}

func TestParseRequestAcceptsConfigSelectors(t *testing.T) {
	request, err := ParseRequest(`{"kind":"read","method":"conversations.info","args":{"channel":"C123"},"base_url":" https://slack.example.com/api ","alias":" workspace-prod "}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.BaseURL != "https://slack.example.com/api" {
		t.Fatalf("request.BaseURL = %q, want trimmed base URL", request.BaseURL)
	}

	if request.Alias != "workspace-prod" {
		t.Fatalf("request.Alias = %q, want trimmed alias", request.Alias)
	}
}

func TestParseRequestRejectsLimitMismatch(t *testing.T) {
	_, err := ParseRequest(`{"kind":"list","method":"users.list","limit":20,"args":{"limit":10}}`)
	if err == nil {
		t.Fatal("ParseRequest returned nil error, want limit mismatch")
	}

	if !strings.Contains(err.Error(), `"limit" and "args.limit" must match`) {
		t.Fatalf("error = %v, want limit mismatch", err)
	}
}

func TestBuildRequestList(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:   "list",
		Method: "conversations.list",
		Args: map[string]any{
			"types": "public_channel,private_channel",
		},
		Limit:  intPtr(25),
		Cursor: "cursor-2",
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if plan.HTTPMethod != "GET" {
		t.Fatalf("plan.HTTPMethod = %q, want %q", plan.HTTPMethod, "GET")
	}

	if got := plan.URL.String(); got != "https://slack.com/api/conversations.list?cursor=cursor-2&limit=25&types=public_channel%2Cprivate_channel" {
		t.Fatalf("plan.URL.String() = %q, want list URL", got)
	}
}

func TestBuildRequestCreateUsesPOSTBody(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:   "create",
		Method: "chat.postMessage",
		Args: map[string]any{
			"channel": "C123",
			"text":    "hello",
		},
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if plan.HTTPMethod != "POST" {
		t.Fatalf("plan.HTTPMethod = %q, want %q", plan.HTTPMethod, "POST")
	}

	if plan.ContentType != "application/json; charset=utf-8" {
		t.Fatalf("plan.ContentType = %q, want JSON content type", plan.ContentType)
	}

	var body map[string]any
	if err := json.Unmarshal(plan.Body, &body); err != nil {
		t.Fatalf("plan.Body is not valid json: %v", err)
	}

	if body["channel"] != "C123" {
		t.Fatalf("body[channel] = %v, want %q", body["channel"], "C123")
	}
}

func TestNormalizeBaseURLAppendsTrailingSlash(t *testing.T) {
	baseURL, err := NormalizeBaseURL("https://slack.example.com/api")
	if err != nil {
		t.Fatalf("NormalizeBaseURL returned error: %v", err)
	}

	if baseURL != "https://slack.example.com/api/" {
		t.Fatalf("baseURL = %q, want trailing slash", baseURL)
	}
}

func intPtr(value int) *int {
	return &value
}
