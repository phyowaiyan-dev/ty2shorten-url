# Architecture Decisions

## Go

Existing decision: Go provides a small static-style deployment story, strong standard library HTTP support, and good operational simplicity.

## Gin

Existing decision: Gin is lightweight and already used for routing and middleware.

## SQLite

Existing decision: SQLite fits the small single-instance deployment target and avoids operating a separate database service.

## Server-Rendered Templates

Existing decision: Go `html/template` keeps the app simple, secure by default through escaping, and free of a frontend build pipeline.

## Temporary Redirects

Existing decision: App and short-link redirects use HTTP 307 so clients do not permanently cache early configuration mistakes.

## Single Binary

Existing decision: Templates and CSS are embedded so deployment only needs the binary and external SQLite storage.

## No Frontend Framework

Existing decision: The admin and public pages are small enough for server-rendered HTML and CSS.

## Environment Variables

Existing decision: Environment variables are used for infrastructure and secret configuration. Editable product settings live in SQLite after setup.
