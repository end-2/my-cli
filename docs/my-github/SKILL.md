---
name: my-github
description: Use the `my-github` CLI to fetch GitHub issue, pull request, commit, or commit history data with one JSON request and a normalized JSON response. Use this skill when the binary is already available and you want GitHub data without hand-writing REST API calls.
---

# My GitHub

This skill uses the `my-github` binary to query GitHub resources from the GitHub REST API. Prefer it over direct API calls when you need an issue, pull request, commit, or branch commit history in a predictable JSON shape.

In Codex CLI installs, the binary lives under `${CODEX_HOME}/bin/my-github` and the config file lives under `${CODEX_HOME}/bin/my-github.yaml`. If it is not on `PATH`, use the provided absolute binary path.

## When to use

Use this skill when:

- Fetching a GitHub issue
- Fetching a GitHub pull request
- Fetching a GitHub commit by SHA, branch, or tag
- Fetching commit history for a specific branch or ref
- Working in an agent or CLI workflow where a single JSON request is easier than composing REST calls manually

## Quick workflow

1. Pass exactly one JSON object as a CLI argument or through `stdin`.
2. Use `--dry-run` first when the request shape is uncertain.
3. Prefer canonical kinds such as `pull_request` and `commit_history` over aliases in new requests.

## Input

Provide exactly one JSON object.

Common required fields:

- `kind`
- `owner`
- `repo`

Resource-specific fields:

- `number` for `issue` and `pull_request`
- `ref` for `commit` and `commit_history`
- `limit` for `commit_history`

Supported `kind` values:

- `issue`
- `pull_request`
- `commit`
- `commit_history`

Accepted pull request aliases:

- `pr`
- `pull-request`

Accepted commit history aliases:

- `commit-history`

Validation rules:

- Do not send both `number` and `ref`.
- Unknown fields are errors.
- `number` is required for `issue` and `pull_request`.
- `ref` is required for `commit`.
- `ref` is required for `commit_history`.
- `limit` is optional for `commit_history`, but when provided it must be between 1 and 100.

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

```json
{"kind":"commit_history","owner":"cli","repo":"cli","ref":"release/1.0","limit":10}
```

## Output

Successful responses always include:

- `kind`
- `repository`

Resource-specific payloads:

- Issue requests return an `issue` object.
- Pull request requests return a `pull_request` object.
- Commit requests return a `commit` object. Single commit lookups may include `stats` and per-file `files` changes.
- Commit history requests return a `commit_history` object.
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

If the binary is not on `PATH`, replace `my-github` with the provided absolute path.

```bash
my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
```

```bash
echo '{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}' | my-github
```

```bash
my-github '{"kind":"commit_history","owner":"cli","repo":"cli","ref":"release/1.0","limit":10}'
```

```bash
my-github --dry-run '{"kind":"pull_request","owner":"cli","repo":"cli","number":456}'
```

## Flags

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Failure prevention

- Use `stdin` if shell escaping is awkward.
- Start with `--dry-run` before real calls when the request is uncertain.
