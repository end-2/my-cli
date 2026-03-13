package discord

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseRequestListAliasAndSelectors(t *testing.T) {
	request, err := ParseRequest(`{"kind":"ls","path":" guilds/G1/members ","limit":200,"page_limit":100,"after":" 0 ","cursor_field":" user.id ","base_url":" https://discord.example.com/api/v10 ","alias":" bot-prod "}`)
	if err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if request.Kind != "list" {
		t.Fatalf("request.Kind = %q, want %q", request.Kind, "list")
	}

	if request.Path != "/guilds/G1/members" {
		t.Fatalf("request.Path = %q, want %q", request.Path, "/guilds/G1/members")
	}

	if request.After != "0" {
		t.Fatalf("request.After = %q, want %q", request.After, "0")
	}

	if request.CursorField != "user.id" {
		t.Fatalf("request.CursorField = %q, want %q", request.CursorField, "user.id")
	}

	if request.BaseURL != "https://discord.example.com/api/v10" {
		t.Fatalf("request.BaseURL = %q, want trimmed base URL", request.BaseURL)
	}

	if request.Alias != "bot-prod" {
		t.Fatalf("request.Alias = %q, want %q", request.Alias, "bot-prod")
	}
}

func TestParseRequestRejectsBeforeAndAfter(t *testing.T) {
	_, err := ParseRequest(`{"kind":"list","path":"/channels/123/messages","before":"1","after":"2"}`)
	if err == nil {
		t.Fatal("ParseRequest returned nil error, want validation error")
	}

	if !strings.Contains(err.Error(), `"before" and "after" cannot be used together`) {
		t.Fatalf("error = %v, want pagination validation error", err)
	}
}

func TestParseRequestRejectsReservedListQueryKeys(t *testing.T) {
	_, err := ParseRequest(`{"kind":"list","path":"/channels/123/messages","query":{"limit":50}}`)
	if err == nil {
		t.Fatal("ParseRequest returned nil error, want reserved query validation error")
	}

	if !strings.Contains(err.Error(), `"query.limit" is reserved`) {
		t.Fatalf("error = %v, want reserved query validation error", err)
	}
}

func TestBuildRequestListUsesPageLimitAndBefore(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:      "list",
		Path:      "/channels/123/messages",
		Query:     map[string]any{"around": "145000000000000010"},
		PageLimit: intPtr(25),
		Before:    "145000000000000002",
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if plan.HTTPMethod != "GET" {
		t.Fatalf("plan.HTTPMethod = %q, want %q", plan.HTTPMethod, "GET")
	}

	if got := plan.URL.String(); got != "https://discord.com/api/v10/channels/123/messages?around=145000000000000010&before=145000000000000002&limit=25" {
		t.Fatalf("plan.URL.String() = %q, want list URL", got)
	}
}

func TestBuildRequestUpdateUsesPATCHBodyAndReason(t *testing.T) {
	client, err := NewClient(ClientConfig{}, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	plan, err := client.BuildRequest(Request{
		Kind:   "update",
		Path:   "/channels/123",
		Body:   map[string]any{"name": "eng-platform"},
		Reason: "rename channel",
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if plan.HTTPMethod != "PATCH" {
		t.Fatalf("plan.HTTPMethod = %q, want %q", plan.HTTPMethod, "PATCH")
	}

	if plan.ContentType != "application/json; charset=utf-8" {
		t.Fatalf("plan.ContentType = %q, want JSON content type", plan.ContentType)
	}

	if plan.AuditLogReason != "rename%20channel" {
		t.Fatalf("plan.AuditLogReason = %q, want encoded reason", plan.AuditLogReason)
	}

	var body map[string]any
	if err := json.Unmarshal(plan.Body, &body); err != nil {
		t.Fatalf("plan.Body is not valid json: %v", err)
	}

	if body["name"] != "eng-platform" {
		t.Fatalf("body[name] = %v, want %q", body["name"], "eng-platform")
	}
}

func TestExtractListItemsRequiresListFieldWhenMultipleArraysExist(t *testing.T) {
	_, _, err := extractListItems(map[string]any{
		"users":             []any{map[string]any{"id": "1"}},
		"audit_log_entries": []any{map[string]any{"id": "2"}},
	}, "")
	if err == nil {
		t.Fatal("extractListItems returned nil error, want inference error")
	}

	if !strings.Contains(err.Error(), `provide json input field "list_field"`) {
		t.Fatalf("error = %v, want list_field hint", err)
	}
}

func TestExtractCursorValueSupportsNestedField(t *testing.T) {
	value, err := extractCursorValue(map[string]any{
		"user": map[string]any{
			"id": json.Number("101"),
		},
	}, "user.id")
	if err != nil {
		t.Fatalf("extractCursorValue returned error: %v", err)
	}

	if value != "101" {
		t.Fatalf("value = %q, want %q", value, "101")
	}
}

func TestNormalizeBaseURLAppendsTrailingSlash(t *testing.T) {
	baseURL, err := NormalizeBaseURL("https://discord.example.com/api/v10")
	if err != nil {
		t.Fatalf("NormalizeBaseURL returned error: %v", err)
	}

	if baseURL != "https://discord.example.com/api/v10/" {
		t.Fatalf("baseURL = %q, want trailing slash", baseURL)
	}
}
