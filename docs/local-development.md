# Local Development

## Prerequisites

- Go 1.26.1 or compatible version.
- CGO-capable compiler.
- SQLite-compatible build environment.
- Node.js and npm for Tailwind CSS and daisyUI.
- Staticcheck for linting.
- Air for Go live reload.

## Setup

```sh
git clone https://github.com/phyowaiyan-dev/ty2shorten-url.git
cd ty2shorten-url
go mod download
npm install
cp .env.example .env
make install-dev-tools
make run
```

Open `http://127.0.0.1:8722`.

If `air` is not found after installation, add the Go binary directory to your shell path:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

For Zsh:

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

Manual Air installation uses the same pinned version as the Makefile:

```sh
go install github.com/air-verse/air@v1.67.1
```

## Running

`make dev` is the main local development command. `make run` is an alias for the same live-reload workflow.

```sh
make dev
```

This starts both:

- Air, which rebuilds and restarts the Go server.
- The Tailwind/daisyUI watcher, which runs `npm run css:watch` with `--watch=always` and rebuilds `web/static/css/app.css`.

Use Ctrl+C to stop both processes. The local server listens on `http://127.0.0.1:8722` unless `APP_HOST` or `APP_PORT` is changed.

For one-time execution without Air:

```sh
make run-once
```

## CSS

Compiled CSS is embedded from `web/static/css/app.css`.

```sh
make css
make css-watch
```

Tailwind scans `web/templates/**/*.html`, `cmd/**/*.go`, and `internal/**/*.go`. Air ignores `web/assets/` and watches the generated embedded CSS under `web/static/`, so editing source CSS is handled by Tailwind first and then Air rebuilds once when the generated CSS changes.

## Test

```sh
make test
make test-race
make vet
make lint
```

## Build

```sh
make build
```

## GoLand Setup

Open the repository as a Go project. Use the module Go version from `go.mod`. Configure run environment variables from `.env.example`.

## Common Local Errors

- `air: command not found`: run `make install-dev-tools`, then ensure `$(go env GOPATH)/bin` is on `PATH`.
- Air rebuilding continuously: do not write generated files into watched source directories. Air excludes `tmp/`, `bin/`, `dist/`, `storage/`, `uploads/`, `node_modules/`, `.git/`, and `web/assets/`.
- Port already in use: stop the existing process or run with a different port, for example `APP_PORT=8723 make dev`.
- CSS watcher missing: run `npm install` and confirm `npm run css:watch` works.
- Node dependencies missing: run `npm install`; npm is the package manager for this repository.
- Templates not triggering rebuild: confirm the file is under `web/templates/` and has a watched extension such as `.html`.
- SQLite changes triggering rebuild: SQLite files should be under `storage/` and match `.db`, `.db-shm`, or `.db-wal`; these are ignored by Air.
- Orphaned Air processes: stop the dev workflow with Ctrl+C; if needed, find leftover listeners with `lsof -nP -iTCP:8722 -sTCP:LISTEN`.
- Go build failure: Air keeps the previous process stopped when a new build is invalid; fix the compile error and save again.
- Ctrl+C does not stop both processes: use `scripts/dev.sh` through `make dev`, not manual shell backgrounding.
- Browser auto-refresh: Air restarts the server but does not automatically refresh the browser.
- `SESSION_SECRET` warning: expected in development; required only in production.
- CGO errors: install Xcode Command Line Tools on macOS or `gcc` on Linux.
- SQLite locked: stop other local server processes using the same database file.
- Setup redirects unexpectedly: delete the local ignored database only when you intentionally want a fresh setup flow.
