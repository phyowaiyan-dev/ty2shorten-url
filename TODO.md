# Ty2Shorten URL Roadmap

## Current Status

Ty2Shorten URL is an alpha-stage single-binary Go application. The core setup, authentication, redirect, short-link, admin, SQLite/MySQL bootstrap, SEO, media branding, footer, audit-log, embedded-template, deployment-file, and CI foundations are implemented and verified locally. Production infrastructure validation on a real Ubuntu VPS behind Apache remains pending.

## Critical Before Production

- [ ] Run an end-to-end deployment on a clean Ubuntu VPS behind Apache.
- [ ] Replace placeholder Privacy Policy and Terms pages with real legal content.
- [ ] Confirm Certbot HTTPS termination and secure cookie behavior on the public domain.
- [ ] Confirm `/etc/ty2shorten/ty2shorten.env` contains a long random `SESSION_SECRET`.
- [ ] Create and test a recurring SQLite backup process.
- [ ] Create and test MySQL backup and restore runbooks when MySQL is selected.
- [ ] Perform a manual security review before handling real user traffic.

## Security

- [x] Passwords are hashed with bcrypt.
- [x] Sessions are signed cookies containing only the admin ID, expiry, and nonce.
- [x] Session cookies are `HttpOnly`, `SameSite=Lax`, and `Secure` in production.
- [x] Login and password-change flows rotate or replace session cookies.
- [x] Logout clears the session cookie.
- [x] Setup, login, logout, settings, links, and account POST forms require CSRF tokens.
- [x] Login throttling is implemented with a small in-memory limiter.
- [x] URL validation rejects unsafe schemes.
- [x] MySQL bootstrap validation rejects malformed, link-local, multicast, unspecified, and cloud metadata hosts.
- [x] Media uploads validate detected MIME type, enforce per-kind size limits, and store randomized filenames outside embedded assets.
- [x] Audit metadata is recursively redacted before persistence.
- [x] Reserved slugs are blocked.
- [x] Gin trusted proxies are configurable and unsafe wildcard proxy values are rejected.
- [ ] Add persistent or distributed throttling before multi-instance deployment.
- [ ] Add automated dependency vulnerability scanning beyond GitHub dependency review.
- [ ] Decide whether to add security headers such as HSTS at Apache or application level.

## Testing

- [x] Config tests cover defaults and production secret validation.
- [x] Health endpoint test covers safe version metadata.
- [x] Route tests cover setup protection, setup success/failure, login/logout, admin protection, redirects, device detection, short links, settings, password change, CSRF rejection, duplicate slugs, and QR rendering.
- [x] Tests use temporary SQLite databases.
- [x] Bootstrap tests cover SQLite defaults, MySQL validation, DSN construction, and proof verification.
- [x] SEO tests cover metadata, robots.txt, sitemap.xml, and JSON-LD rendering.
- [x] Branding/media tests cover valid upload, fake image rejection, oversized favicon rejection, public media serving, and public branding rendering.
- [x] Footer settings tests cover updates, disabling, and unsafe URL rejection.
- [x] Audit-log tests cover settings audit creation, filtering, detail view, redaction expectations, and read-only route behavior.
- [x] `go test -race ./...` passes locally.
- [ ] Add explicit session-expiration tests.
- [ ] Add explicit database-health-failure tests.
- [ ] Add browser-level accessibility and responsive-layout checks.
- [ ] Add coverage reporting thresholds only after baseline coverage is measured.

## Core Features

- [x] First-run setup wizard.
- [x] Database-selection bootstrap wizard for SQLite and MySQL.
- [x] MySQL connection testing uses a signed short-lived proof tied to the tested configuration.
- [x] Administrator login and logout.
- [x] Password change form.
- [x] Public landing page with app buttons and QR code.
- [x] `/android`, `/apple`, `/get`, and `/r/:slug` redirects.
- [ ] Replace Privacy and Terms placeholders.
- [ ] Add support for multiple administrators.

## Admin Dashboard

- [x] Dashboard metrics show app URL status, active links, total clicks, and environment.
- [x] Settings management is implemented.
- [x] Branding media settings are implemented.
- [x] SEO settings are implemented.
- [x] Footer settings are implemented.
- [x] Read-only audit-log list and detail pages are implemented.
- [x] Short-link list/create/edit/enable/disable/delete is implemented.
- [x] Click counts are displayed.
- [x] Public short URLs are copyable as read-only fields.
- [ ] Add pagination and search for large link sets.
- [ ] Add richer analytics views.

## Redirect Engine

- [x] Android and Apple redirects use HTTP 307.
- [x] `/get` detects Android, iPhone, iPad, and iPod with simple User-Agent matching.
- [x] Unknown or desktop devices redirect to `/`.
- [x] Missing app destinations render a fallback page.
- [x] Active short links redirect and increment click counts.
- [x] Inactive, missing, invalid, and reserved slugs return an HTML 404.
- [ ] Consider bot-specific handling after real traffic is observed.
- [ ] Consider link expiration and scheduled activation.

## Database and Data Integrity

