# my-github

`my-github`는 GitHub REST API에서 issue, issue list, pull request, pull request list, commit, 특정 ref의 commit history를 조회하는 단일 목적 CLI입니다.  
입력은 JSON 객체 하나만 받으며, 결과도 JSON으로 출력합니다.

이 커맨드는 `my-cli` 프로젝트의 일부이며, 빌드/테스트/린트는 모두 Docker 컨테이너에서 실행합니다.

LLM/agent 환경에서 `my-github` binary를 활용하는 규칙은 [docs/my-github/SKILL.md](../../../docs/my-github/SKILL.md)를 참고합니다.

## Requirements

- Docker
- GNU Make

로컬에 Go나 `golangci-lint`를 직접 설치하지 않아도 됩니다.

## Build

저장소 루트에서 아래처럼 빌드합니다.

```bash
make build my-github
make build CMD=my-github
```

자주 사용하는 옵션 예시는 아래와 같습니다.

```bash
make build my-github VERSION=1.0.0
make build my-github GO_VERSION=1.26.1
make build my-github CGO_ENABLED=1
make build my-github GOOS=linux GOARCH=amd64
make build my-github GOPRIVATE='github.com/your-org/*'
```

출력 파일은 `bin/my-github`에 생성됩니다.  
빌드 시 `ldflags`로 `main.Version`에 `VERSION-git_commit` 값이 주입됩니다.

```bash
./bin/my-github --help
./bin/my-github --dry-run '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
./bin/my-github --version
./bin/my-github -version
```

## Install into Codex

Codex CLI에서 바로 쓰려면 저장소 루트에서 아래 스크립트를 실행합니다.

```bash
./scripts/install-my-github-codex.sh
```

이 스크립트는 `make build CMD=my-github` 실행 후 `~/.codex/bin/my-github`와 `~/.codex/skills/my-github/*`를 함께 갱신합니다.  
다른 Codex 홈을 쓰면 `CODEX_HOME=/path/to/codex ./scripts/install-my-github-codex.sh`처럼 실행하면 됩니다.

## Test

테스트는 `golang:<GO_VERSION>` Docker 이미지에서 실행합니다.

```bash
make test my-github
make test CMD=my-github
```

추가 옵션도 전달할 수 있습니다.

```bash
make test my-github TEST_FLAGS="-v"
make test my-github TEST_FLAGS="-run TestRootCommandFetchesIssue -v"
```

## Lint

린트는 Docker 안에서 `golangci-lint`를 사용합니다. 기본 이미지는 `golangci/golangci-lint:v2.9.0`입니다.

```bash
make lint my-github
make lint CMD=my-github
```

추가 옵션 예시는 아래와 같습니다.

```bash
make lint my-github LINT_FLAGS="--verbose"
make lint my-github LINT_TIMEOUT=10m
make lint my-github GOLANGCI_LINT_VERSION=2.9.0
```

## Helpful Commands

```bash
make list-cmds
make print-version
make clean
```

## 설정 파일

설정 파일 이름은 `my-github.yaml`입니다.  
설정은 [`src/pkg/config/config.go`](../../pkg/config/config.go) 로더를 통해 읽습니다.

검색 경로는 아래 순서입니다.

1. `/etc/my-github/my-github.yaml`
2. `~/my-github.yaml`
3. `./my-github.yaml`

존재하는 파일은 위 순서대로 병합되며, 뒤에 읽은 값이 앞선 값을 덮어씁니다.  
설정 파일이 하나도 없으면 아래 기본값으로 실행합니다.

- `github.base_url`: `https://api.github.com/`
- `github.timeout`: `15s`
- `github.user_agent`: `my-cli/my-github`
- `github.token`: 비어 있음
- `github.by_base_url`: 비어 있음

