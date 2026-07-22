# Ty2Shorten URL

[![CI](https://github.com/phyowaiyan-dev/ty2shorten-url/actions/workflows/ci.yml/badge.svg)](https://github.com/phyowaiyan-dev/ty2shorten-url/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Ty2Shorten URL is a lightweight Go URL shortener and mobile app redirect service. It uses Gin, GORM, SQLite or MySQL, server-rendered templates, embedded CSS/templates, secure cookie sessions, and a single-binary deployment model.

> Project status: **Alpha**. Core application flows are implemented and verified locally. Real Ubuntu VPS deployment behind Apache still needs environment-specific validation before production use.

## Screenshot

Screenshot or preview image: pending.

## Overview

The application provides a public landing page, database bootstrap onboarding, first-run admin setup, administrator dashboard, app-store redirects, device detection, custom short links, configurable SEO, branding media, footer settings, and read-only audit logs. It is designed for small deployments where a single Go binary and SQLite database are easy to operate, while still allowing MySQL when an operator already has database infrastructure.

## Why Ty2Shorten URL?

- Small operational footprint.
- Tailwind/daisyUI CSS build used only at development/build time; Node.js is not required at runtime.
- No Redis, Docker, Node.js runtime, React, or Vue requirement.
- SQLite storage by default, with optional MySQL selected during database bootstrap.
- Embedded templates and CSS in the final binary.
- Conservative Ubuntu/systemd/Apache deployment files.

## Features

- First-run setup wizard.
- Administrator login, logout, and password change.
- Settings management for site metadata, app URLs, fallback URL, support email, and public base URL.
- Database-bootstrap page for SQLite/MySQL selection before app setup.
- Dynamic `robots.txt` and `sitemap.xml`.
- Open Graph, Twitter/X, favicon, Apple touch icon, and JSON-LD metadata.
- Branding media uploads served from controlled `/media/:name` URLs.
- Configurable public footer.
- Short-link list, create, edit, enable/disable, delete, click counts, and copyable public URLs.
- Read-only audit log list and detail pages.
- `/android`, `/apple`, `/get`, and `/r/:slug` redirects.
- QR code generation for the `/get` URL.
- Health endpoint with database check and safe version metadata.
- Structured request logging with request IDs.
- CSRF protection on state-changing HTML forms.
- SQLite WAL mode and busy timeout.

## Current Project Status

This repository is suitable for continued development and internal MVP testing. It is not yet a production-ready public release because VPS deployment, TLS termination, backup automation, legal pages, and a final security review remain pending.

See [TODO.md](TODO.md) for the actionable roadmap.

## Technology Stack

- Go 1.26.1
- Gin
- GORM
- SQLite through `github.com/mattn/go-sqlite3` via `gorm.io/driver/sqlite`
- MySQL through `github.com/go-sql-driver/mysql` via `gorm.io/driver/mysql`
- Go `html/template`
- Go `embed`
- Tailwind CSS and daisyUI for compiled admin/public CSS
- `log/slog`
- bcrypt
- go-qrcode

SQLite uses CGO. Linux builds need a C compiler and SQLite-compatible build environment.

## Architecture

Request flow:

```text
Apache reverse proxy -> Go http.Server -> Gin middleware -> handlers -> services -> repositories -> SQLite
```

Templates and static assets are embedded under `web/`, so the deployed binary does not need template files beside it.

More detail:

- [Architecture](docs/architecture.md)
- [Project structure](docs/project-structure.md)
- [Database](docs/database.md)
- [Security](docs/security.md)

## Project Structure

```text
cmd/server              executable entry point
internal/config         environment configuration
internal/database       SQLite/GORM startup and migrations
internal/handlers       HTTP handlers
internal/middleware     request logging, security, setup/auth/CSRF guards
internal/models         GORM models
internal/repositories   database access
internal/server         router and HTTP server composition
internal/services       business logic
internal/session        signed cookies and CSRF tokens
internal/validation     reusable validators
web                     embedded templates and CSS
deployments             systemd, Apache, install, and update files
docs                    project documentation
```

## Requirements

- Go 1.26.1 or newer compatible version.
- CGO-capable build environment.
- For Ubuntu deployment: `gcc`, `sqlite3`, Apache, systemd, and Certbot for TLS.

Supported operating systems currently verified locally:

- macOS ARM64 for development and tests.

Expected deployment target:

- Ubuntu Linux AMD64 behind Apache.

## Quick Start

```sh
git clone https://github.com/phyowaiyan-dev/ty2shorten-url.git
cd ty2shorten-url
go mod download
npm install
cp .env.example .env
make install-dev-tools
make run
```

Open:

```text
http://127.0.0.1:8722
```

On an empty database, browser traffic redirects to `/setup`.

## Configuration

| Variable | Default | Required | Sensitive | Notes |
| --- | --- | --- | --- | --- |
| `APP_ENV` | `development` | Yes | No | `development`, `production`, or `test`. |
| `APP_HOST` | `127.0.0.1` | Yes | No | Listen host. |
| `APP_PORT` | `8722` | Yes | No | Local default; deployment example uses `8080`. |
| `DATABASE_PATH` | `./storage/ty2shorten.db` | Legacy/dev | No | Used as the default SQLite path before bootstrap config exists. |
| `APP_CONFIG_PATH` | `./storage/config.json` | Yes | May contain secrets | Versioned bootstrap database config path. Use `/var/lib/ty2shorten/config.json` in production. |
| `MEDIA_STORAGE_PATH` | `./storage/media` | Yes | No | Directory for uploaded logo/favicon/social images. |
| `SESSION_SECRET` | empty | Production | Yes | Required in production. |
| `BASE_URL` | `http://localhost:8722` | Yes | No | Initial public base URL for setup. |
| `TRUSTED_PROXIES` | `127.0.0.1,::1` | Yes | No | Do not use wildcard trust. |
| `APP_VERSION` | `dev` | No | No | Safe health metadata. |
| `APP_COMMIT` | `unknown` | No | No | Safe health metadata. |
| `APP_BUILD_TIME` | `unknown` | No | No | Safe health metadata. |

See [configuration docs](docs/configuration.md).

## First-Run Setup

When no bootstrap database configuration exists, browser traffic redirects to `/setup/database`. Choose SQLite or MySQL, test the connection, and save the configuration. The application then connects to the configured database and continues to `/setup`, where the first administrator and settings row are created in a transaction. After setup completes, `/setup` is locked out and redirects to `/admin/login`.

See [first-run setup docs](docs/first-run-setup.md) and [database configuration docs](docs/database-configuration.md).

## Running Locally

```sh
make run
```

`make run` is the live-reload development workflow. It runs Air for Go rebuild/restart and the Tailwind/daisyUI watcher for compiled embedded CSS. Use `make run-once` when you want a single `go run ./cmd/server` process without Air.

If `air` is not found:

```sh
make install-dev-tools
export PATH="$PATH:$(go env GOPATH)/bin"
```

Useful checks:

```sh
curl http://127.0.0.1:8722/health
```

See [local development docs](docs/local-development.md).

## Testing

```sh
go mod tidy
go fmt ./...
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
go build ./cmd/server
```

Or:

```sh
make check
```

See [testing docs](docs/testing.md).

## Building

Local build:

```sh
make build
```

Versioned build:

```sh
go build -trimpath \
  -ldflags "-s -w -X main.version=dev -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/ty2shorten-url ./cmd/server
```

## Deployment

Deployment files are provided for an Ubuntu VPS behind Apache:

- [Deployment overview](docs/deployment.md)
- [Apache reverse proxy](docs/apache.md)
- [systemd service](docs/systemd.md)
- [Backup and restore](docs/backup-and-restore.md)
- [Release process](docs/release-process.md)

Suggested production paths:

- Binary: `/opt/ty2shorten/ty2shorten`
- Database: `/var/lib/ty2shorten/ty2shorten.db`
- Environment: `/etc/ty2shorten/ty2shorten.env`

## Docker Status

Docker is intentionally not part of the current implementation. The project is designed for single-binary deployment with an external SQLite file.

## API and Routes

See [API and routes](docs/api-routes.md).

## Security

Implemented controls include bcrypt password hashing, signed session cookies, `HttpOnly`, `SameSite=Lax`, production secure cookies, CSRF protection, login throttling, URL validation, reserved slug blocking, security headers, request size limits, and trusted proxy configuration.

Security reporting instructions are in [SECURITY.md](SECURITY.md). Implementation details are in [docs/security.md](docs/security.md).

## Backup and Restore

SQLite remains external to the binary and must be backed up. Use SQLite online backup commands rather than copying a live database file directly.

See [backup and restore docs](docs/backup-and-restore.md).

## Updating

Use `deployments/update.sh` after a new binary is built. The script backs up the binary and database, restarts the service, verifies `/health`, and rolls back the binary when health fails.

See [deployment docs](docs/deployment.md).

## Troubleshooting

See [troubleshooting docs](docs/troubleshooting.md).

## Roadmap

See [TODO.md](TODO.md) and [future vision](docs/future-vision.md).

## Documentation

Start with [docs/README.md](docs/README.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md), [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md), and [docs/coding-standards.md](docs/coding-standards.md).

## Security Reports

Do not open public issues for vulnerabilities. See [SECURITY.md](SECURITY.md).

## License

MIT License. See [LICENSE](LICENSE).

## Contributors

See [CONTRIBUTORS.md](CONTRIBUTORS.md).

## Acknowledgements

See [ACKNOWLEDGEMENTS.md](ACKNOWLEDGEMENTS.md).
