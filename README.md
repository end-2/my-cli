# my-cli

`my-cli`는 개인적으로 사용하는 커스텀 CLI 도구를 만들기 위한 Go 프로젝트입니다.
빌드, 테스트, 린트는 모두 Docker 컨테이너에서 실행합니다.

## Requirements

- Docker
- GNU Make

로컬에 Go나 `golangci-lint`를 직접 설치하지 않아도 됩니다.

## Build

빌드는 `golang:<GO_VERSION>` Docker 이미지를 사용합니다.
출력 파일은 `bin/<command>`에 생성됩니다.

```bash
make build sample
make build CMD=sample
```

현재 `src/cmd` 아래 디렉토리가 하나뿐이면 아래처럼 기본 빌드도 가능합니다.

```bash
make build
```

자주 사용하는 옵션은 아래와 같습니다.

```bash
make build sample VERSION=1.0.0
make build sample GO_VERSION=1.26.1
make build sample CGO_ENABLED=1
make build sample GOOS=linux GOARCH=amd64
make build sample GOPRIVATE='github.com/your-org/*'
```

빌드 시 `ldflags`로 `main.Version`에 `VERSION-git_commit` 값이 주입됩니다.
현재 binary 버전은 아래처럼 확인할 수 있습니다.

```bash
./bin/sample --help
./bin/sample --dry-run
./bin/sample --version
./bin/sample -version
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
