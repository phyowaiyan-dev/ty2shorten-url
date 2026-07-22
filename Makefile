VERSION ?= dev
COMMIT ?= unknown
BUILD_TIME ?= unknown
AIR_VERSION ?= v1.67.1
STATICCHECK ?= $(shell if command -v staticcheck >/dev/null 2>&1; then command -v staticcheck; else printf '%s/bin/staticcheck' "$$(go env GOPATH)"; fi)

.PHONY: help run run-once dev dev-go dev-css fmt fmt-check test test-race test-mysql coverage vet lint staticcheck css css-watch build build-linux clean check install-dev-tools

help:
	@printf '%s\n' \
		'Targets:' \
		'  run          Alias for dev live reload' \
		'  run-once     Run the local server once without Air' \
		'  dev          Run Air and Tailwind/daisyUI watchers' \
		'  dev-go       Run Air Go live reload only' \
		'  dev-css      Run Tailwind/daisyUI CSS watcher only' \
		'  fmt          Format Go files' \
		'  fmt-check    Fail when Go files need formatting' \
		'  test         Run tests' \
		'  test-race    Run tests with the race detector' \
		'  test-mysql   Run optional MySQL integration tests' \
		'  coverage     Write coverage.out' \
		'  vet          Run go vet' \
		'  lint         Run Staticcheck' \
		'  css          Build compiled Tailwind/daisyUI CSS' \
		'  css-watch    Watch Tailwind/daisyUI CSS sources' \
		'  build        Build local binary' \
		'  build-linux  Build Linux AMD64 binary' \
		'  check        Run local validation sequence' \
		'  clean        Remove build and coverage outputs' \
		'  install-dev-tools  Install pinned local development tools'

run: dev

run-once:
	go run ./cmd/server

dev:
	./scripts/dev.sh

dev-go:
	@command -v air >/dev/null 2>&1 || { \
		echo "Air is not installed."; \
		echo "Install it with: make install-dev-tools"; \
		exit 1; \
	}
	air

dev-css:
	npm run css:watch

fmt:
	go fmt ./...

fmt-check:
	test -z "$$(gofmt -l .)"

test:
	go test ./...

test-race:
	go test -race ./...

test-mysql:
	TY2_MYSQL_INTEGRATION=1 go test -tags mysqlintegration ./...

coverage:
	go test -coverprofile=coverage.out ./...

vet:
	go vet ./...

lint: staticcheck

staticcheck:
	$(STATICCHECK) ./...

css:
	npm run css

css-watch:
	npm run css:watch

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o bin/ty2shorten-url ./cmd/server

build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)" -o dist/ty2shorten-linux-amd64 ./cmd/server

clean:
	rm -rf bin dist tmp coverage.out

check: css fmt-check test vet lint build

install-dev-tools:
	go install github.com/air-verse/air@$(AIR_VERSION)
