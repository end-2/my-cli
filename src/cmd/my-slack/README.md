# my-slack

`my-slack`는 Slack Web API에 create, read, update, delete, list 성격의 요청을 JSON 입력/출력으로 보내는 단일 목적 CLI입니다.  
입력은 JSON 객체 하나만 받으며, 결과도 JSON으로 출력합니다.

이 커맨드는 `my-cli` 프로젝트의 일부이며, 빌드/테스트/린트는 모두 Docker 컨테이너에서 실행합니다.

LLM/agent 환경에서 `my-slack` binary를 활용하는 규칙은 [docs/my-slack/SKILL.md](../../../docs/my-slack/SKILL.md)를 참고합니다.

## Requirements

- Docker
- GNU Make

로컬에 Go나 `golangci-lint`를 직접 설치하지 않아도 됩니다.

## Build

저장소 루트에서 아래처럼 빌드합니다.

```bash
make build my-slack
make build CMD=my-slack
```

자주 사용하는 옵션 예시는 아래와 같습니다.

```bash
make build my-slack VERSION=1.0.0
make build my-slack GO_VERSION=1.26.1
make build my-slack CGO_ENABLED=1
make build my-slack GOOS=linux GOARCH=amd64
make build my-slack GOPRIVATE='github.com/your-org/*'
```

출력 파일은 `bin/my-slack`에 생성됩니다.  
빌드 시 `ldflags`로 `main.Version`에 `VERSION-git_commit` 값이 주입됩니다.

```bash
./bin/my-slack --help
./bin/my-slack --dry-run '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'
./bin/my-slack --version
./bin/my-slack -version
```

## Install into Codex

Codex CLI에서 바로 쓰려면 저장소 루트에서 아래 스크립트를 실행합니다.

```bash
./scripts/install-my-slack-codex.sh
```

이 스크립트는 `make build CMD=my-slack` 실행 후 `~/.codex/bin/my-slack`와 `~/.codex/skills/my-slack/*`를 함께 갱신합니다.  
다른 Codex 홈을 쓰면 `CODEX_HOME=/path/to/codex ./scripts/install-my-slack-codex.sh`처럼 실행하면 됩니다.

## Test

테스트는 `golang:<GO_VERSION>` Docker 이미지에서 실행합니다.

```bash
make test my-slack
make test CMD=my-slack
```

추가 옵션도 전달할 수 있습니다.

```bash
make test my-slack TEST_FLAGS="-v"
make test my-slack TEST_FLAGS="-run TestRootCommandFetchesListWithPagination -v"
```

## Lint

린트는 Docker 안에서 `golangci-lint`를 사용합니다. 기본 이미지는 `golangci/golangci-lint:v2.9.0`입니다.

```bash
make lint my-slack
make lint CMD=my-slack
```

추가 옵션 예시는 아래와 같습니다.

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

## 설정 파일

설정 파일 이름은 `my-slack.yaml`입니다.  
설정은 [`src/pkg/config/config.go`](../../pkg/config/config.go) 로더를 통해 읽습니다.

검색 경로는 아래 순서입니다.

1. `/etc/my-slack/my-slack.yaml`
2. `~/my-slack.yaml`
3. `./my-slack.yaml`

존재하는 파일은 위 순서대로 병합되며, 뒤에 읽은 값이 앞선 값을 덮어씁니다.  
설정 파일이 하나도 없으면 아래 기본값으로 실행합니다.

- `slack.base_url`: `https://slack.com/api/`
- `slack.timeout`: `15s`
- `slack.user_agent`: `my-cli/my-slack`
- `slack.token`: 비어 있음
- `slack.workspaces`: 비어 있음

`slack` 최상위 값은 공통 기본값입니다.  
요청 JSON에 `base_url`을 넣으면 이번 요청에서만 base URL을 바꿀 수 있고, `alias`를 넣으면 `slack.workspaces[]`의 특정 workspace 설정을 선택합니다.

예시입니다.

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

