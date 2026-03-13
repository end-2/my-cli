# my-discord

`my-discord` is a single-purpose CLI that sends create, read, update, delete, and list style requests to the Discord REST API through JSON input/output.  
It accepts exactly one JSON object as input and prints JSON as output.

This command is part of the `my-cli` project, and all build, test, and lint tasks run inside Docker containers.

For rules on using the `my-discord` binary in LLM / agent environments, see [docs/my-discord/SKILL.md](../../../docs/my-discord/SKILL.md).

## Requirements

- Docker
- GNU Make

You do not need to install Go or `golangci-lint` locally.

## Build

Build from the repository root like this.

```bash
make build my-discord
make build CMD=my-discord
```

Common option examples:

```bash
make build my-discord VERSION=1.0.0
make build my-discord GO_VERSION=1.26.1
make build my-discord CGO_ENABLED=1
make build my-discord GOOS=linux GOARCH=amd64
make build my-discord GOPRIVATE='github.com/your-org/*'
```

The output binary is created at `bin/my-discord`.  
During the build, `ldflags` injects a `VERSION-git_commit` value into `main.Version`.

```bash
./bin/my-discord --help
./bin/my-discord --dry-run '{"kind":"read","path":"/channels/123"}'
./bin/my-discord --version
./bin/my-discord -version
```

## Install into Codex

To use it directly from Codex CLI, run the following script from the repository root.

```bash
./scripts/install-my-discord-codex.sh
```

This script runs `make build CMD=my-discord`, then updates both `~/.codex/bin/my-discord` and `~/.codex/skills/my-discord/*`.  
If you use a different Codex home, run it like `CODEX_HOME=/path/to/codex ./scripts/install-my-discord-codex.sh`.

## Test

Tests run inside the `golang:<GO_VERSION>` Docker image.

```bash
make test my-discord
make test CMD=my-discord
```

You can pass extra options as well.

```bash
make test my-discord TEST_FLAGS="-v"
make test my-discord TEST_FLAGS="-run TestRootCommandFetchesListWithAfterPagination -v"
```

## Lint

Linting uses `golangci-lint` inside Docker. The default image is `golangci/golangci-lint:v2.9.0`.

```bash
make lint my-discord
make lint CMD=my-discord
```

Additional option examples:

```bash
make lint my-discord LINT_FLAGS="--verbose"
make lint my-discord LINT_TIMEOUT=10m
make lint my-discord GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

```bash
make list-cmds
make print-version
make clean
```

## Configuration File

The configuration file name is `my-discord.yaml`.  
Settings are loaded through [`src/pkg/config/config.go`](../../pkg/config/config.go).

The search paths are checked in this order.

1. `/etc/my-discord/my-discord.yaml`
2. `~/.config/my-discord.yaml`
3. `./my-discord.yaml`

Existing files are merged in that order, and later values override earlier ones.  
If no configuration file exists, the following defaults are used.

- `discord.base_url`: `https://discord.com/api/v10/`
- `discord.timeout`: `15s`
- `discord.user_agent`: `DiscordBot (https://github.com/end-2/my-cli, 1.0) my-cli/my-discord`
- `discord.token_type`: `Bot`
- `discord.token`: empty
- `discord.bots`: empty

The top-level `discord` values are the shared defaults.  
If you include `base_url` in the request JSON, the base URL is changed only for that request.  
If you include `alias`, a specific bot configuration from `discord.bots[]` is selected.

Example:

```yaml
discord:
  base_url: https://discord.com/api/v10/
  timeout: 15s
  user_agent: DiscordBot (https://github.com/end-2/my-cli, 1.0) my-cli/my-discord
  token_type: Bot
  bots:
    - alias: bot-dev
      token: "{{ .DISCORD_DEV_BOT_TOKEN }}"
    - alias: bot-prod
      token: "{{ .DISCORD_PROD_BOT_TOKEN }}"
      timeout: 30s
      user_agent: DiscordBot (https://github.com/end-2/my-cli, 1.0) my-cli/my-discord-prod
```

