GO_VERSION ?= 1.26.1
DOCKER_IMAGE := golang:$(GO_VERSION)
GOLANGCI_LINT_VERSION ?= 2.9.0
GOLANGCI_LINT_IMAGE := golangci/golangci-lint:v$(GOLANGCI_LINT_VERSION)

CMD_DIRS := $(sort $(notdir $(patsubst %/,%,$(wildcard src/cmd/*/))))
CMD_FROM_GOAL := $(shell set -- $(MAKECMDGOALS); prev=''; for arg in "$$@"; do if [ "$$prev" = "build" ] || [ "$$prev" = "test" ] || [ "$$prev" = "lint" ]; then printf '%s\n' "$$arg"; break; fi; prev="$$arg"; done)
CMD ?= $(CMD_FROM_GOAL)

ifeq ($(words $(CMD_DIRS)),1)
DEFAULT_BUILD_CMD := $(firstword $(CMD_DIRS))
endif

BUILD_CMD := $(if $(strip $(CMD)),$(CMD),$(DEFAULT_BUILD_CMD))
BUILD_PACKAGE = ./src/cmd/$(BUILD_CMD)
BINARY_NAME ?= $(BUILD_CMD)
BIN_DIR ?= bin
CACHE_DIR ?= .cache
VERSION ?= dev
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BINARY_VERSION := $(VERSION)-$(GIT_COMMIT)
LDFLAGS := -s -w -X main.Version=$(BINARY_VERSION)
TEST_FLAGS ?=
TEST_TARGET ?= $(if $(strip $(CMD)),./src/cmd/$(CMD)/...,./...)
LINT_FLAGS ?=
LINT_TIMEOUT ?= 5m
LINT_TARGET ?= $(if $(strip $(CMD)),./src/cmd/$(CMD)/...,./...)

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
HOST_GOOS := darwin
else ifeq ($(UNAME_S),Linux)
HOST_GOOS := linux
else
HOST_GOOS := windows
endif

ifeq ($(UNAME_M),x86_64)
HOST_GOARCH := amd64
else ifeq ($(UNAME_M),arm64)
HOST_GOARCH := arm64
else ifeq ($(UNAME_M),aarch64)
HOST_GOARCH := arm64
else
HOST_GOARCH := $(UNAME_M)
endif

GOOS ?= $(HOST_GOOS)
GOARCH ?= $(HOST_GOARCH)
CGO_ENABLED ?= 0
GOPRIVATE ?=
GONOPROXY ?= $(GOPRIVATE)
GONOSUMDB ?= $(GOPRIVATE)
GOPROXY ?= https://proxy.golang.org,direct
OUTPUT_EXT := $(if $(filter windows,$(GOOS)),.exe,)
OUTPUT := $(BIN_DIR)/$(BINARY_NAME)$(OUTPUT_EXT)
DOCKER_GOCACHE := /src/$(CACHE_DIR)/go-build
DOCKER_GOMODCACHE := /src/$(CACHE_DIR)/gomod
DOCKER_GOLANGCI_LINT_CACHE := /src/$(CACHE_DIR)/golangci-lint

DOCKER_COMMON_ARGS = --rm \
	-u "$$(id -u):$$(id -g)" \
	-v "$(CURDIR):/src" \
	-w /src \
	-e HOME=/tmp \
	-e GOPRIVATE=$(GOPRIVATE) \
	-e GONOPROXY=$(GONOPROXY) \
	-e GONOSUMDB=$(GONOSUMDB) \
	-e GOPROXY=$(GOPROXY) \
	-e GOCACHE=$(DOCKER_GOCACHE) \
	-e GOMODCACHE=$(DOCKER_GOMODCACHE)

.PHONY: build test lint clean print-version list-cmds validate-build-cmd validate-scope-cmd prepare-cache $(CMD_DIRS)

build: validate-build-cmd prepare-cache
	docker run $(DOCKER_COMMON_ARGS) \
		-e CGO_ENABLED=$(CGO_ENABLED) \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		$(DOCKER_IMAGE) \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUTPUT) $(BUILD_PACKAGE)

test: validate-scope-cmd prepare-cache
	docker run $(DOCKER_COMMON_ARGS) \
		-e CGO_ENABLED=$(CGO_ENABLED) \
		$(DOCKER_IMAGE) \
		go test $(TEST_FLAGS) $(TEST_TARGET)

lint: validate-scope-cmd prepare-cache
	docker run $(DOCKER_COMMON_ARGS) \
		-e CGO_ENABLED=$(CGO_ENABLED) \
		-e GOLANGCI_LINT_CACHE=$(DOCKER_GOLANGCI_LINT_CACHE) \
		$(GOLANGCI_LINT_IMAGE) \
		golangci-lint run --timeout $(LINT_TIMEOUT) $(LINT_FLAGS) $(LINT_TARGET)

clean:
	rm -rf $(BIN_DIR) $(CACHE_DIR)

print-version:
	@printf '%s\n' "$(BINARY_VERSION)"

list-cmds:
	@printf '%s\n' $(CMD_DIRS)

prepare-cache:
	@mkdir -p $(BIN_DIR) $(CACHE_DIR)/go-build $(CACHE_DIR)/gomod $(CACHE_DIR)/golangci-lint

validate-build-cmd:
	@if [ -z "$(BUILD_CMD)" ]; then \
		echo "CMD is required. Use 'make build CMD=<name>' or 'make build <name>'."; \
		echo "Available commands: $(CMD_DIRS)"; \
		exit 1; \
	fi
	@if [ ! -d "src/cmd/$(BUILD_CMD)" ]; then \
		echo "Unknown CMD: $(BUILD_CMD)"; \
		echo "Available commands: $(CMD_DIRS)"; \
		exit 1; \
	fi

validate-scope-cmd:
	@if [ -n "$(CMD)" ] && [ ! -d "src/cmd/$(CMD)" ]; then \
		echo "Unknown CMD: $(CMD)"; \
		echo "Available commands: $(CMD_DIRS)"; \
		exit 1; \
	fi

$(CMD_DIRS):
	@:
