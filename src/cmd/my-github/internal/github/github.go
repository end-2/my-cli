package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/end-2/my-cli/src/pkg/cliutil"
)

const (
	DefaultAPIBaseURL   = "https://api.github.com/"
	APIVersion          = "2022-11-28"
	DefaultUserAgent    = "my-cli/my-github"
	DefaultTimeout      = 15 * time.Second
	DefaultListLimit    = 30
	MaxListLimit        = 100
	DefaultHistoryLimit = DefaultListLimit
	MaxHistoryLimit     = MaxListLimit
)

type Request struct {
	Kind    string `json:"kind"`
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Number  int    `json:"number,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Limit   *int   `json:"limit,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
	Alias   string `json:"alias,omitempty"`
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
	r.Owner = strings.TrimSpace(r.Owner)
	r.Repo = strings.TrimSpace(r.Repo)
	r.Ref = strings.TrimSpace(r.Ref)
	r.BaseURL = strings.TrimSpace(r.BaseURL)
	r.Alias = strings.TrimSpace(r.Alias)

	if r.Owner == "" {
		return errors.New("json input field \"owner\" is required")
	}

	if r.Repo == "" {
		return errors.New("json input field \"repo\" is required")
	}

	switch r.Kind {
	case "issue", "pull_request":
		if r.Number <= 0 {
			return fmt.Errorf("json input field \"number\" must be greater than zero for kind %q", r.Kind)
		}

		if r.Ref != "" {
			return fmt.Errorf("json input field \"ref\" is not allowed for kind %q", r.Kind)
		}
	case "issue_list", "pull_request_list":
		if r.Number != 0 {
			return fmt.Errorf("json input field \"number\" is not allowed for kind %q", r.Kind)
		}

		if r.Ref != "" {
			return fmt.Errorf("json input field \"ref\" is not allowed for kind %q", r.Kind)
		}

		if r.Limit != nil && (*r.Limit < 1 || *r.Limit > MaxListLimit) {
			return fmt.Errorf("json input field \"limit\" must be between 1 and %d for kind %q", MaxListLimit, r.Kind)
		}
	case "commit":
		if r.Ref == "" {
			return errors.New("json input field \"ref\" is required for kind \"commit\"")
		}

		if r.Number != 0 {
			return errors.New("json input field \"number\" is not allowed for kind \"commit\"")
		}
	case "commit_history":
		if r.Ref == "" {
			return errors.New("json input field \"ref\" is required for kind \"commit_history\"")
		}

		if r.Number != 0 {
			return errors.New("json input field \"number\" is not allowed for kind \"commit_history\"")
		}

		if r.Limit != nil && (*r.Limit < 1 || *r.Limit > MaxHistoryLimit) {
			return fmt.Errorf("json input field \"limit\" must be between 1 and %d for kind %q", MaxHistoryLimit, r.Kind)
		}
	default:
		return fmt.Errorf(
			"json input field \"kind\" must be one of %q, %q, %q, %q, %q, or %q",
			"issue",
			"issue_list",
			"pull_request",
			"pull_request_list",
			"commit",
			"commit_history",
		)
	}

	return nil
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "pr", "pull-request", "pull_request":
		return "pull_request"
	case "issue-list", "issue_list", "issues":
		return "issue_list"
	case "pr-list", "pr_list", "prs", "pull-request-list", "pull_request_list", "pulls":
		return "pull_request_list"
	case "commit-history", "commit_history":
		return "commit_history"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
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
		return "", errors.New("parse github api base url: empty value")
	}

	baseURL, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse github api base url: %w", err)
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
		return nil, fmt.Errorf("parse github api base url: %w", err)
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
	URL  *url.URL
	Path string
}

