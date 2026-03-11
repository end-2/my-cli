---
name: my-github
description: Use the `my-github` CLI to fetch GitHub issue, pull request, or commit data with one JSON request and a normalized JSON response. Use this skill when the binary is already available and you want GitHub data without hand-writing REST API calls.
---

# My GitHub

This skill uses the `my-github` binary to query GitHub resources from the GitHub REST API. Prefer it over direct API calls when you need a single issue, pull request, or commit in a predictable JSON shape.

## When to use

Use this skill when:

- Fetching a GitHub issue
- Fetching a GitHub pull request
- Fetching a GitHub commit by SHA, branch, or tag
- Working in an agent or CLI workflow where a single JSON request is easier than composing REST calls manually

## Prerequisites

- Assume the `my-github` binary already exists.
- If it is on `PATH`, run `my-github`.
- If it is not on `PATH`, use the provided absolute binary path.
- Do not build from source unless the user explicitly asks.
- If you need build, test, or lint instructions, read [README.md](../../src/cmd/my-github/README.md).

## Quick workflow

1. Confirm the binary location with `command -v my-github` if needed.
2. Create `my-github.yaml` only when you need a token, GitHub Enterprise base URL, or custom timeout.
3. Pass exactly one JSON object as a CLI argument or through `stdin`.
4. Use `--dry-run` first when the request shape or config is uncertain.
5. Prefer `pull_request` over aliases in new requests.

## Input

Provide exactly one JSON object.

Common required fields:

- `kind`
- `owner`
- `repo`

Resource-specific fields:

- `number` for `issue` and `pull_request`
- `ref` for `commit`

Supported `kind` values:

- `issue`
- `pull_request`
- `commit`

Accepted pull request aliases:

- `pr`
- `pull-request`

Validation rules:

- Do not send both `number` and `ref`.
- Unknown fields are errors.
- `number` is required for `issue` and `pull_request`.
- `ref` is required for `commit`.

Example inputs:

```json
{"kind":"issue","owner":"cli","repo":"cli","number":123}
```

```json
{"kind":"pull_request","owner":"cli","repo":"cli","number":456}
```

```json
{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}
```

## Output

Successful responses always include:

- `kind`
- `repository`

Resource-specific payloads:

- Issue requests return an `issue` object.
- Pull request requests return a `pull_request` object.
- Commit requests return a `commit` object.
- `--dry-run` returns the planned request without calling GitHub.

Example output shape:

```json
{
  "kind": "issue",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "issue": {
    "number": 123,
    "title": "Issue title"
  }
}
```

## Command examples

```bash
my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
```

```bash
echo '{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}' | my-github
```

```bash
my-github --dry-run '{"kind":"pull_request","owner":"cli","repo":"cli","number":456}'
```

If the binary is not on `PATH`, replace `my-github` with the provided absolute path.

## Configuration

Use `my-github.yaml` only when defaults are not enough.

Search order:

1. `/etc/my-github/my-github.yaml`
2. `~/my-github.yaml`
3. `./my-github.yaml`

Recommended rules:

- Keep tokens in config, not in CLI arguments.
- Use template-based secret injection such as `{{ .GITHUB_TOKEN }}`.
- Set `github.base_url` for GitHub Enterprise.
- Use the example file at [my-github-example.yaml](./my-github-example.yaml) when creating config.

Example:

```yaml
github:
  base_url: https://api.github.com/
  timeout: 15s
  user_agent: my-cli/my-github
  token: "{{ .GITHUB_TOKEN }}"
```

## Flags

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Failure prevention

- Use `stdin` if shell escaping is awkward.
- Assume private repositories require a token.
- Start with `--dry-run` before real calls when the request is uncertain.
