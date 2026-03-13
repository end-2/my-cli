# my-github

`my-github` is a single-purpose CLI for fetching issues, issue lists, pull requests, pull request lists, commits, and commit history for a specific ref from the GitHub REST API.  
It accepts exactly one JSON object as input and prints JSON as output.

This command is part of the `my-cli` project, and all build, test, and lint tasks run inside Docker containers.

For rules on using the `my-github` binary in LLM / agent environments, see [docs/my-github/SKILL.md](../../../docs/my-github/SKILL.md).

## Requirements

- Docker
- GNU Make

You do not need to install Go or `golangci-lint` locally.

## Build

Build from the repository root like this.

```bash
make build my-github
make build CMD=my-github
```

Common option examples:

```bash
make build my-github VERSION=1.0.0
make build my-github GO_VERSION=1.26.1
make build my-github CGO_ENABLED=1
make build my-github GOOS=linux GOARCH=amd64
make build my-github GOPRIVATE='github.com/your-org/*'
```

The output binary is created at `bin/my-github`.  
During the build, `ldflags` injects a `VERSION-git_commit` value into `main.Version`.

```bash
./bin/my-github --help
./bin/my-github --dry-run '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
./bin/my-github --version
./bin/my-github -version
```

## Install into Codex

To use it directly from Codex CLI, run the following script from the repository root.

```bash
./scripts/install-my-github-codex.sh
```

This script runs `make build CMD=my-github`, then updates both `~/.codex/bin/my-github` and `~/.codex/skills/my-github/*`.  
If you use a different Codex home, run it like `CODEX_HOME=/path/to/codex ./scripts/install-my-github-codex.sh`.

## Test

Tests run inside the `golang:<GO_VERSION>` Docker image.

```bash
make test my-github
make test CMD=my-github
```

You can pass extra options as well.

```bash
make test my-github TEST_FLAGS="-v"
make test my-github TEST_FLAGS="-run TestRootCommandFetchesIssue -v"
```

## Lint

Linting uses `golangci-lint` inside Docker. The default image is `golangci/golangci-lint:v2.9.0`.

```bash
make lint my-github
make lint CMD=my-github
```

Additional option examples:

```bash
make lint my-github LINT_FLAGS="--verbose"
make lint my-github LINT_TIMEOUT=10m
make lint my-github GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

```bash
make list-cmds
make print-version
make clean
```

## Configuration File

The configuration file name is `my-github.yaml`.  
Settings are loaded through [`src/pkg/config/config.go`](../../pkg/config/config.go).

The search paths are checked in this order.

1. `/etc/my-github/my-github.yaml`
2. `~/.config/my-github.yaml`
3. `./my-github.yaml`

Existing files are merged in that order, and later values override earlier ones.  
If no configuration file exists, the following defaults are used.

- `github.base_url`: `https://api.github.com/`
- `github.timeout`: `15s`
- `github.user_agent`: `my-cli/my-github`
- `github.token`: empty
- `github.by_base_url`: empty

The top-level `github` values are the shared defaults.  
Selection starts from `github.base_url`, and if the request JSON includes `base_url` or `alias`, that choice takes precedence.  
If there is a `github.by_base_url[]` entry that matches the selected GitHub instance, its `base_url`, `token`, `timeout`, and `user_agent` values are applied last.  
`github.by_base_url[].alias` can be selected directly through the request JSON `alias` field, and `base_url` matching ignores a trailing `/`.

Example:

```yaml
github:
  base_url: https://api.github.com/
  timeout: 15s
  user_agent: my-cli/my-github
  by_base_url:
    - alias: github.com
      base_url: https://api.github.com/
      token: "{{ .GITHUB_TOKEN }}"
    - alias: example-ghe
      base_url: https://ghe.example.com/api/v3/
      token: "{{ .GHE_TOKEN }}"
      timeout: 30s
      user_agent: my-cli/my-github-enterprise
```

Secret values such as `token` can also be managed through config templates.  
In the example above, the runtime environment variables `GITHUB_TOKEN` and `GHE_TOKEN` are read and injected into the token for each matching base URL.  
When you use multiple GitHub instances, changing only `github.base_url` is enough for the matching override to be applied automatically, and `alias` gives you a quick way to identify each entry.