func (c *Client) BuildRequest(input Request) (RequestPlan, error) {
	var path string

	switch input.Kind {
	case "issue":
		path = fmt.Sprintf("repos/%s/%s/issues/%d", url.PathEscape(input.Owner), url.PathEscape(input.Repo), input.Number)
	case "issue_list":
		path = fmt.Sprintf("repos/%s/%s/issues", url.PathEscape(input.Owner), url.PathEscape(input.Repo))
	case "pull_request":
		path = fmt.Sprintf("repos/%s/%s/pulls/%d", url.PathEscape(input.Owner), url.PathEscape(input.Repo), input.Number)
	case "pull_request_list":
		path = fmt.Sprintf("repos/%s/%s/pulls", url.PathEscape(input.Owner), url.PathEscape(input.Repo))
	case "commit":
		path = fmt.Sprintf("repos/%s/%s/commits/%s", url.PathEscape(input.Owner), url.PathEscape(input.Repo), url.PathEscape(input.Ref))
	case "commit_history":
		path = fmt.Sprintf("repos/%s/%s/commits", url.PathEscape(input.Owner), url.PathEscape(input.Repo))
	default:
		return RequestPlan{}, fmt.Errorf("unsupported kind %q", input.Kind)
	}

	endpoint, err := c.baseURL.Parse(path)
	if err != nil {
		return RequestPlan{}, fmt.Errorf("build github api url: %w", err)
	}

	switch input.Kind {
	case "issue_list", "pull_request_list":
		query := endpoint.Query()
		query.Set("per_page", strconv.Itoa(listLimit(input.Limit)))
		endpoint.RawQuery = query.Encode()
	case "commit_history":
		query := endpoint.Query()
		query.Set("sha", input.Ref)
		query.Set("per_page", strconv.Itoa(listLimit(input.Limit)))
		endpoint.RawQuery = query.Encode()
	}

	return RequestPlan{
		URL:  endpoint,
		Path: endpoint.RequestURI(),
	}, nil
}

func (c *Client) Execute(plan RequestPlan, input Request) (any, error) {
	if input.Kind == "issue_list" {
		return c.executeIssueList(plan, input)
	}

	resp, err := c.do(plan)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, decodeAPIError(resp, plan.Path)
	}

	switch input.Kind {
	case "issue":
		return decodeIssueOutput(resp.Body, input)
	case "pull_request":
		return decodePullRequestOutput(resp.Body, input)
	case "pull_request_list":
		return decodePullRequestListOutput(resp.Body, input)
	case "commit":
		return decodeCommitOutput(resp.Body, input)
	case "commit_history":
		return decodeCommitHistoryOutput(resp.Body, input)
	default:
		return nil, fmt.Errorf("unsupported kind %q", input.Kind)
	}
}

func (c *Client) do(plan RequestPlan) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, plan.URL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create github api request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-GitHub-Api-Version", APIVersion)

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call github api %s: %w", plan.Path, err)
	}

	return resp, nil
}

func (c *Client) executeIssueList(plan RequestPlan, input Request) (issueListEnvelope, error) {
	limit := listLimit(input.Limit)
	items := make([]issueOutput, 0, limit)

	for page := 1; len(items) < limit; page++ {
		pagePlan := requestPlanWithPage(plan, page)

		resp, err := c.do(pagePlan)
		if err != nil {
			return issueListEnvelope{}, err
		}

		pageItems, rawCount, err := func() ([]issueOutput, int, error) {
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode >= http.StatusBadRequest {
				return nil, 0, decodeAPIError(resp, pagePlan.Path)
			}

			return decodeIssueListPage(resp.Body)
		}()
		if err != nil {
			return issueListEnvelope{}, err
		}

		for _, item := range pageItems {
			if len(items) >= limit {
				break
			}

			items = append(items, item)
		}

		if rawCount < listLimit(input.Limit) {
			break
		}
	}

	return issueListEnvelope{
		Kind: "issue_list",
		Repository: repositoryOutput{
			Owner: input.Owner,
			Repo:  input.Repo,
		},
		IssueList: issueListOutput{
			Limit:  limit,
			Issues: items,
		},
	}, nil
}

func requestPlanWithPage(plan RequestPlan, page int) RequestPlan {
	endpoint := *plan.URL
	query := endpoint.Query()
	query.Set("page", strconv.Itoa(page))
	endpoint.RawQuery = query.Encode()

	return RequestPlan{
		URL:  &endpoint,
		Path: endpoint.RequestURI(),
	}
}

type apiError struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
}

func decodeAPIError(resp *http.Response, path string) error {
	var apiErr apiError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("github api %s returned %s", path, resp.Status)
	}

	if apiErr.Message == "" {
		return fmt.Errorf("github api %s returned %s", path, resp.Status)
	}

	if apiErr.DocumentationURL != "" {
		return fmt.Errorf("github api %s returned %s: %s (%s)", path, resp.Status, apiErr.Message, apiErr.DocumentationURL)
	}

	return fmt.Errorf("github api %s returned %s: %s", path, resp.Status, apiErr.Message)
}

