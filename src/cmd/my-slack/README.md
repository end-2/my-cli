# my-slack

`my-slack` is a single-purpose CLI that sends create, read, update, delete, and list style requests to the Slack Web API through JSON input/output.  
It accepts exactly one JSON object as input and prints JSON as output.

This command is part of the `my-cli` project, and all build, test, and lint tasks run inside Docker containers.

For rules on using the `my-slack` binary in LLM / agent environments, see [docs/my-slack/SKILL.md](../../../docs/my-slack/SKILL.md).

## Requirements

- Docker
- GNU Make

You do not need to install Go or `golangci-lint` locally.

## Build

Build from the repository root like this.

```bash
make build my-slack
make build CMD=my-slack
```

Common option examples:

```bash
make build my-slack VERSION=1.0.0
make build my-slack GO_VERSION=1.26.1
make build my-slack CGO_ENABLED=1
make build my-slack GOOS=linux GOARCH=amd64
make build my-slack GOPRIVATE='github.com/your-org/*'
```

The output binary is created at `bin/my-slack`.  
During the build, `ldflags` injects a `VERSION-git_commit` value into `main.Version`.

```bash
./bin/my-slack --help
./bin/my-slack --dry-run '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'
./bin/my-slack --version
./bin/my-slack -version
```

## Install into Codex

To use it directly from Codex CLI, run the following script from the repository root.

```bash
./scripts/install-my-slack-codex.sh
```

This script runs `make build CMD=my-slack`, then updates both `~/.codex/bin/my-slack` and `~/.codex/skills/my-slack/*`.  
If you use a different Codex home, run it like `CODEX_HOME=/path/to/codex ./scripts/install-my-slack-codex.sh`.

## Test

Tests run inside the `golang:<GO_VERSION>` Docker image.

```bash
make test my-slack
make test CMD=my-slack
```

You can pass extra options as well.

```bash
make test my-slack TEST_FLAGS="-v"
make test my-slack TEST_FLAGS="-run TestRootCommandFetchesListWithPagination -v"
```

## Lint

Linting uses `golangci-lint` inside Docker. The default image is `golangci/golangci-lint:v2.9.0`.

```bash
make lint my-slack
make lint CMD=my-slack
```

Additional option examples:

```bash
make lint my-slack LINT_FLAGS="--verbose"
make lint my-slack LINT_TIMEOUT=10m
make lint my-slack GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

```bash
make list-cmds
make print-version
make clean
```

## Configuration File

The configuration file name is `my-slack.yaml`.  
Settings are loaded through [`src/pkg/config/config.go`](../../pkg/config/config.go).

The search paths are checked in this order.

1. `/etc/my-slack/my-slack.yaml`
2. `~/.config/my-slack.yaml`
3. `./my-slack.yaml`

Existing files are merged in that order, and later values override earlier ones.  
If no configuration file exists, the following defaults are used.

- `slack.base_url`: `https://slack.com/api/`
- `slack.timeout`: `15s`
- `slack.user_agent`: `my-cli/my-slack`
- `slack.token`: empty
- `slack.workspaces`: empty

The top-level `slack` values are the shared defaults.  
If you include `base_url` in the request JSON, the base URL is changed only for that request.  
If you include `alias`, a specific workspace configuration from `slack.workspaces[]` is selected.

Example:

```yaml
slack:
  base_url: https://slack.com/api/
  timeout: 15s
  user_agent: my-cli/my-slack
  workspaces:
    - alias: workspace-dev
      token: "{{ .SLACK_DEV_BOT_TOKEN }}"
    - alias: workspace-prod
      token: "{{ .SLACK_PROD_BOT_TOKEN }}"
      timeout: 30s
      user_agent: my-cli/my-slack-prod
```

Secret values such as `token` can also be managed through config templates.  
In the example above, the runtime environment variables `SLACK_DEV_BOT_TOKEN` and `SLACK_PROD_BOT_TOKEN` are read and injected into the token for each workspace.

## Usage

