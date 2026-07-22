# Release Process

Ty2Shorten URL has not shipped a stable release yet.

## Versioning

Use Semantic Versioning after the first tagged release.

## Flow

1. Update `CHANGELOG.md`.
2. Run the full validation sequence.
3. Build with version metadata.
4. Tag with `v*`, for example `v0.3.0`.
5. Push the tag.
6. Verify GitHub release workflow artifacts and checksums.
7. Smoke test the released binary.

## Unsupported Targets

The current SQLite driver requires CGO. Release automation currently documents Linux AMD64 as the verified artifact target. Additional native runners or cross-compilers should be validated before publishing ARM64 or macOS artifacts.