type repositoryOutput struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

type issueEnvelope struct {
	Kind       string           `json:"kind"`
	Repository repositoryOutput `json:"repository"`
	Issue      issueOutput      `json:"issue"`
}

type issueListEnvelope struct {
	Kind       string           `json:"kind"`
	Repository repositoryOutput `json:"repository"`
	IssueList  issueListOutput  `json:"issue_list"`
}

type issueAPIResponse struct {
	URL         string               `json:"url"`
	HTMLURL     string               `json:"html_url"`
	Number      int                  `json:"number"`
	Title       string               `json:"title"`
	State       string               `json:"state"`
	Body        string               `json:"body"`
	Comments    int                  `json:"comments"`
	User        gitHubUser           `json:"user"`
	Assignees   []gitHubUser         `json:"assignees"`
	Labels      []gitHubLabel        `json:"labels"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
	ClosedAt    *string              `json:"closed_at"`
	PullRequest *issuePullRequestRef `json:"pull_request"`
}

type issuePullRequestRef struct {
	URL string `json:"url"`
}

type issueOutput struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Author    string   `json:"author"`
	Assignees []string `json:"assignees"`
	Labels    []string `json:"labels"`
	Comments  int      `json:"comments"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	ClosedAt  *string  `json:"closed_at,omitempty"`
	URL       string   `json:"url"`
	APIURL    string   `json:"api_url"`
	Body      string   `json:"body"`
}

func decodeIssueOutput(body io.Reader, input Request) (issueEnvelope, error) {
	var payload issueAPIResponse
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return issueEnvelope{}, fmt.Errorf("decode github issue response: %w", err)
	}

	if payload.PullRequest != nil {
		return issueEnvelope{}, fmt.Errorf("github item %s/%s#%d is a pull request, not an issue", input.Owner, input.Repo, input.Number)
	}

	return issueEnvelope{
		Kind: "issue",
		Repository: repositoryOutput{
			Owner: input.Owner,
			Repo:  input.Repo,
		},
		Issue: normalizeIssueOutput(payload),
	}, nil
}

type issueListOutput struct {
	Limit  int           `json:"limit"`
	Issues []issueOutput `json:"issues"`
}

type pullRequestEnvelope struct {
	Kind        string            `json:"kind"`
	Repository  repositoryOutput  `json:"repository"`
	PullRequest pullRequestOutput `json:"pull_request"`
}

type pullRequestListEnvelope struct {
	Kind            string                `json:"kind"`
	Repository      repositoryOutput      `json:"repository"`
	PullRequestList pullRequestListOutput `json:"pull_request_list"`
}

type pullRequestAPIResponse struct {
	URL       string     `json:"url"`
	HTMLURL   string     `json:"html_url"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Body      string     `json:"body"`
	Draft     bool       `json:"draft"`
	Merged    bool       `json:"merged"`
	User      gitHubUser `json:"user"`
	Base      gitHubRef  `json:"base"`
	Head      gitHubRef  `json:"head"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	MergedAt  *string    `json:"merged_at"`
}

