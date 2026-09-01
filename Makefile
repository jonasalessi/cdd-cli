.PHONY: setup build test cover lint fmt check-literals

GOLANGCI_LINT_VERSION = v2.13.2
GOLANGCI_LINT = bin/golangci-lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -X github.com/jonasalessi/cdd-cli/cmd.version=$(VERSION) \
           -X github.com/jonasalessi/cdd-cli/cmd.commit=$(COMMIT) \
           -X github.com/jonasalessi/cdd-cli/cmd.date=$(DATE)

## setup: configure the local clone (installs the git hooks in .githooks)
setup:
	git config core.hooksPath .githooks
	chmod +x .githooks/*
	@echo "Git hooks installed from .githooks/"

## build: compile the cdd binary into bin/ with version info injected
build:
	go build -ldflags "$(LDFLAGS)" -o bin/cdd .

## test: run every test with the race detector on
test:
	go test ./... -race

## cover: run the tests and print per-function coverage
cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -func=coverage.out

## lint: run golangci-lint (auto-installed into bin/) and the vocabulary literal check
lint: check-literals $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b bin $(GOLANGCI_LINT_VERSION)

## fmt: rewrite every Go file with gofmt
fmt:
	gofmt -l -w .

## check-literals: metric, language and mode ids may only be spelled out in vocabulary.go
check-literals:
	@hits=$$(grep -rnE '"(go|java|kotlin|typescript|code_branch|condition|exception_handling|internal_coupling|external_coupling|inheritance|local_variable|lambda|greenfield|legacy|strict_all|strict_on_new_only|boy_scout|measure_only)"' \
		--include='*.go' --exclude='*_test.go' --exclude='vocabulary.go' . | grep -v 'yaml:"' || true); \
	if [ -n "$$hits" ]; then \
		echo "vocabulary literals found outside internal/config/vocabulary.go:"; \
		echo "$$hits"; \
		exit 1; \
	fi
