---
name: my-slack
description: Use the `my-slack` CLI to call Slack Web API create, read, update, delete, and list flows with one JSON request and a normalized JSON response.
---

# My Slack

This skill uses the `my-slack` binary to call Slack Web API methods from a single JSON request. Prefer it over ad hoc REST calls when you want predictable JSON input, `--dry-run` support, config-based token selection, and cursor-aware list pagination.

In Codex CLI installs, the binary lives under `${CODEX_HOME}/bin/my-slack` and the config file lives under `${HOME}/.config/my-slack.yaml`. If it is not on `PATH`, use the provided absolute binary path.

## When to use

Use this skill when:

- Creating Slack resources such as channels or messages
- Reading Slack resources such as channel info
- Updating Slack resources such as channel names or messages
- Deleting or archiving Slack resources
- Listing cursor-based Slack resources such as channels, users, members, or messages
- Working in an agent or CLI workflow where one JSON request is easier than composing HTTP calls manually

## Quick workflow

1. Pass exactly one JSON object as a CLI argument or through `stdin`.
2. Use `--dry-run` first when the request shape is uncertain.
3. Prefer canonical kinds such as `create`, `read`, `update`, `delete`, and `list`.
4. For list methods, add `list_field` only when the response contains multiple array candidates.

## Input

Provide exactly one JSON object.

Common required fields:

- `kind`
- `method`

Common optional fields:

- `args` for Slack method arguments
- `base_url` to choose a specific Slack API base URL for this request
- `alias` to choose a configured `slack.workspaces[].alias` entry
- `http_method` to force `GET` or `POST`

List-specific fields:

- `limit` for the maximum number of items to collect
- `cursor` for the starting cursor
- `list_field` for the array field to aggregate, such as `channels`, `messages`, or `members`

Supported `kind` values:

- `create`
- `read`
- `update`
- `delete`
- `list`

Accepted aliases:

- `post` -> `create`
- `get` -> `read`
- `put`, `patch` -> `update`
- `remove` -> `delete`
- `ls` -> `list`

Validation rules:

- Unknown fields are errors.
- `method` must look like a Slack Web API method such as `conversations.list`.
- `limit`, `cursor`, and `list_field` are only valid for `kind=list`.
- `limit` must be between 1 and 1000 when provided.
- If both `limit` and `args.limit` are provided, they must match.
- If both `cursor` and `args.cursor` are provided, they must match.
- If both `base_url` and `alias` are provided, they must point to the same configured workspace when that workspace defines a `base_url`.

Example inputs:

```json
{"kind":"create","method":"conversations.create","args":{"name":"eng-bot-playground"}}
```

```json
{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}
```

```json
{"kind":"update","method":"conversations.rename","args":{"channel":"C12345678","name":"eng-platform"}}
```

```json
{"kind":"delete","method":"conversations.archive","args":{"channel":"C12345678"}}
```

```json
{"kind":"list","method":"conversations.list","limit":50,"args":{"types":"public_channel,private_channel"}}
```

```json
{"kind":"list","method":"conversations.history","limit":20,"list_field":"messages","args":{"channel":"C12345678"}}
```

```json
{"kind":"create","method":"chat.postMessage","args":{"channel":"C12345678","text":"hello from my-slack"}}
```

## Output

Successful responses always include:

- `kind`
- `method`
- `response`

List requests also return:

- `list.field`
- `list.limit`
- `list.count`
- `list.next_cursor`
- `list.items`

`response` keeps the Slack response shape as much as possible. For list requests, the aggregated array replaces the original array field in `response`.

Example output shape:

```json
{
  "kind": "read",
  "method": "conversations.info",
  "response": {
    "ok": true,
    "channel": {
      "id": "C12345678",
      "name": "eng-platform"
    }
  }
}
```

## Command examples

If the binary is not on `PATH`, replace `my-slack` with the provided absolute path.

```bash
my-slack '{"kind":"create","method":"conversations.create","args":{"name":"eng-bot-playground"}}'
```

```bash
my-slack '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'
```

```bash
my-slack '{"kind":"update","method":"chat.update","args":{"channel":"C12345678","ts":"1710000000.000100","text":"updated text"}}'
```

```bash
my-slack '{"kind":"delete","method":"chat.delete","args":{"channel":"C12345678","ts":"1710000000.000100"}}'
```

```bash
my-slack '{"kind":"list","method":"users.list","limit":200,"alias":"workspace-prod"}'
```

```bash
my-slack --dry-run '{"kind":"list","method":"conversations.history","limit":20,"list_field":"messages","args":{"channel":"C12345678"}}'
```

```bash
echo '{"kind":"create","method":"chat.postMessage","args":{"channel":"C12345678","text":"hello from my-slack"}}' | my-slack
```

## Flags

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Failure prevention

- Use `stdin` if shell escaping is awkward.
- Start with `--dry-run` before real calls when the request is uncertain.
- For list methods with multiple top-level arrays, set `list_field` explicitly.
- If you need build, test, or lint instructions, read [README.md](../../src/cmd/my-slack/README.md).