`github` 최상위 값은 공통 기본값입니다.  
기본 선택은 `github.base_url`로 이뤄지고, 요청 JSON에 `base_url` 또는 `alias`를 넣으면 그 값이 우선합니다.  
선택된 GitHub 인스턴스와 일치하는 `github.by_base_url[]` 항목이 있으면 해당 항목의 `base_url`, `token`, `timeout`, `user_agent`가 마지막에 적용됩니다.  
`github.by_base_url[].alias`는 JSON 요청의 `alias` 필드로 직접 선택할 수 있고, `base_url` 비교는 뒤쪽 `/` 유무를 무시합니다.

예시입니다.

```yaml
github:
  base_url: https://api.github.com/
  timeout: 15s
  user_agent: my-cli/my-github
  by_base_url:
    - alias: github.com
      base_url: https://api.github.com/
      token: "{{ .GITHUB_TOKEN }}"
    - alias: example-ghe
      base_url: https://ghe.example.com/api/v3/
      token: "{{ .GHE_TOKEN }}"
      timeout: 30s
      user_agent: my-cli/my-github-enterprise
```

`token` 같은 secret 값도 config 템플릿으로 관리합니다.  
위 예시는 실행 시점 환경 변수 `GITHUB_TOKEN`, `GHE_TOKEN` 값을 읽어 각 base URL에 맞는 token에 주입합니다.  
여러 GitHub 인스턴스를 함께 쓸 때는 `github.base_url`만 바꾸면 대응하는 override가 자동으로 적용되고, `alias`로 어떤 항목인지 빠르게 구분할 수 있습니다.

## 사용 방법

JSON 입력은 둘 중 하나로 전달합니다.

```bash
./bin/my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123}'
```

```bash
./bin/my-github '{"kind":"issue_list","owner":"cli","repo":"cli","limit":10}'
```

```bash
echo '{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}' | ./bin/my-github
```

```bash
./bin/my-github '{"kind":"commit_history","owner":"cli","repo":"cli","ref":"release/1.0","limit":10}'
```

```bash
./bin/my-github '{"kind":"pull_request_list","owner":"cli","repo":"cli","limit":10}'
```

```bash
./bin/my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123,"alias":"example-ghe"}'
```

```bash
./bin/my-github '{"kind":"pull_request","owner":"cli","repo":"cli","number":456,"base_url":"https://ghe.example.com/api/v3"}'
```

지원 플래그는 아래와 같습니다.

- `--version`, `-version`, `-v`
- `--dry-run`, `-dry-run`, `-n`
- `--help`, `-help`, `-h`

## JSON 입력 공통 규칙

- 입력은 JSON 객체 하나만 허용합니다.
- 인자는 최대 1개만 받을 수 있습니다.
- 알 수 없는 필드는 에러입니다.
- `kind`, `owner`, `repo`는 항상 필요합니다.
- `base_url`과 `alias`는 선택값이며, 특정 `github.by_base_url` 설정을 고를 때 사용합니다.
- 인증이 필요하면 `my-github.yaml`의 `github.token` 또는 선택된 `github.by_base_url[].token`에 값을 넣습니다.
- 환경 변수 기반 secret 주입이 필요하면 `github.token` 또는 `github.by_base_url[].token`에 `{{ .GITHUB_TOKEN }}` 같은 템플릿을 사용합니다.
- `base_url`과 `alias`를 함께 전달하면 같은 `github.by_base_url` 항목을 가리켜야 합니다.

## 공통 필드

| 필드 | 타입 | 필수 | 설명 |
| --- | --- | --- | --- |
| `kind` | string | 예 | 조회 대상 종류 |
| `owner` | string | 예 | GitHub owner 또는 org |
| `repo` | string | 예 | GitHub repository 이름 |
| `number` | integer | 조건부 | `issue`, `pull_request` 조회 시 필요 |
| `ref` | string | 조건부 | `commit`, `commit_history` 조회 시 필요. `commit`에서는 SHA/branch/tag, `commit_history`에서는 보통 branch 이름을 사용 |
| `limit` | integer | 조건부 | `issue_list`, `pull_request_list`, `commit_history` 조회 시 선택값. 1부터 100까지 가능하며, 생략 시 30 |
| `base_url` | string | 아니오 | 이번 요청에서 사용할 GitHub API base URL. 일치하는 `github.by_base_url` 항목이 있으면 override도 함께 적용 |
| `alias` | string | 아니오 | 이번 요청에서 사용할 `github.by_base_url[].alias` 값 |