- [x] SQLite directory is created automatically.
- [x] SQLite remains the default database.
- [x] MySQL is supported through bootstrap configuration and GORM's MySQL driver.
- [x] Bootstrap database configuration is stored outside the application database with atomic writes and `0600` permissions.
- [x] GORM AutoMigrate runs on startup.
- [x] WAL mode, foreign keys, busy timeout, and constrained connection pool are configured.
- [x] Setup creates admin and settings in a transaction.
- [ ] Add explicit migration/versioning strategy before schema changes become frequent.
- [ ] Consider soft deletion for frequently used links.

## Performance and Reliability

- [x] HTTP server uses read-header, read, write, and idle timeouts.
- [x] Graceful shutdown handles SIGINT and SIGTERM.
- [x] Static assets and templates are embedded.
- [x] Local Air live reload runs alongside Tailwind/daisyUI watch mode.
- [x] SQLite busy timeout is configured.
- [ ] Load test expected VPS traffic.
- [ ] Add request metrics or structured log aggregation guidance.

## Observability

- [x] Structured request logging includes request ID, method, path, status, bytes, client IP, and duration.
- [x] Startup and shutdown are logged.
- [x] Authentication failures are logged without passwords.
- [x] Settings and link changes are logged without sensitive values.
- [x] Redirect resolution failures are logged safely.
- [x] `/health` returns status, service, version, commit, and build time.
- [x] Audit logs are persisted for general, SEO, branding, and footer settings changes.
- [ ] Audit every important mutation in a same-transaction policy.
- [ ] Add log retention and rotation guidance for production.

## User Interface

- [x] Server-rendered setup, login, dashboard, settings, links, account, fallback, and 404 pages.
- [x] Tailwind/daisyUI CSS is compiled locally and embedded for runtime.
- [x] Admin pages include responsive sidebar/drawer navigation markers.
- [x] Public landing page can render configured logo, favicon, social image metadata, and footer content.
- [x] Delete confirmation uses HTML and works with CSP.
- [ ] Run manual accessibility review with keyboard and screen-reader checks.
- [ ] Add screenshots to documentation.

## Deployment

- [x] systemd service file is present.
- [x] Apache reverse-proxy config is present.
- [x] Conservative install and update scripts are present.
- [x] README and docs describe Ubuntu deployment.
- [ ] Test install and update scripts on a real VPS.
- [ ] Verify rollback behavior during a simulated failed health check.

## CI/CD

- [x] CI runs formatting, vet, tests, race tests, Staticcheck, and Linux AMD64 build.
- [x] CI includes CSS build and optional MySQL integration-test scaffolding.
- [x] CodeQL workflow is planned in this audit pass.
- [x] Dependency review, release, and Scorecard workflows are planned in this audit pass.
- [ ] Confirm release workflow on the first version tag.

## Documentation

- [x] README rewritten for open-source alpha status.
- [x] docs/ structure added.
- [x] Database bootstrap, SEO, robots/sitemap, social sharing, branding, media upload, footer, admin dashboard, and audit-log docs added.
- [x] Root contributing, security, support, changelog, governance, releasing, and acknowledgements documents added.
- [ ] Add screenshots after the first stable UI pass.
- [ ] Keep docs synchronized with route and deployment changes.

## Open-Source Readiness

- [x] MIT License added.
- [x] Issue templates and pull-request template added.
- [x] Contributor, support, security, governance, and code-of-conduct files added.
- [ ] Replace placeholder maintainer contacts when a public contact is available.
- [ ] Enable GitHub private vulnerability reporting.

## Future Product Vision

- [ ] Multiple administrators with roles and permissions.
- [ ] Custom domains.
- [ ] Link expiration and scheduled activation.
- [ ] Password-protected links.
- [ ] Detailed analytics and geographic reports.
- [ ] QR customization.
- [ ] API tokens and REST API.
- [ ] Webhooks.
- [ ] Import and export.
- [ ] PostgreSQL support.
- [ ] High-availability deployment.
- [ ] Redis-backed rate limiting.
- [ ] Multi-tenant SaaS mode.
- [ ] Branded landing pages and localization.

## Milestones

- [x] v0.1.0 - Internal MVP foundation: config, database, embedded assets, health, setup, auth, redirects, admin basics.
- [x] v0.2.0 - Admin and redirect completion: settings, links, account password change, QR code, deployment files.
- [x] v0.3.0 - Product hardening foundation: SQLite/MySQL bootstrap, SEO, media branding, footer configuration, audit logs, Tailwind/daisyUI CSS.
- [ ] v0.4.0 - Production validation: VPS validation, backup automation, legal placeholders replaced, manual security review.
- [ ] v1.0.0 - Stable open-source release: documented operations, tested release workflow, clear upgrade path, no critical blockers.

## Completed Work

- [x] Lightweight Go/Gin/GORM/SQLite server-rendered architecture.
- [x] Single-binary embedded templates and CSS.
- [x] First-run setup lockout after completion.
- [x] Cookie sessions and CSRF.
- [x] App-store redirects and custom short links.
- [x] Admin settings, links, dashboard, and password change pages.
- [x] Database bootstrap configuration, dynamic crawler resources, media branding uploads, footer settings, and audit-log UI.
- [x] Ubuntu/systemd/Apache deployment materials.
- [x] GitHub CI foundation.
