# MY-CLI

`my-cli` is a Go project for building custom CLI tools that I use personally.

The CLIs in this project are designed to be easy to use in AI / LLM-powered CLI environments.

Key characteristics:

- All input and output use a JSON-based interface
- Single-purpose executables that AI agents can call directly
- Can be used as skills in AI CLIs such as Codex CLI
- A stable CLI contract that works for both humans and AI

## CLI List

### my-github

`my-github` is a CLI that fetches issue, pull request, and commit information from the GitHub REST API through JSON input/output.
See [src/cmd/my-github/README.md](src/cmd/my-github/README.md) for detailed usage.

### my-slack

`my-slack` is a CLI that sends create, read, update, delete, and list style requests to the Slack Web API through JSON input/output.
See [src/cmd/my-slack/README.md](src/cmd/my-slack/README.md) for detailed usage.

### my-prom

`my-prom` is a CLI that sends instant query, range query, series lookup, label name lookup, and label value lookup requests to the Prometheus HTTP API through JSON input/output.
See [src/cmd/my-prom/README.md](src/cmd/my-prom/README.md) for detailed usage.

### my-discord

`my-discord` is a CLI that sends create, read, update, delete, and list style requests to the Discord REST API through JSON input/output.
See [src/cmd/my-discord/README.md](src/cmd/my-discord/README.md) for detailed usage.

## Requirements

- Docker
- GNU Make

You do not need to install `Go` or `golangci-lint` locally.

## Build

Builds use the `golang:<GO_VERSION>` Docker image.
The output binary is created at `bin/<command>`.

```bash
# sample
make build sample
# my-github
make build my-github
```

```bash
./bin/sample --help
./bin/sample --dry-run
./bin/sample --version
```

### Use my-github in Codex CLI

To use `my-github` directly from Codex CLI, run the installation script.

```bash
./scripts/install-my-github-codex.sh
```

The config file for `my-github` is `${HOME}/.config/my-github.yaml`.
Edit it as needed with the following command.

```bash
vi ${HOME}/.config/my-github.yaml
```

It is safest to verify connectivity by starting with `--dry-run`, like this.

```bash
${CODEX_HOME}/bin/my-github --dry-run '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
```

Mentioning `my-github` directly in a Codex CLI prompt helps the skill get selected more reliably. Examples:

```text
Use my-github to fetch issue #123 from the cli/cli repository and summarize only the title, state, and author.

Use the my-github skill to fetch pull request #456 from the openai/openai-python repository and summarize the key changes in 3 lines.

Use my-github to fetch the commit for the trunk ref in the cli/cli repository and tell me the SHA, author, and message.
```

### Use my-slack in Codex CLI

To use `my-slack` directly from Codex CLI, run the installation script.

```bash
./scripts/install-my-slack-codex.sh
```

The config file for `my-slack` is `${HOME}/.config/my-slack.yaml`.
Edit it as needed with the following command.

```bash
vi ${HOME}/.config/my-slack.yaml
```

It is safest to verify connectivity by starting with `--dry-run`, like this.

```bash
${CODEX_HOME}/bin/my-slack --dry-run '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'
```

Mentioning `my-slack` directly in a Codex CLI prompt helps the skill get selected more reliably. Examples:

```text
Use my-slack to fetch channel information for C12345678 with conversations.info.

Use the my-slack skill to call users.list and return only 20 users.

Use my-slack and show me the dry-run result for chat.postMessage first.
```

### Use my-prom in Codex CLI

To use `my-prom` directly from Codex CLI, run the installation script.

```bash
./scripts/install-my-prom-codex.sh
```

The config file for `my-prom` is `${HOME}/.config/my-prom.yaml`.
Edit it as needed with the following command.

```bash
vi ${HOME}/.config/my-prom.yaml
```

It is safest to verify connectivity by starting with `--dry-run`, like this.

```bash
${CODEX_HOME}/bin/my-prom --dry-run '{"kind":"query","query":"up"}'
```

Mentioning `my-prom` directly in a Codex CLI prompt helps the skill get selected more reliably. Examples:

```text
Use my-prom to fetch the current up metric result and summarize the status by instance.

Use the my-prom skill to fetch rate(http_requests_total[5m]) for the last hour, then show only the number of result series and the latest 3 values from the first series.

Use my-prom to fetch 50 __name__ label values and tell me which metric names are available.
```

### Use my-discord in Codex CLI

To use `my-discord` directly from Codex CLI, run the installation script.

```bash
./scripts/install-my-discord-codex.sh
```

The config file for `my-discord` is `${HOME}/.config/my-discord.yaml`.
Edit it as needed with the following command.

```bash
vi ${HOME}/.config/my-discord.yaml
```

It is safest to verify connectivity by starting with `--dry-run`, like this.

```bash
${CODEX_HOME}/bin/my-discord --dry-run '{"kind":"read","path":"/channels/123"}'
```

Mentioning `my-discord` directly in a Codex CLI prompt helps the skill get selected more reliably. Examples:

```text
Use my-discord to fetch the /channels/123 path.

Use the my-discord skill to fetch 50 items from /guilds/123/members and paginate by user.id.

Use my-discord and show me the dry-run result for a create request to /channels/123/messages first.
```

## Test

Tests run `go test` inside the `golang:<GO_VERSION>` Docker image.

This runs the full test suite.

```bash
make test
```

You can also run tests for a specific command.

```bash
make test sample
make test CMD=sample
```

You can pass additional options as well.

```bash
make test TEST_FLAGS="-v"
make test sample TEST_FLAGS="-run TestName -v"
```

## Lint

Linting uses `golangci-lint` inside Docker.
The default image is `golangci/golangci-lint:v2.9.0`.

This lints the entire repository.

```bash
make lint
```

You can also lint a specific command.

```bash
make lint sample
make lint CMD=sample
```

Additional option examples:

```bash
make lint LINT_FLAGS="--verbose"
make lint sample LINT_TIMEOUT=10m
make lint GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

Show the list of buildable commands.

```bash
make list-cmds
```

Check the current version string.

```bash
make print-version
```

Remove build outputs and caches.

```bash
make clean
```