type gitHubRef struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type pullRequestOutput struct {
	Number     int     `json:"number"`
	Title      string  `json:"title"`
	State      string  `json:"state"`
	Draft      bool    `json:"draft"`
	Merged     bool    `json:"merged"`
	Author     string  `json:"author"`
	BaseBranch string  `json:"base_branch"`
	BaseSHA    string  `json:"base_sha"`
	HeadBranch string  `json:"head_branch"`
	HeadSHA    string  `json:"head_sha"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	MergedAt   *string `json:"merged_at,omitempty"`
	URL        string  `json:"url"`
	APIURL     string  `json:"api_url"`
	Body       string  `json:"body"`
}

func decodePullRequestOutput(body io.Reader, input Request) (pullRequestEnvelope, error) {
	var payload pullRequestAPIResponse
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return pullRequestEnvelope{}, fmt.Errorf("decode github pull request response: %w", err)
	}

	return pullRequestEnvelope{
		Kind: "pull_request",
		Repository: repositoryOutput{
			Owner: input.Owner,
			Repo:  input.Repo,
		},
		PullRequest: normalizePullRequestOutput(payload),
	}, nil
}

type pullRequestListOutput struct {
	Limit        int                 `json:"limit"`
	PullRequests []pullRequestOutput `json:"pull_requests"`
}

type commitEnvelope struct {
	Kind       string           `json:"kind"`
	Repository repositoryOutput `json:"repository"`
	Commit     commitOutput     `json:"commit"`
}

type commitHistoryEnvelope struct {
	Kind          string              `json:"kind"`
	Repository    repositoryOutput    `json:"repository"`
	CommitHistory commitHistoryOutput `json:"commit_history"`
}

type commitAPIResponse struct {
	SHA       string          `json:"sha"`
	URL       string          `json:"url"`
	HTMLURL   string          `json:"html_url"`
	Author    *gitHubUser     `json:"author"`
	Committer *gitHubUser     `json:"committer"`
	Commit    gitCommitDetail `json:"commit"`
	Parents   []gitHubParent  `json:"parents"`
	Stats     *gitCommitStats `json:"stats"`
	Files     []gitCommitFile `json:"files"`
}

type gitCommitDetail struct {
	Author    gitIdentity `json:"author"`
	Committer gitIdentity `json:"committer"`
	Message   string      `json:"message"`
}

type gitIdentity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type gitHubParent struct {
	SHA string `json:"sha"`
}

type gitHubUser struct {
	Login string `json:"login"`
}

type gitCommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

type gitCommitFile struct {
	Filename         string `json:"filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	PreviousFilename string `json:"previous_filename"`
	Patch            string `json:"patch"`
}

type gitHubLabel struct {
	Name string `json:"name"`
}

type commitOutput struct {
	SHA       string             `json:"sha"`
	Message   string             `json:"message"`
	Author    commitPerson       `json:"author"`
	Committer commitPerson       `json:"committer"`
	Parents   []string           `json:"parents"`
	Stats     *commitStatsOutput `json:"stats,omitempty"`
	Files     []commitFileOutput `json:"files,omitempty"`
	URL       string             `json:"url"`
	APIURL    string             `json:"api_url"`
}

type commitHistoryOutput struct {
	Ref     string         `json:"ref"`
	Limit   int            `json:"limit"`
	Commits []commitOutput `json:"commits"`
}

type commitPerson struct {
	Login string `json:"login,omitempty"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type commitStatsOutput struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

type commitFileOutput struct {
	Filename         string `json:"filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	PreviousFilename string `json:"previous_filename,omitempty"`
	Patch            string `json:"patch,omitempty"`
}

func decodeCommitOutput(body io.Reader, input Request) (commitEnvelope, error) {
	var payload commitAPIResponse
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return commitEnvelope{}, fmt.Errorf("decode github commit response: %w", err)
	}

	return commitEnvelope{
		Kind: "commit",
		Repository: repositoryOutput{
			Owner: input.Owner,
			Repo:  input.Repo,
		},
		Commit: normalizeCommitOutput(payload),
	}, nil
}

func decodeCommitHistoryOutput(body io.Reader, input Request) (commitHistoryEnvelope, error) {
	var payload []commitAPIResponse
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return commitHistoryEnvelope{}, fmt.Errorf("decode github commit history response: %w", err)
	}

	commits := make([]commitOutput, 0, len(payload))
	for _, item := range payload {
		commits = append(commits, normalizeCommitOutput(item))
	}

	return commitHistoryEnvelope{
		Kind: "commit_history",
		Repository: repositoryOutput{
			Owner: input.Owner,
			Repo:  input.Repo,
		},
		CommitHistory: commitHistoryOutput{
			Ref:     input.Ref,
			Limit:   listLimit(input.Limit),
			Commits: commits,
		},
	}, nil
}

func decodeIssueListPage(body io.Reader) ([]issueOutput, int, error) {
	var payload []issueAPIResponse
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, 0, fmt.Errorf("decode github issue list response: %w", err)
	}

	items := make([]issueOutput, 0, len(payload))
	for _, item := range payload {
		if item.PullRequest != nil {
			continue
		}

		items = append(items, normalizeIssueOutput(item))
	}

	return items, len(payload), nil
}