## kind 값

| 값 | 설명 |
| --- | --- |
| `issue` | Issue 조회 |
| `issue_list` | Repository issue 목록 조회 |
| `pull_request` | Pull Request 조회 |
| `pull_request_list` | Repository pull request 목록 조회 |
| `commit` | Commit 조회 |
| `commit_history` | 특정 ref 기준 commit history 조회 |

아래 별칭도 허용합니다.

- `issue-list`
- `issues`
- `pr`
- `pull-request`
- `pr-list`
- `pr_list`
- `prs`
- `pull-request-list`
- `pulls`
- `commit-history`

별칭을 넣어도 출력의 `kind` 값은 각각 `issue_list`, `pull_request`, `pull_request_list`, `commit_history`로 정규화됩니다.

## 종류별 스펙

### 1. Issue

입력 예시입니다.

```json
{
  "kind": "issue",
  "owner": "cli",
  "repo": "cli",
  "number": 123
}
```

규칙입니다.

- `number`는 1 이상의 정수여야 합니다.
- `ref`는 허용되지 않습니다.
- GitHub API가 pull request 항목을 반환하면 `issue`가 아니라 에러로 처리합니다.

출력 예시입니다.

```json
{
  "kind": "issue",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "issue": {
    "number": 123,
    "title": "Issue title",
    "state": "open",
    "author": "octocat",
    "assignees": ["hubot"],
    "labels": ["bug", "good first issue"],
    "comments": 4,
    "created_at": "2026-03-10T12:00:00Z",
    "updated_at": "2026-03-11T12:00:00Z",
    "closed_at": null,
    "url": "https://github.com/cli/cli/issues/123",
    "api_url": "https://api.github.com/repos/cli/cli/issues/123",
    "body": "Issue body"
  }
}
```

`closed_at`은 닫히지 않은 이슈면 생략되거나 `null`일 수 있습니다.

### 2. Issue List

입력 예시입니다.

```json
{
  "kind": "issue_list",
  "owner": "cli",
  "repo": "cli",
  "limit": 2
}
```

`issue-list`와 `issues`도 같은 의미입니다.

규칙입니다.

- `number`는 허용되지 않습니다.
- `ref`는 허용되지 않습니다.
- `limit`은 1부터 100까지 가능하며, 생략하면 30을 사용합니다.
- GitHub `/issues` 응답에 pull request가 섞여 있으면 내부에서 제외합니다.

출력 예시입니다.

```json
{
  "kind": "issue_list",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "issue_list": {
    "limit": 2,
    "issues": [
      {
        "number": 123,
        "title": "Issue title",
        "state": "open",
        "author": "octocat",
        "assignees": ["hubot"],
        "labels": ["bug"],
        "comments": 4,
        "created_at": "2026-03-10T12:00:00Z",
        "updated_at": "2026-03-11T12:00:00Z",
        "closed_at": null,
        "url": "https://github.com/cli/cli/issues/123",
        "api_url": "https://api.github.com/repos/cli/cli/issues/123",
        "body": "Issue body"
      },
      {
        "number": 122,
        "title": "Second issue",
        "state": "open",
        "author": "hubot",
        "assignees": [],
        "labels": ["docs"],
        "comments": 2,
        "created_at": "2026-03-09T12:00:00Z",
        "updated_at": "2026-03-10T12:00:00Z",
        "closed_at": null,
        "url": "https://github.com/cli/cli/issues/122",
        "api_url": "https://api.github.com/repos/cli/cli/issues/122",
        "body": "Another issue body"
      }
    ]
  }
}
```

### 3. Pull Request

입력 예시입니다.

```json
{
  "kind": "pull_request",
  "owner": "cli",
  "repo": "cli",
  "number": 456
}
```

`pr`와 `pull-request`도 같은 의미입니다.