Secret values such as `token` can also be managed through config templates.  
In the example above, the runtime environment variables `DISCORD_DEV_BOT_TOKEN` and `DISCORD_PROD_BOT_TOKEN` are read and injected into the token for each bot.

## Usage

Pass JSON input in one of the following ways.

```bash
./bin/my-discord '{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from my-discord"}}'
```

```bash
./bin/my-discord '{"kind":"read","path":"/channels/123"}'
```

```bash
./bin/my-discord '{"kind":"update","path":"/channels/123","body":{"name":"eng-platform"},"reason":"rename channel"}'
```

```bash
./bin/my-discord '{"kind":"delete","path":"/channels/123/messages/456"}'
```

```bash
./bin/my-discord '{"kind":"list","path":"/channels/123/messages","limit":150,"before":"145000000000000002"}'
```

```bash
./bin/my-discord '{"kind":"list","path":"/guilds/123/members","limit":200,"page_limit":100,"after":"0","cursor_field":"user.id"}'
```

```bash
./bin/my-discord '{"kind":"list","path":"/guilds/123/audit-logs","limit":50,"list_field":"audit_log_entries","query":{"action_type":10}}'
```

```bash
echo '{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from stdin"}}' | ./bin/my-discord
```

Supported flags:

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Common JSON Input Rules

- Only one JSON object is allowed as input.
- At most one argument may be provided.
- Unknown fields are treated as errors.
- `kind` and `path` are always required.
- `query` is the query string object.
- `body` is the JSON body object.
- `base_url` and `alias` are optional.
- If authentication is required, set `discord.token` or the selected `discord.bots[].token` in `my-discord.yaml`.
- `token_type` must be `Bot` or `Bearer`.
- If `reason` is provided, it is sent in Discord's `X-Audit-Log-Reason` header.
- Multipart or file upload is not currently supported; only JSON bodies are supported.

## Common Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kind` | string | Yes | Request type: `create`, `read`, `update`, `delete`, `list` |
| `path` | string | Yes | Relative path under the Discord API base URL, for example `/channels/123/messages` |
| `query` | object | No | Query string object |
| `body` | object | No | JSON body object |
| `http_method` | string | No | `GET`, `POST`, `PUT`, `PATCH`, or `DELETE` |
| `reason` | string | No | Value to send in the `X-Audit-Log-Reason` header |
| `base_url` | string | No | Discord API base URL to use for this request |
| `alias` | string | No | `discord.bots[].alias` value to use for this request |

## `list`-Only Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `limit` | integer | Conditional | Maximum total number of items to collect. Must be between 1 and 1000, and defaults to 100 |
| `page_limit` | integer | No | `limit` value to send on each Discord API call. Defaults to 100 |
| `before` | string | No | Starting cursor for `before`-based pagination |
| `after` | string | No | Starting cursor for `after`-based pagination |
| `list_field` | string | No | Array field name to aggregate when the response is an object, for example `audit_log_entries` |
| `cursor_field` | string | No | Item field path used to extract the next-page cursor. Defaults to `id`, for example `user.id` |

## kind Values

| Value | Description |
| --- | --- |
| `create` | Discord create request |
| `read` | Discord read request |
| `update` | Discord update request |
| `delete` | Discord delete request |
| `list` | Discord list request with `before`- or `after`-based pagination |

The following aliases are also accepted.

- `post` -> `create`
- `get` -> `read`
- `put`, `patch` -> `update`
- `remove` -> `delete`
- `ls` -> `list`

## How `list` Works

When `kind=list`, the CLI collects items if the Discord response is an array or an object containing an array field.  
The default pagination direction is `before`. If `after` is provided, the next page is requested using `after`.

- If the response is a top-level array, the items are merged directly.
- If the response is an object with exactly one array field, that field is inferred automatically.
- If the response contains multiple array fields, you must specify `list_field`.
- `cursor_field` is used to extract the next cursor from each item. If omitted, `id` is used.

## Response Shape

On success, the response always includes the following fields.

- `kind`
- `path`
- `http_method`
- `response`

When `kind=list`, a `list` object is added as well.

Example:

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
