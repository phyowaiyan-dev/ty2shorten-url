# Coding Standards

- Use `go fmt ./...`.
- Prefer `rg` for repository searches.
- Keep handlers focused on parsing, service calls, and rendering.
- Put business rules in `internal/services`.
- Put database access in `internal/repositories`.
- Wrap errors with context.
- Avoid global mutable state.
- Use constructors for dependency injection.
- Use `log/slog` and avoid logging secrets.
- Keep templates in `web/templates`.
- Keep CSS in `web/static/css`.
- Use server-rendered HTML and avoid frontend build tooling.
- Tests should use temporary SQLite databases.
- Route behavior changes should update tests and docs.
- Commits should be focused and should not include local DBs, binaries, or secrets.