Pass JSON input in one of the following ways.

```bash
./bin/my-slack '{"kind":"create","method":"conversations.create","args":{"name":"eng-bot-playground"}}'
```

```bash
./bin/my-slack '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'
```

```bash
./bin/my-slack '{"kind":"update","method":"conversations.rename","args":{"channel":"C12345678","name":"eng-platform"}}'
```

```bash
./bin/my-slack '{"kind":"delete","method":"conversations.archive","args":{"channel":"C12345678"}}'
```

```bash
./bin/my-slack '{"kind":"list","method":"conversations.list","limit":50,"args":{"types":"public_channel,private_channel"}}'
```

```bash
echo '{"kind":"create","method":"chat.postMessage","args":{"channel":"C12345678","text":"hello from my-slack"}}' | ./bin/my-slack
```

```bash
./bin/my-slack '{"kind":"list","method":"conversations.history","limit":20,"list_field":"messages","args":{"channel":"C12345678"}}'
```

```bash
./bin/my-slack '{"kind":"list","method":"users.list","limit":200,"alias":"workspace-prod"}'
```

Supported flags:

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Common JSON Input Rules

- Only one JSON object is allowed as input.
- At most one argument may be provided.
- Unknown fields are treated as errors.
- `kind` and `method` are always required.
- `args` is the Slack method argument object.
- `base_url` and `alias` are optional.
- If authentication is required, set `slack.token` or the selected `slack.workspaces[].token` in `my-slack.yaml`.
- If you need environment-variable-based secret injection, use a template such as `{{ .SLACK_BOT_TOKEN }}` in `slack.token` or `slack.workspaces[].token`.
- `http_method` is optional and only `GET` or `POST` is allowed.

## Common Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kind` | string | Yes | Request type: `create`, `read`, `update`, `delete`, `list` |
| `method` | string | Yes | Slack Web API method name, for example `conversations.list` |
| `args` | object | No | Slack method argument object |
| `limit` | integer | Conditional | Maximum number of items to collect for `list`. Must be between 1 and 1000, and defaults to 100 |
| `cursor` | string | No | Starting cursor for `list` |
| `list_field` | string | No | Use this when you want to explicitly set the response array field for `list`, for example `channels`, `messages`, or `members` |
| `http_method` | string | No | `GET` or `POST`. Defaults to `POST` for `create` / `update` / `delete`, and `GET` for `read` / `list` |
| `base_url` | string | No | Slack API base URL to use for this request |
| `alias` | string | No | `slack.workspaces[].alias` value to use for this request |

## kind Values

| Value | Description |
| --- | --- |
| `create` | Slack create/write request |
| `read` | Slack read request |
| `update` | Slack update request |
| `delete` | Slack delete request |
| `list` | Slack list request with automatic cursor-based pagination |

The following aliases are also accepted.

- `post` -> `create`
- `get` -> `read`
- `put`, `patch` -> `update`
- `remove` -> `delete`
- `ls` -> `list`

## Representative Method Examples

- `create`: `conversations.create`, `chat.postMessage`
- `read`: `conversations.info`, `auth.test`
- `update`: `conversations.rename`, `chat.update`
- `delete`: `conversations.archive`, `chat.delete`
- `list`: `conversations.list`, `users.list`, `conversations.history`, `conversations.members`, `conversations.replies`

## How `list` Works

When `kind=list`, the CLI follows `response_metadata.next_cursor` and collects items up to the requested `limit`.  
The response includes the following two shapes together.

- `list`: the field name, requested limit, count, next cursor, and merged items
- `response`: preserves the original Slack response shape as much as possible, but replaces the list field array with the merged result

If the response array field cannot be detected automatically, specify `list_field`.

## Response Shape

On success, the response always includes the following fields.

- `kind`
- `method`
- `response`

When `kind=list`, a `list` object is added as well.

Example:

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

## References

- Slack Web API overview: <https://api.slack.com/web>
- Method reference: <https://docs.slack.dev/reference/methods/>
