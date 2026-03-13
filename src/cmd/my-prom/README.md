# my-prom

`my-prom` is a single-purpose CLI that sends instant query, range query, series lookup, label name lookup, and label value lookup requests to the Prometheus HTTP API through JSON input/output.  
It accepts exactly one JSON object as input and prints JSON as output.

This command is part of the `my-cli` project, and all build, test, and lint tasks run inside Docker containers.

For rules on using the `my-prom` binary in LLM / agent environments, see [docs/my-prom/SKILL.md](../../../docs/my-prom/SKILL.md).

## Requirements

- Docker
- GNU Make

You do not need to install Go or `golangci-lint` locally.

## Build

Build from the repository root like this.

```bash
make build my-prom
make build CMD=my-prom
```

Common option examples:

```bash
make build my-prom VERSION=1.0.0
make build my-prom GO_VERSION=1.26.1
make build my-prom CGO_ENABLED=1
make build my-prom GOOS=linux GOARCH=amd64
make build my-prom GOPRIVATE='github.com/your-org/*'
```

The output binary is created at `bin/my-prom`.  
During the build, `ldflags` injects a `VERSION-git_commit` value into `main.Version`.

```bash
./bin/my-prom --help
./bin/my-prom --dry-run '{"kind":"query","query":"up"}'
./bin/my-prom --version
./bin/my-prom -version
```

## Install into Codex

To use it directly from Codex CLI, run the following script from the repository root.

```bash
./scripts/install-my-prom-codex.sh
```

This script runs `make build CMD=my-prom`, then updates both `~/.codex/bin/my-prom` and `~/.codex/skills/my-prom/*`.  
If you use a different Codex home, run it like `CODEX_HOME=/path/to/codex ./scripts/install-my-prom-codex.sh`.

## Test

Tests run inside the `golang:<GO_VERSION>` Docker image.

```bash
make test my-prom
make test CMD=my-prom
```

You can pass extra options as well.

```bash
make test my-prom TEST_FLAGS="-v"
make test my-prom TEST_FLAGS="-run TestRootCommandFetchesInstantQuery -v"
```

## Lint

Linting uses `golangci-lint` inside Docker. The default image is `golangci/golangci-lint:v2.9.0`.

```bash
make lint my-prom
make lint CMD=my-prom
```

Additional option examples:

```bash
make lint my-prom LINT_FLAGS="--verbose"
make lint my-prom LINT_TIMEOUT=10m
make lint my-prom GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

```bash
make list-cmds
make print-version
make clean
```

## Configuration File

The configuration file name is `my-prom.yaml`.  
Settings are loaded through [`src/pkg/config/config.go`](../../pkg/config/config.go).

The search paths are checked in this order.

1. `/etc/my-prom/my-prom.yaml`
2. `~/.config/my-prom.yaml`
3. `./my-prom.yaml`

Existing files are merged in that order, and later values override earlier ones.  
If no configuration file exists, the following defaults are used.

- `prometheus.base_url`: `http://localhost:9090/`
- `prometheus.timeout`: `15s`
- `prometheus.user_agent`: `my-cli/my-prom`
- `prometheus.token`: empty
- `prometheus.instances`: empty

The top-level `prometheus` values are the shared defaults.  
Selection starts from `prometheus.base_url`, and if the request JSON includes `base_url` or `alias`, that choice takes precedence.  
If there is a `prometheus.instances[]` entry that matches the selected Prometheus instance, its `base_url`, `token`, `timeout`, and `user_agent` values are applied last.  
`prometheus.instances[].alias` can be selected directly through the request JSON `alias` field, and `base_url` matching ignores a trailing `/`.

Example:

```yaml
prometheus:
  base_url: http://localhost:9090/
  timeout: 15s
  user_agent: my-cli/my-prom
  instances:
    - alias: local-prom
      base_url: http://localhost:9090/
    - alias: prod-prom
      base_url: https://prom.example.com/
      token: "{{ .PROM_PROD_TOKEN }}"
      timeout: 30s
      user_agent: my-cli/my-prom-prod
```

Secret values such as `token` can also be managed through config templates.  
In the example above, the runtime environment variable `PROM_PROD_TOKEN` is read and injected into the token for the production Prometheus instance.

## Usage

Pass JSON input in one of the following ways.

```bash
./bin/my-prom '{"kind":"query","query":"up"}'
```

```bash
./bin/my-prom '{"kind":"query_range","query":"rate(http_requests_total[5m])","start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z","step":"5m"}'
```

```bash
./bin/my-prom '{"kind":"series","matchers":["up{job=\"node\"}"],"start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z"}'
```

```bash
./bin/my-prom '{"kind":"label_values","label":"__name__","limit":100}'
```

```bash
./bin/my-prom '{"kind":"label_names","matchers":["{job=\"api\"}"],"alias":"prod-prom"}'
```

```bash
echo '{"kind":"query","query":"sum(rate(container_cpu_usage_seconds_total[5m])) by (pod)","http_method":"POST"}' | ./bin/my-prom
```

Supported flags:

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Common JSON Input Rules

- Only one JSON object is allowed as input.
- At most one argument may be provided.
- Unknown fields are treated as errors.
- `kind` is always required.
- `base_url` and `alias` are optional.
- If authentication is required, set `prometheus.token` or the selected `prometheus.instances[].token` in `my-prom.yaml`.
- If you need environment-variable-based secret injection, use a template such as `{{ .PROM_TOKEN }}` in `prometheus.token` or `prometheus.instances[].token`.
- `http_method` is optional and only `GET` or `POST` is allowed.
- `label_values` only supports `GET`.
- `limit` follows the Prometheus API meaning, and `0` disables the limit.

## Common Fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kind` | string | Yes | Request type |
| `query` | string | Conditional | PromQL used for `query` and `query_range` |
| `time` | string | No | Evaluation time for `query` |
| `start` | string | Conditional | Required for `query_range`, optional for some other kinds |
| `end` | string | Conditional | Required for `query_range`, optional for some other kinds |
| `step` | string | Conditional | Required for `query_range` |
| `timeout` | string | No | API timeout for `query` and `query_range` |
| `limit` | integer | No | Limit on returned series or items. `0` disables the limit |
| `matchers` | array[string] | Conditional | List of `match[]` values used for `series`, `label_names`, and `label_values` |
| `label` | string | Conditional | Label name to fetch in `label_values`, for example `job` or `__name__` |
| `http_method` | string | No | `GET` or `POST` |
| `base_url` | string | No | Prometheus API base URL to use for this request |
| `alias` | string | No | `prometheus.instances[].alias` value to use for this request |

## kind Values

| Value | Description |
| --- | --- |
| `query` | Instant query |
| `query_range` | Range query |
| `series` | Fetch the series list for `match[]` conditions |
| `label_names` | Fetch label names |
| `label_values` | Fetch values for a specific label |

The following aliases are also accepted.

- `instant`
- `instant-query`
- `instant_query`
- `range`
- `range-query`
- `range_query`
- `label-names`
- `labels`
- `label-values`

## Response Shape

On success, the response has the following common characteristics.

- `kind`
- Echoed key request fields
- `count`
- `warnings`
- `infos`

`query` and `query_range` additionally return the following fields.

- `result_type`
- `result`

Sample values are normalized like this so they are easier for agents to work with.

```json
{
  "timestamp": "2024-03-12T22:40:00.5Z",
  "timestamp_unix": 1710283200.5,
  "value": "1"
}
```

For vector and matrix results, the metric label map is preserved while value or value lists are wrapped in the shape above.

## AI / Agent Usage Tips

- If you do not know the metric name, start by exploring with `{"kind":"label_values","label":"__name__","limit":100}`.
- For high-cardinality queries, it is safer to provide `limit` as well.
- For long PromQL expressions or matcher arrays, using `"http_method":"POST"` helps avoid shell escaping and URL length issues.
- When connecting for the first time, it is safest to start with `--dry-run` to verify the endpoint and auth mode.