규칙입니다.

- `number`는 1 이상의 정수여야 합니다.
- `ref`는 허용되지 않습니다.

출력 예시입니다.

```json
{
  "kind": "pull_request",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "pull_request": {
    "number": 456,
    "title": "PR title",
    "state": "open",
    "draft": false,
    "merged": false,
    "author": "monalisa",
    "base_branch": "main",
    "base_sha": "base-sha",
    "head_branch": "feature",
    "head_sha": "head-sha",
    "created_at": "2026-03-10T12:00:00Z",
    "updated_at": "2026-03-11T12:00:00Z",
    "merged_at": null,
    "url": "https://github.com/cli/cli/pull/456",
    "api_url": "https://api.github.com/repos/cli/cli/pulls/456",
    "body": "PR body"
  }
}
```

`merged_at`은 merge되지 않은 PR이면 생략되거나 `null`일 수 있습니다.

### 4. Pull Request List

입력 예시입니다.

```json
{
  "kind": "pull_request_list",
  "owner": "cli",
  "repo": "cli",
  "limit": 2
}
```

`pr-list`, `pr_list`, `prs`, `pull-request-list`, `pulls`도 같은 의미입니다.

규칙입니다.

- `number`는 허용되지 않습니다.
- `ref`는 허용되지 않습니다.
- `limit`은 1부터 100까지 가능하며, 생략하면 30을 사용합니다.

출력 예시입니다.

```json
{
  "kind": "pull_request_list",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "pull_request_list": {
    "limit": 2,
    "pull_requests": [
      {
        "number": 456,
        "title": "PR title",
        "state": "open",
        "draft": false,
        "merged": false,
        "author": "monalisa",
        "base_branch": "main",
        "base_sha": "base-sha",
        "head_branch": "feature",
        "head_sha": "head-sha",
        "created_at": "2026-03-10T12:00:00Z",
        "updated_at": "2026-03-11T12:00:00Z",
        "merged_at": null,
        "url": "https://github.com/cli/cli/pull/456",
        "api_url": "https://api.github.com/repos/cli/cli/pulls/456",
        "body": "PR body"
      }
    ]
  }
}
```

### 5. Commit

입력 예시입니다.

```json
{
  "kind": "commit",
  "owner": "cli",
  "repo": "cli",
  "ref": "trunk"
}
```

규칙입니다.

- `ref`는 비어 있으면 안 됩니다.
- `ref`에는 SHA, branch, tag를 사용할 수 있습니다.
- `number`는 허용되지 않습니다.

출력 예시입니다.

```json
{
  "kind": "commit",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "commit": {
    "sha": "abc123",
    "message": "Commit message",
    "author": {
      "login": "octocat",
      "name": "Octo Cat",
      "email": "octo@example.com",
      "date": "2026-03-10T12:00:00Z"
    },
    "committer": {
      "name": "Octo Bot",
      "email": "bot@example.com",
      "date": "2026-03-10T12:01:00Z"
    },
    "parents": ["parent1", "parent2"],
    "stats": {
      "additions": 12,
      "deletions": 3,
      "total": 15
    },
    "files": [
      {
        "filename": "README.md",
        "status": "modified",
        "additions": 10,
        "deletions": 2,
        "changes": 12,
        "patch": "@@ -1 +1 @@\n-old\n+new"
      },
      {
        "filename": "docs/old.md",
        "status": "renamed",
        "additions": 2,
        "deletions": 1,
        "changes": 3,
        "previous_filename": "docs/legacy.md"
      }
    ],
    "url": "https://github.com/cli/cli/commit/abc123",
    "api_url": "https://api.github.com/repos/cli/cli/commits/abc123"
  }
}
```

GitHub commit author와 Git author가 다를 수 있으므로 `author.login`은 없을 수도 있습니다.
`files[].patch`는 binary 파일이거나 diff가 너무 크면 비어 있거나 생략될 수 있습니다.

### 6. Commit History

입력 예시입니다.

