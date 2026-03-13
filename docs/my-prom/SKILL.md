---
name: my-prom
description: Use the `my-prom` CLI to query Prometheus instant queries, range queries, series, label names, and label values with one JSON request and a normalized JSON response.
---

# My Prom

This skill uses the `my-prom` binary to call Prometheus HTTP API endpoints from a single JSON request. Prefer it over ad hoc `curl` commands when you want predictable JSON input, `--dry-run` support, config-based instance selection, and normalized output for AI agents.

In Codex CLI installs, the binary lives under `${CODEX_HOME}/bin/my-prom` and the config file lives under `${HOME}/.config/my-prom.yaml`. If it is not on `PATH`, use the provided absolute binary path.

## When to use

Use this skill when:

- Running a Prometheus instant query
- Running a Prometheus range query
- Looking up matching series by `match[]`
- Discovering label names
- Discovering label values such as metric names via `__name__`
- Working in an agent or CLI workflow where one JSON request is easier than composing HTTP calls manually

## Quick workflow

1. Pass exactly one JSON object as a CLI argument or through `stdin`.
2. Use `--dry-run` first when the request shape is uncertain.
3. Prefer canonical kinds such as `query`, `query_range`, `series`, `label_names`, and `label_values`.
4. Use `label_values` with `label="__name__"` when you need metric discovery.
5. Use `http_method="POST"` for long PromQL or many matchers.

## Input

Provide exactly one JSON object.

Common required fields:

- `kind`

Common optional fields:

- `base_url` to choose a specific Prometheus API base URL for this request
- `alias` to choose a configured `prometheus.instances[].alias` entry
- `http_method` to force `GET` or `POST`
- `limit` to cap returned series or items. `0` means no limit

Query fields:

- `query` for `query` and `query_range`
- `time` for `query`
- `start`, `end`, `step` for `query_range`
- `timeout` for `query` and `query_range`

Metadata fields:

- `matchers` for `series`, `label_names`, and `label_values`
- `label` for `label_values`
- `start` and `end` are optional filters for `series`, `label_names`, and `label_values`

Supported `kind` values:

- `query`
- `query_range`
- `series`
- `label_names`
- `label_values`

Accepted aliases:

- `instant`, `instant-query`, `instant_query` -> `query`
- `range`, `range-query`, `range_query` -> `query_range`
- `label-names`, `labels` -> `label_names`
- `label-values` -> `label_values`

Validation rules:

- Unknown fields are errors.
- `query` is required for `query` and `query_range`.
- `start`, `end`, and `step` are required for `query_range`.
- `matchers` must contain at least one matcher for `series`.
- `label` is required for `label_values`.
- `label_values` only supports `GET`.
- `http_method` must be `GET` or `POST` when provided.
- If both `base_url` and `alias` are provided, they must point to the same configured instance.

Example inputs:

```json
{"kind":"query","query":"up"}
```

```json
{"kind":"query_range","query":"rate(http_requests_total[5m])","start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z","step":"5m"}
```

```json
{"kind":"series","matchers":["up{job=\"node\"}"],"start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z"}
```

```json
{"kind":"label_names","matchers":["{job=\"api\"}"],"alias":"prod-prom"}
```

```json
{"kind":"label_values","label":"__name__","limit":100}
```

```json
{"kind":"query","query":"sum(rate(container_cpu_usage_seconds_total[5m])) by (pod)","http_method":"POST"}
```

## Output

Successful responses always include:

- `kind`
- request echo fields such as `query`, `matchers`, or `label`
- `count`
- `warnings`
- `infos`

Query responses also include:

- `result_type`
- `result`

Sample timestamps are normalized to both RFC3339 and Unix seconds so agents can reason about time without extra parsing.

Example output shape:

```json
{
  "kind": "query",
  "query": "up",
  "result_type": "vector",
  "count": 1,
  "result": [
    {
      "metric": {
        "__name__": "up",
        "job": "node"
      },
      "value": {
        "timestamp": "2024-03-12T22:40:00.5Z",
        "timestamp_unix": 1710283200.5,
        "value": "1"
      }
    }
  ]
}
```

## Command examples

If the binary is not on `PATH`, replace `my-prom` with the provided absolute path.

```bash
my-prom '{"kind":"query","query":"up"}'
```

```bash
my-prom '{"kind":"query_range","query":"rate(http_requests_total[5m])","start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z","step":"5m"}'
```

```bash
my-prom '{"kind":"series","matchers":["up{job=\"node\"}"]}'
```

```bash
my-prom '{"kind":"label_names","matchers":["{job=\"api\"}"],"alias":"prod-prom"}'
```

```bash
my-prom '{"kind":"label_values","label":"__name__","limit":100}'
```

```bash
my-prom --dry-run '{"kind":"query","query":"up","time":"2026-03-13T12:00:00Z"}'
```

```bash
echo '{"kind":"query","query":"sum(rate(container_cpu_usage_seconds_total[5m])) by (pod)","http_method":"POST"}' | my-prom
```

## Flags

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## Failure prevention

- Use `stdin` if shell escaping is awkward.
- Start with `--dry-run` before real calls when the request is uncertain.
- Use `label_values` with `label="__name__"` before writing PromQL against an unfamiliar Prometheus.
- Use `POST` for long PromQL expressions or matcher lists.
- If you need build, test, or lint instructions, read [README.md](../../src/cmd/my-prom/README.md).