`token` 같은 secret 값도 config 템플릿으로 관리합니다.  
위 예시는 실행 시점 환경 변수 `SLACK_DEV_BOT_TOKEN`, `SLACK_PROD_BOT_TOKEN` 값을 읽어 workspace별 token에 주입합니다.

## 사용 방법

JSON 입력은 둘 중 하나로 전달합니다.

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

지원 플래그는 아래와 같습니다.

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## JSON 입력 공통 규칙

- 입력은 JSON 객체 하나만 허용합니다.
- 인자는 최대 1개만 받을 수 있습니다.
- 알 수 없는 필드는 에러입니다.
- `kind`, `method`는 항상 필요합니다.
- `args`는 Slack method argument 객체입니다.
- `base_url`과 `alias`는 선택값입니다.
- 인증이 필요하면 `my-slack.yaml`의 `slack.token` 또는 선택된 `slack.workspaces[].token`에 값을 넣습니다.
- 환경 변수 기반 secret 주입이 필요하면 `slack.token` 또는 `slack.workspaces[].token`에 `{{ .SLACK_BOT_TOKEN }}` 같은 템플릿을 사용합니다.
- `http_method`는 선택값이며 `GET` 또는 `POST`만 허용합니다.

## 공통 필드

| 필드 | 타입 | 필수 | 설명 |
| --- | --- | --- | --- |
| `kind` | string | 예 | 요청 종류. `create`, `read`, `update`, `delete`, `list` |
| `method` | string | 예 | Slack Web API method 이름. 예: `conversations.list` |
| `args` | object | 아니오 | Slack method argument 객체 |
| `limit` | integer | 조건부 | `list`에서 수집할 최대 item 수. 1부터 1000까지 가능하며, 생략 시 100 |
| `cursor` | string | 아니오 | `list`에서 시작 cursor |
| `list_field` | string | 아니오 | list 응답 배열 필드명을 명시하고 싶을 때 사용. 예: `channels`, `messages`, `members` |
| `http_method` | string | 아니오 | `GET` 또는 `POST`. 생략 시 `create/update/delete`는 `POST`, `read/list`는 `GET` |
| `base_url` | string | 아니오 | 이번 요청에서 사용할 Slack API base URL |
| `alias` | string | 아니오 | 이번 요청에서 사용할 `slack.workspaces[].alias` 값 |

## kind 값

| 값 | 설명 |
| --- | --- |
| `create` | Slack 쓰기 생성 요청 |
| `read` | Slack 조회 요청 |
| `update` | Slack 수정 요청 |
| `delete` | Slack 삭제 요청 |
| `list` | Slack 목록 조회 요청. cursor 기반 pagination 자동 지원 |

아래 별칭도 허용합니다.

- `post` -> `create`
- `get` -> `read`
- `put`, `patch` -> `update`
- `remove` -> `delete`
- `ls` -> `list`

## 대표 메서드 예시

- `create`: `conversations.create`, `chat.postMessage`
- `read`: `conversations.info`, `auth.test`
- `update`: `conversations.rename`, `chat.update`
- `delete`: `conversations.archive`, `chat.delete`
- `list`: `conversations.list`, `users.list`, `conversations.history`, `conversations.members`, `conversations.replies`

## list 동작 방식

`kind=list`이면 `response_metadata.next_cursor`를 따라가며 요청한 `limit`만큼 아이템을 모읍니다.  
응답에는 아래 두 형태가 함께 들어갑니다.

- `list`: field 이름, 요청 limit, count, next cursor, 합쳐진 items
- `response`: Slack 원본 응답 형태를 최대한 유지하되, list field 배열은 합쳐진 결과로 교체

응답 배열 필드를 자동으로 찾지 못하면 `list_field`를 명시하면 됩니다.

## 응답 형태

성공 시 공통으로 아래 필드를 포함합니다.

- `kind`
- `method`
- `response`

`kind=list`일 때는 `list` 객체가 추가됩니다.

예시입니다.

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

## 참고 문서

- Slack Web API overview: <https://api.slack.com/web>
- Method reference: <https://docs.slack.dev/reference/methods/>
