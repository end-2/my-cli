# MY-CLI

`my-cli`는 개인적으로 사용하는 커스텀 CLI 도구를 만들기 위한 Go 프로젝트입니다.
이 프로젝트의 CLI는 **AI / LLM 기반 CLI 환경에서 사용하기 쉽게 설계**되었습니다.

특징은 다음과 같습니다.

- 모든 입출력은 **JSON 기반 인터페이스**
- **AI Agent가 직접 호출하기 쉬운 단일 실행 CLI**
- **Codex CLI 같은 AI CLI에서 skill 형태로 사용 가능**
- 사람과 AI가 모두 사용할 수 있는 **Stable CLI contract**

## CLI List

### my-github

`my-github`는 GitHub REST API에서 issue, pull request, commit 정보를 JSON 입력/출력으로 조회하는 CLI입니다.
자세한 사용법은 [src/cmd/my-github/README.md](src/cmd/my-github/README.md)를 참고하세요.

### my-slack

`my-slack`는 Slack Web API에 create, read, update, delete, list 성격의 요청을 JSON 입력/출력으로 보내는 CLI입니다.
자세한 사용법은 [src/cmd/my-slack/README.md](src/cmd/my-slack/README.md)를 참고하세요.

## Requirements

- Docker
- GNU Make

로컬에 `Go`나 `golangci-lint`를 직접 설치하지 않아도 됩니다.

## Build

빌드는 `golang:<GO_VERSION>` Docker 이미지를 사용합니다.
출력 파일은 `bin/<command>`에 생성됩니다.

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

### Codex CLI에서 my-github 사용

`my-github`를 실제 Codex CLI에서 사용하려면 설치 스크립트를 실행합니다.

```bash
./scripts/install-my-github-codex.sh
```

`my-github`의 설정 파일은 `${HOME}/my-github.yaml`입니다.
다음 명령어를 통해 필요한 내용을 수정해주세요.

```bash
vi ${HOME}/my-github.yaml
```

`my-github` 연결 확인은 아래처럼 `--dry-run`으로 시작하는 편이 안전합니다.

```bash
${CODEX_HOME}/bin/my-github --dry-run '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
```

Codex CLI 프롬프트에서는 `my-github`를 직접 언급하면 skill이 더 안정적으로 선택됩니다. 예시는 아래와 같습니다.

```text
my-github를 사용해서 cli/cli 저장소의 issue #123을 조회하고 제목, 상태, 작성자만 요약해줘.

my-github skill로 openai/openai-python 저장소의 pull request #456을 조회해서 핵심 변경사항을 3줄로 정리해줘.

my-github를 사용해서 cli/cli 저장소의 trunk ref commit 정보를 조회하고 SHA, 작성자, 메시지를 알려줘.
```

### Codex CLI에서 my-slack 사용

`my-slack`를 실제 Codex CLI에서 사용하려면 설치 스크립트를 실행합니다.

```bash
./scripts/install-my-slack-codex.sh
```

`my-slack`의 설정 파일은 `${HOME}/my-slack.yaml`입니다.
다음 명령어를 통해 필요한 내용을 수정해주세요.

```bash
vi ${HOME}/my-slack.yaml
```

`my-slack` 연결 확인은 아래처럼 `--dry-run`으로 시작하는 편이 안전합니다.

```bash
${CODEX_HOME}/bin/my-slack --dry-run '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'
```

Codex CLI 프롬프트에서는 `my-slack`를 직접 언급하면 skill이 더 안정적으로 선택됩니다. 예시는 아래와 같습니다.

```text
my-slack를 사용해서 conversations.info로 C12345678 채널 정보를 조회해줘.

my-slack skill로 users.list를 호출해서 사용자 목록을 20개만 가져와줘.

my-slack를 사용해서 chat.postMessage dry-run 결과를 먼저 보여줘.
```

## Test

테스트는 `golang:<GO_VERSION>` Docker 이미지에서 `go test`를 실행합니다.

전체 테스트입니다.

```bash
make test
```

특정 커맨드만 테스트할 수도 있습니다.

```bash
make test sample
make test CMD=sample
```

추가 옵션도 전달할 수 있습니다.

```bash
make test TEST_FLAGS="-v"
make test sample TEST_FLAGS="-run TestName -v"
```

## Lint

린트는 Docker 안에서 `golangci-lint`를 사용합니다.
기본 이미지는 `golangci/golangci-lint:v2.9.0`입니다.

전체 린트입니다.

```bash
make lint
```

특정 커맨드만 린트할 수도 있습니다.

```bash
make lint sample
make lint CMD=sample
```

추가 옵션 예시는 아래와 같습니다.

```bash
make lint LINT_FLAGS="--verbose"
make lint sample LINT_TIMEOUT=10m
make lint GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

빌드 가능한 커맨드 목록 확인입니다.

```bash
make list-cmds
```

현재 버전 문자열 확인입니다.

```bash
make print-version
```

빌드 산출물과 캐시 정리입니다.

```bash
make clean
```