```json
{
  "kind": "commit_history",
  "owner": "cli",
  "repo": "cli",
  "ref": "release/1.0",
  "limit": 2
}
```

규칙입니다.

- `ref`는 비어 있으면 안 됩니다.
- `ref`는 GitHub API의 `sha` query에 전달되며, 일반적으로 branch 이름을 사용합니다.
- `limit`은 1부터 100까지 가능하며, 생략하면 30을 사용합니다.
- `number`는 허용되지 않습니다.

출력 예시입니다.

```json
{
  "kind": "commit_history",
  "repository": {
    "owner": "cli",
    "repo": "cli"
  },
  "commit_history": {
    "ref": "release/1.0",
    "limit": 2,
    "commits": [
      {
        "sha": "abc123",
        "message": "First commit",
        "author": {
          "login": "octocat",
          "name": "Octo Cat",
          "email": "octo@example.com",
          "date": "2026-03-10T12:00:00Z"
        },
        "committer": {
          "login": "github-actions[bot]",
          "name": "GitHub Actions",
          "email": "bot@example.com",
          "date": "2026-03-10T12:01:00Z"
        },
        "parents": ["parent1"],
        "url": "https://github.com/cli/cli/commit/abc123",
        "api_url": "https://api.github.com/repos/cli/cli/commits/abc123"
      },
      {
        "sha": "def456",
        "message": "Second commit",
        "author": {
          "name": "Mona Lisa",
          "email": "mona@example.com",
          "date": "2026-03-09T10:00:00Z"
        },
        "committer": {
          "name": "Mona Lisa",
          "email": "mona@example.com",
          "date": "2026-03-09T10:05:00Z"
        },
        "parents": ["parent2", "parent3"],
        "url": "https://github.com/cli/cli/commit/def456",
        "api_url": "https://api.github.com/repos/cli/cli/commits/def456"
      }
    ]
  }
}
```

결과는 GitHub API 응답 순서를 따르며, 일반적으로 최신 commit부터 내려옵니다.

## dry-run 출력

`--dry-run`은 GitHub API를 호출하지 않고, 어떤 요청을 보낼지만 JSON으로 출력합니다.  
설정 파일의 `github.base_url`, `github.token`, `github.by_base_url`와 요청 JSON의 `base_url`/`alias`를 합쳐 계산한 최종 값이 dry-run 결과에도 반영됩니다.

```json
{
  "mode": "dry-run",
  "http": {
    "method": "GET",
    "url": "https://api.github.com/repos/cli/cli/issues/123",
    "auth": "token"
  },
  "request": {
    "kind": "issue",
    "owner": "cli",
    "repo": "cli",
    "number": 123
  }
}
```

`auth` 값은 아래 둘 중 하나입니다.

- `token`
- `none`

## 에러 규칙

아래 경우는 에러입니다.

- JSON 문법 오류
- JSON 객체가 아닌 입력
- JSON 객체가 둘 이상 포함된 입력
- 인자를 2개 이상 전달한 경우
- 알 수 없는 필드 포함
- `kind` 값 불일치
- `owner` 또는 `repo` 누락
- `issue` 또는 `pull_request`에서 `number` 누락 또는 0 이하
- `issue` 또는 `pull_request`에서 `ref` 전달
- `issue_list` 또는 `pull_request_list`에서 `number` 전달
- `issue_list` 또는 `pull_request_list`에서 `ref` 전달
- `issue_list` 또는 `pull_request_list`에서 `limit`이 1 미만이거나 100 초과인 경우
- `commit`에서 `ref` 누락
- `commit`에서 `number` 전달
- `commit_history`에서 `ref` 누락
- `commit_history`에서 `number` 전달
- `commit_history`에서 `limit`이 1 미만이거나 100 초과인 경우
- `alias`가 어떤 `github.by_base_url[].alias`와도 일치하지 않는 경우
- `base_url`과 `alias`를 함께 넣었지만 서로 다른 `github.by_base_url` 항목을 가리키는 경우
- GitHub API가 4xx/5xx를 반환한 경우
