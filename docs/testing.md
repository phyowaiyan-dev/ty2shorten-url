# Testing

## Automated Tests

Tests cover configuration, health, database bootstrap, setup, login, logout, admin protection, settings, branding uploads, SEO metadata, robots, sitemap, footer settings, audit logs, short links, redirects, QR rendering, CSRF rejection, and related validation.

Tests use temporary SQLite databases and must not depend on production data.

## Commands

```sh
go mod tidy
go fmt ./...
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
go build ./cmd/server
npm run css
make build
make check
```

Optional MySQL integration tests are opt-in and should use a disposable database:

```sh
TY2_MYSQL_INTEGRATION=1 go test -tags mysqlintegration ./...
```

Coverage:

```sh
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Manual Smoke Tests

- Fresh setup flow.
- Fresh `/setup/database` SQLite flow.
- MySQL connection-test flow against a disposable MySQL database.
- Login and logout.
- Settings update.
- Branding upload and public `/media/:name` serving.
- SEO settings, `/robots.txt`, and `/sitemap.xml`.
- Footer settings.
- Audit-log list, filters, and detail page.
- Link create/edit/disable/delete.
- `/android`, `/apple`, `/get`, `/r/:slug`.
- QR code rendering.
- CSRF rejection.
- Unauthorized admin access.
- Missing destination fallback.
- `/health`.

## CI Checks

GitHub Actions runs formatting verification, CSS build, vet, tests, race tests, Staticcheck, application build, and artifact upload. MySQL integration coverage is intended to stay optional and disposable.

Air is a local development tool only. CI should keep using direct commands such as `go test ./...`, `go vet ./...`, `npm run css`, and `go build ./cmd/server`; CI does not need to install or run Air.

## Development Workflow Smoke Checks

When changing the live-reload workflow, manually verify:

- `make dev` starts Air and the Tailwind/daisyUI watcher.
- Ctrl+C stops both processes and releases the local port.
- Editing a `.go` file triggers an Air rebuild.
- Editing a template under `web/templates/` triggers an Air rebuild.
- Editing `web/assets/src/app.css` rebuilds `web/static/css/app.css`, and that generated embedded CSS change triggers Air.
- Creating or touching SQLite files under `storage/` does not trigger Air.
- Production commands such as `make build` do not require Air or Node.js at runtime.
