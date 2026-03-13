---
name: my-discord
description: Use the `my-discord` CLI to call Discord REST API create, read, update, delete, and list flows with one JSON request and a normalized JSON response.
---

# My Discord

This skill uses the `my-discord` binary to call Discord REST API routes from a single JSON request. Prefer it over ad hoc HTTP calls when you want predictable JSON input, `--dry-run` support, config-based token selection, and route-aware list pagination with `before` or `after`.

In Codex CLI installs, the binary lives under `${CODEX_HOME}/bin/my-discord` and the config file lives under `${HOME}/.config/my-discord.yaml`. If it is not on `PATH`, use the provided absolute binary path.

## When to use

Use this skill when:

- Creating Discord resources such as messages or channels
- Reading Discord resources such as channels, messages, or guild metadata
- Updating Discord resources such as channel names or permissions
- Deleting Discord resources such as messages
- Listing Discord resources such as channel messages, guild members, or audit log entries
- Working in an agent or CLI workflow where one JSON request is easier than composing HTTP calls manually

## Quick workflow

1. Pass exactly one JSON object as a CLI argument or through `stdin`.
2. Use `--dry-run` first when the request shape is uncertain.
3. Prefer canonical kinds such as `create`, `read`, `update`, `delete`, and `list`.
4. For list routes that return multiple arrays, add `list_field`.
5. For list routes whose cursor is nested, add `cursor_field`, for example `user.id`.

## Input

Provide exactly one JSON object.

Common required fields:

- `kind`
- `path`

Common optional fields:

- `query` for query parameters
- `body` for JSON request bodies
- `reason` for the `X-Audit-Log-Reason` header
- `base_url` to choose a specific Discord API base URL for this request
- `alias` to choose a configured `discord.bots[].alias` entry
- `http_method` to force `GET`, `POST`, `PUT`, `PATCH`, or `DELETE`

List-specific fields:

- `limit` for the total number of items to collect
- `page_limit` for the per-request Discord `limit` value
- `before` or `after` for pagination
- `list_field` for object responses with one or more array fields
- `cursor_field` for the field used to derive the next cursor, such as `id` or `user.id`

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
- `path` must be relative to the Discord API base URL.
- `body` is not allowed for `GET` requests.
- `limit`, `page_limit`, `before`, `after`, `list_field`, and `cursor_field` are only valid for `kind=list`.
- `limit` must be between 1 and 1000 when provided.
- `page_limit` must be between 1 and 1000 when provided.
- `before` and `after` cannot be used together.
- `query.limit`, `query.before`, and `query.after` are reserved for list routes and must be set through top-level fields instead.
- If both `base_url` and `alias` are provided, they must point to the same configured bot when that bot defines a `base_url`.

Example inputs:

```json
{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from my-discord"}}
```

```json
{"kind":"read","path":"/channels/123"}
```

```json
{"kind":"update","path":"/channels/123","body":{"name":"eng-platform"},"reason":"rename channel"}
```

```json
{"kind":"delete","path":"/channels/123/messages/456"}
```

```json
{"kind":"list","path":"/channels/123/messages","limit":150,"before":"145000000000000002"}
```

```json
{"kind":"list","path":"/guilds/123/members","limit":200,"page_limit":100,"after":"0","cursor_field":"user.id"}
```

```json
{"kind":"list","path":"/guilds/123/audit-logs","limit":50,"list_field":"audit_log_entries","query":{"action_type":10}}
```

## Output

Successful responses always include:

- `kind`
- `path`
- `http_method`
- `response`

List requests also return:

- `list.field` when the response is an object-backed list
- `list.cursor_field`
- `list.pagination`
- `list.limit`
- `list.count`
- `list.next_cursor`
- `list.items`

`response` keeps the Discord response shape as much as possible. For list requests, the aggregated array replaces the original array field or becomes the top-level response when the endpoint returns an array.

Example output shape:

```json
{
  "kind": "read",
  "path": "/channels/123",
  "http_method": "GET",
  "response": {
    "id": "123",
    "name": "eng-platform"
  }
}
```

## Command examples

If the binary is not on `PATH`, replace `my-discord` with the provided absolute path.

```bash
my-discord '{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from my-discord"}}'
```

```bash
my-discord '{"kind":"read","path":"/channels/123"}'
```

```bash
my-discord '{"kind":"update","path":"/channels/123","body":{"name":"eng-platform"},"reason":"rename channel"}'
```

```bash
my-discord '{"kind":"delete","path":"/channels/123/messages/456"}'
```

```bash
my-discord '{"kind":"list","path":"/guilds/123/members","limit":200,"page_limit":100,"after":"0","cursor_field":"user.id"}'
```

```bash
my-discord --dry-run '{"kind":"list","path":"/guilds/123/audit-logs","limit":50,"list_field":"audit_log_entries","query":{"action_type":10}}'
```

```bash
echo '{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from stdin"}}' | my-discord
```

## Flags

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Failure prevention

- Use `stdin` if shell escaping is awkward.
- Start with `--dry-run` before real calls when the request is uncertain.
- For object responses with multiple array fields, set `list_field` explicitly.
- For nested cursor values such as guild members, set `cursor_field`.
- If you need build, test, or lint instructions, read [README.md](../../src/cmd/my-discord/README.md).