## Usage

Pass JSON input in one of the following ways.

```bash
./bin/my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
```

```bash
./bin/my-github '{"kind":"issue_list","owner":"cli","repo":"cli","limit":10}'
```

```bash
echo '{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}' | ./bin/my-github
```

```bash
./bin/my-github '{"kind":"commit_history","owner":"cli","repo":"cli","ref":"release/1.0","limit":10}'
```

```bash
./bin/my-github '{"kind":"pull_request_list","owner":"cli","repo":"cli","limit":10}'
```

```bash
./bin/my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123,"alias":"example-ghe"}'
```

```bash
./bin/my-github '{"kind":"pull_request","owner":"cli","repo":"cli","number":456,"base_url":"https://ghe.example.com/api/v3"}'
```

Supported flags:

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Common JSON Input Rules

- Only one JSON object is allowed as input.
- At most one argument may be provided.
- Unknown fields are treated as errors.
- `kind`, `owner`, and `repo` are always required.
- `base_url` and `alias` are optional and are used to choose a specific `github.by_base_url` configuration.
- If authentication is required, set `github.token` or the selected `github.by_base_url[].token` in `my-github.yaml`.
- If you need environment-variable-based secret injection, use a template such as `{{ .GITHUB_TOKEN }}` in `github.token` or `github.by_base_url[].token`.
- If `base_url` and `alias` are provided together, they must point to the same `github.by_base_url` entry.

## Common Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kind` | string | Yes | Type of object to fetch |
| `owner` | string | Yes | GitHub owner or org |
| `repo` | string | Yes | GitHub repository name |
| `number` | integer | Conditional | Required for `issue` and `pull_request` |
| `ref` | string | Conditional | Required for `commit` and `commit_history`. For `commit`, use a SHA, branch, or tag. For `commit_history`, a branch name is typical |
| `limit` | integer | Conditional | Optional for `issue_list`, `pull_request_list`, and `commit_history`. Must be between 1 and 100, and defaults to 30 |
| `base_url` | string | No | GitHub API base URL to use for this request. If it matches a `github.by_base_url` entry, that override is applied too |
| `alias` | string | No | `github.by_base_url[].alias` value to use for this request |

## kind Values

| Value | Description |
| --- | --- |
| `issue` | Fetch an issue |
| `issue_list` | Fetch the repository issue list |
| `pull_request` | Fetch a pull request |
| `pull_request_list` | Fetch the repository pull request list |
| `commit` | Fetch a commit |
| `commit_history` | Fetch commit history for a specific ref |

The following aliases are also accepted.

- `issue-list`
- `issues`
- `pr`
- `pull-request`
- `pr-list`
- `pr_list`
- `prs`
- `pull-request-list`
- `pulls`
- `commit-history`

Even if you pass an alias, the output `kind` value is normalized to `issue_list`, `pull_request`, `pull_request_list`, or `commit_history`.

## Spec by Kind

### 1. Issue

Example input:

```json
{
  "kind": "issue",
  "owner": "cli",
  "repo": "cli",
  "number": 123
}
```

Rules:

- `number` must be an integer greater than or equal to 1.
- `ref` is not allowed.
- If the GitHub API returns a pull request item, it is treated as an error instead of an issue.

Example output:

```json
{
  "kind": "issue",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "issue": {
    "number": 123,
    "title": "Issue title",
    "state": "open",
    "author": "octocat",
    "assignees": ["hubot"],
    "labels": ["bug", "good first issue"],
    "comments": 4,
    "created_at": "2026-03-10T12:00:00Z",
    "updated_at": "2026-03-11T12:00:00Z",
    "closed_at": null,
    "url": "https://github.com/cli/cli/issues/123",
    "api_url": "https://api.github.com/repos/cli/cli/issues/123",
    "body": "Issue body"
  }
}
```

`closed_at` may be omitted or `null` if the issue is still open.

### 2. Issue List

Example input:

```json
{
  "kind": "issue_list",
  "owner": "cli",
  "repo": "cli",
  "limit": 2
}
```

`issue-list` and `issues` mean the same thing.

Rules:

- `number` is not allowed.
- `ref` is not allowed.
- `limit` must be between 1 and 100, and defaults to 30.
- If pull requests are mixed into the GitHub `/issues` response, they are filtered out internally.

Example output:

```json
{
  "kind": "issue_list",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "issue_list": {
    "limit": 2,
    "issues": [
      {
        "number": 123,
        "title": "Issue title",
        "state": "open",
        "author": "octocat",
        "assignees": ["hubot"],
        "labels": ["bug"],
        "comments": 4,
        "created_at": "2026-03-10T12:00:00Z",
        "updated_at": "2026-03-11T12:00:00Z",
        "closed_at": null,
        "url": "https://github.com/cli/cli/issues/123",
        "api_url": "https://api.github.com/repos/cli/cli/issues/123",
        "body": "Issue body"
      },
      {
        "number": 122,
        "title": "Second issue",
        "state": "open",
        "author": "hubot",
        "assignees": [],
        "labels": ["docs"],
        "comments": 2,
        "created_at": "2026-03-09T12:00:00Z",
        "updated_at": "2026-03-10T12:00:00Z",
        "closed_at": null,
        "url": "https://github.com/cli/cli/issues/122",
        "api_url": "https://api.github.com/repos/cli/cli/issues/122",
        "body": "Another issue body"
      }
    ]
  }
}
```

### 3. Pull Request

Example input:

```json
{
  "kind": "pull_request",
  "owner": "cli",
  "repo": "cli",
  "number": 456
}
```

`pr` and `pull-request` mean the same thing.

Rules:

- `number` must be an integer greater than or equal to 1.
- `ref` is not allowed.

Example output:

```json
{
  "kind": "pull_request",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "pull_request": {
    "number": 456,
    "title": "PR title",
    "state": "open",
    "draft": false,
    "merged": false,
    "author": "monalisa",
    "base_branch": "main",
    "base_sha": "base-sha",
    "head_branch": "feature",
    "head_sha": "head-sha",
    "created_at": "2026-03-10T12:00:00Z",
    "updated_at": "2026-03-11T12:00:00Z",
    "merged_at": null,
    "url": "https://github.com/cli/cli/pull/456",
    "api_url": "https://api.github.com/repos/cli/cli/pulls/456",
    "body": "PR body"
  }
}
```

`merged_at` may be omitted or `null` if the pull request has not been merged.

### 4. Pull Request List

Example input:

```json
{
  "kind": "pull_request_list",
  "owner": "cli",
  "repo": "cli",
  "limit": 2
}
```

`pr-list`, `pr_list`, `prs`, `pull-request-list`, and `pulls` all mean the same thing.

Rules:

- `number` is not allowed.
- `ref` is not allowed.
- `limit` must be between 1 and 100, and defaults to 30.

Example output:

```json
{
  "kind": "pull_request_list",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "pull_request_list": {
    "limit": 2,
    "pull_requests": [
      {
        "number": 456,
        "title": "PR title",
        "state": "open",
        "draft": false,
        "merged": false,
        "author": "monalisa",
        "base_branch": "main",
        "base_sha": "base-sha",
        "head_branch": "feature",
        "head_sha": "head-sha",
        "created_at": "2026-03-10T12:00:00Z",
        "updated_at": "2026-03-11T12:00:00Z",
        "merged_at": null,
        "url": "https://github.com/cli/cli/pull/456",
        "api_url": "https://api.github.com/repos/cli/cli/pulls/456",
        "body": "PR body"
      }
    ]
  }
}
```

### 5. Commit

Example input:

```json
{
  "kind": "commit",
  "owner": "cli",
  "repo": "cli",
  "ref": "trunk"
}
```

Rules:

- `ref` must not be empty.
- `ref` can be a SHA, branch, or tag.
- `number` is not allowed.

Example output:

