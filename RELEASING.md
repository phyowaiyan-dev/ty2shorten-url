# Releasing

Ty2Shorten URL has not published a stable release yet. Use this process for future tagged releases.

## Version Selection

Use Semantic Versioning once releases begin:

- Patch for bug fixes.
- Minor for backward-compatible features.
- Major for breaking changes.

## Release Checklist

1. Update [CHANGELOG.md](CHANGELOG.md).
2. Run:

   ```sh
   go mod tidy
   go fmt ./...
   go test ./...
   go test -race ./...
   go vet ./...
   staticcheck ./...
   go build ./cmd/server
   ```

3. Build a versioned binary:

   ```sh
   go build -trimpath -ldflags "-s -w -X main.version=v0.3.0 -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o dist/ty2shorten ./cmd/server
   ```

4. Create and push a tag:

   ```sh
   git tag v0.3.0
   git push origin v0.3.0
   ```

5. Verify GitHub release artifacts and checksums.
6. Smoke test the released binary.

## CGO Note

The SQLite driver uses CGO. Release artifacts should be built on compatible native runners or with a verified cross-compilation toolchain.