func decodePullRequestListOutput(body io.Reader, input Request) (pullRequestListEnvelope, error) {
	var payload []pullRequestAPIResponse
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return pullRequestListEnvelope{}, fmt.Errorf("decode github pull request list response: %w", err)
	}

	items := make([]pullRequestOutput, 0, len(payload))
	for _, item := range payload {
		items = append(items, normalizePullRequestOutput(item))
	}

	return pullRequestListEnvelope{
		Kind: "pull_request_list",
		Repository: repositoryOutput{
			Owner: input.Owner,
			Repo:  input.Repo,
		},
		PullRequestList: pullRequestListOutput{
			Limit:        listLimit(input.Limit),
			PullRequests: items,
		},
	}, nil
}

func normalizeCommitOutput(payload commitAPIResponse) commitOutput {
	output := commitOutput{
		SHA:     payload.SHA,
		Message: payload.Commit.Message,
		Author: commitPerson{
			Login: loginOrEmpty(payload.Author),
			Name:  payload.Commit.Author.Name,
			Email: payload.Commit.Author.Email,
			Date:  payload.Commit.Author.Date,
		},
		Committer: commitPerson{
			Login: loginOrEmpty(payload.Committer),
			Name:  payload.Commit.Committer.Name,
			Email: payload.Commit.Committer.Email,
			Date:  payload.Commit.Committer.Date,
		},
		Parents: collectParentSHAs(payload.Parents),
		URL:     payload.HTMLURL,
		APIURL:  payload.URL,
	}

	if payload.Stats != nil {
		output.Stats = &commitStatsOutput{
			Additions: payload.Stats.Additions,
			Deletions: payload.Stats.Deletions,
			Total:     payload.Stats.Total,
		}
	}

	if len(payload.Files) > 0 {
		output.Files = normalizeCommitFiles(payload.Files)
	}

	return output
}

func normalizeIssueOutput(payload issueAPIResponse) issueOutput {
	return issueOutput{
		Number:    payload.Number,
		Title:     payload.Title,
		State:     payload.State,
		Author:    payload.User.Login,
		Assignees: collectUserLogins(payload.Assignees),
		Labels:    collectLabelNames(payload.Labels),
		Comments:  payload.Comments,
		CreatedAt: payload.CreatedAt,
		UpdatedAt: payload.UpdatedAt,
		ClosedAt:  payload.ClosedAt,
		URL:       payload.HTMLURL,
		APIURL:    payload.URL,
		Body:      payload.Body,
	}
}

func normalizePullRequestOutput(payload pullRequestAPIResponse) pullRequestOutput {
	return pullRequestOutput{
		Number:     payload.Number,
		Title:      payload.Title,
		State:      payload.State,
		Draft:      payload.Draft,
		Merged:     payload.Merged,
		Author:     payload.User.Login,
		BaseBranch: payload.Base.Ref,
		BaseSHA:    payload.Base.SHA,
		HeadBranch: payload.Head.Ref,
		HeadSHA:    payload.Head.SHA,
		CreatedAt:  payload.CreatedAt,
		UpdatedAt:  payload.UpdatedAt,
		MergedAt:   payload.MergedAt,
		URL:        payload.HTMLURL,
		APIURL:     payload.URL,
		Body:       payload.Body,
	}
}

func listLimit(limit *int) int {
	if limit == nil {
		return DefaultListLimit
	}

	return *limit
}

func loginOrEmpty(user *gitHubUser) string {
	if user == nil {
		return ""
	}

	return user.Login
}

func collectUserLogins(users []gitHubUser) []string {
	logins := make([]string, 0, len(users))
	for _, user := range users {
		if user.Login == "" {
			continue
		}

		logins = append(logins, user.Login)
	}

	return logins
}

func collectLabelNames(labels []gitHubLabel) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		if label.Name == "" {
			continue
		}

		names = append(names, label.Name)
	}

	return names
}

func normalizeCommitFiles(files []gitCommitFile) []commitFileOutput {
	output := make([]commitFileOutput, 0, len(files))
	for _, file := range files {
		output = append(output, commitFileOutput{
			Filename:         file.Filename,
			Status:           file.Status,
			Additions:        file.Additions,
			Deletions:        file.Deletions,
			Changes:          file.Changes,
			PreviousFilename: file.PreviousFilename,
			Patch:            file.Patch,
		})
	}

	return output
}

func collectParentSHAs(parents []gitHubParent) []string {
	shas := make([]string, 0, len(parents))
	for _, parent := range parents {
		if parent.SHA == "" {
			continue
		}

		shas = append(shas, parent.SHA)
	}

	return shas
}
