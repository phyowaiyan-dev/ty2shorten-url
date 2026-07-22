# CI/CD

GitHub Actions workflows live under `.github/workflows`.

## CI

`ci.yml` runs on pushes, pull requests, and manual dispatch. It installs SQLite build dependencies, verifies formatting, runs vet, tests, race tests, Staticcheck, builds a Linux AMD64 binary, and uploads artifacts.

## CodeQL

`codeql.yml` performs Go static analysis using GitHub CodeQL.

## Dependency Review

`dependency-review.yml` reviews dependency changes on pull requests.

## Scorecard

`scorecard.yml` runs OpenSSF Scorecard with restricted permissions.

## Release

`release.yml` runs on `v*` tags. Because the SQLite driver requires CGO, releases are limited to verified native Linux AMD64 artifacts until additional runners/toolchains are validated.