```json
{
  "kind": "commit",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "commit": {
    "sha": "abc123",
    "message": "Commit message",
    "author": {
      "login": "octocat",
      "name": "Octo Cat",
      "email": "octo@example.com",
      "date": "2026-03-10T12:00:00Z"
    },
    "committer": {
      "name": "Octo Bot",
      "email": "bot@example.com",
      "date": "2026-03-10T12:01:00Z"
    },
    "parents": ["parent1", "parent2"],
    "stats": {
      "additions": 12,
      "deletions": 3,
      "total": 15
    },
    "files": [
      {
        "filename": "README.md",
        "status": "modified",
        "additions": 10,
        "deletions": 2,
        "changes": 12,
        "patch": "@@ -1 +1 @@\n-old\n+new"
      },
      {
        "filename": "docs/old.md",
        "status": "renamed",
        "additions": 2,
        "deletions": 1,
        "changes": 3,
        "previous_filename": "docs/legacy.md"
      }
    ],
    "url": "https://github.com/cli/cli/commit/abc123",
    "api_url": "https://api.github.com/repos/cli/cli/commits/abc123"
  }
}
```

The GitHub commit author and the Git author may differ, so `author.login` may be missing.  
`files[].patch` may be empty or omitted for binary files or when the diff is too large.

### 6. Commit History

Example input:

```json
{
  "kind": "commit_history",
  "owner": "cli",
  "repo": "cli",
  "ref": "release/1.0",
  "limit": 2
}
```

Rules:

- `ref` must not be empty.
- `ref` is passed to the GitHub API `sha` query parameter, and a branch name is typically used.
- `limit` must be between 1 and 100, and defaults to 30.
- `number` is not allowed.

Example output:

```json
{
  "kind": "commit_history",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "commit_history": {
    "ref": "release/1.0",
    "limit": 2,
    "commits": [
      {
        "sha": "abc123",
        "message": "First commit",
        "author": {
          "login": "octocat",
          "name": "Octo Cat",
          "email": "octo@example.com",
          "date": "2026-03-10T12:00:00Z"
        },
        "committer": {
          "login": "github-actions[bot]",
          "name": "GitHub Actions",
          "email": "bot@example.com",
          "date": "2026-03-10T12:01:00Z"
        },
        "parents": ["parent1"],
        "url": "https://github.com/cli/cli/commit/abc123",
        "api_url": "https://api.github.com/repos/cli/cli/commits/abc123"
      },
      {
        "sha": "def456",
        "message": "Second commit",
        "author": {
          "name": "Mona Lisa",
          "email": "mona@example.com",
          "date": "2026-03-09T10:00:00Z"
        },
        "committer": {
          "name": "Mona Lisa",
          "email": "mona@example.com",
          "date": "2026-03-09T10:05:00Z"
        },
        "parents": ["parent2", "parent3"],
        "url": "https://github.com/cli/cli/commit/def456",
        "api_url": "https://api.github.com/repos/cli/cli/commits/def456"
      }
    ]
  }
}
```

Results follow the GitHub API response order and are typically returned from newest commit to oldest.

## dry-run Output

`--dry-run` does not call the GitHub API. It prints the request that would be sent as JSON.  
The final values computed by combining `github.base_url`, `github.token`, `github.by_base_url`, and the request JSON `base_url` / `alias` are also reflected in the dry-run output.

```json
{
  "mode": "dry-run",
  "http": {
    "method": "GET",
    "url": "https://api.github.com/repos/cli/cli/issues/123",
    "auth": "token"
  },
  "request": {
    "kind": "issue",
    "owner": "cli",
    "repo": "cli",
    "number": 123
  }
}
```

`auth` is one of the following values.

- `token`
- `none`

## Error Rules

The following cases produce an error.

- Invalid JSON syntax
- Input that is not a JSON object
- Input that contains more than one JSON object
- Passing more than one argument
- Including unknown fields
- An invalid `kind` value
- Missing `owner` or `repo`
- Missing `number` or `number <= 0` for `issue` or `pull_request`
- Providing `ref` for `issue` or `pull_request`
- Providing `number` for `issue_list` or `pull_request_list`
- Providing `ref` for `issue_list` or `pull_request_list`
- `limit < 1` or `limit > 100` for `issue_list` or `pull_request_list`
- Missing `ref` for `commit`
- Providing `number` for `commit`
- Missing `ref` for `commit_history`
- Providing `number` for `commit_history`
- `limit < 1` or `limit > 100` for `commit_history`
- `alias` does not match any `github.by_base_url[].alias`
- `base_url` and `alias` are both provided but point to different `github.by_base_url` entries
- The GitHub API returns a 4xx or 5xx response
