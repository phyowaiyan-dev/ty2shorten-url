# Contributing

Thank you for considering a contribution to Ty2Shorten URL. This project is intentionally small: Go, Gin, GORM, SQLite, server-rendered templates, and embedded assets.

## Ways To Contribute

- Report bugs with clear reproduction steps.
- Suggest focused features.
- Improve documentation.
- Add tests for uncovered behavior.
- Review security-sensitive code carefully.

## Development Setup

```sh
git clone https://github.com/phyowaiyan-dev/ty2shorten-url.git
cd ty2shorten-url
go mod download
cp .env.example .env
make run
```

## Branch Workflow

Create a topic branch from `main`. For larger changes, open an issue first so the design can be discussed before implementation.

## Coding Standards

- Run `go fmt ./...`.
- Keep handlers thin.
- Put business rules in services.
- Put database access in repositories.
- Wrap errors with useful context.
- Do not log passwords, password hashes, session cookies, CSRF tokens, or secrets.
- Keep templates server-rendered and assets embedded.

## Testing Requirements

Before opening a pull request, run:

```sh
make check
go test -race ./...
```

When Staticcheck is not installed:

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## Documentation Requirements

Update README or docs/ whenever behavior, routes, configuration, deployment, or security posture changes.

## Commits

Use concise, descriptive commit messages. Avoid committing local databases, secrets, generated binaries, or private environment files.

## Pull Requests

Pull requests should describe the problem, the implementation, tests run, security impact, and database impact.

## Security Reports

Do not report vulnerabilities in public issues. See [SECURITY.md](SECURITY.md).

## Contributor License

No Contributor License Agreement is currently required. By contributing, you agree that your contribution may be distributed under the project license.
