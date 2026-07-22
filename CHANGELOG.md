# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project intends to use Semantic Versioning after the first tagged release.

## [Unreleased]

### Added

- First-run setup wizard.
- Administrator authentication and password changes.
- Settings and short-link management.
- App-store redirects and device detection.
- QR code generation for `/get`.
- SQLite storage with GORM migrations.
- Embedded server-rendered templates and CSS.
- systemd, Apache, install, and update deployment files.
- GitHub Actions CI foundation.
- Open-source documentation and community files.
- Two-stage database bootstrap with SQLite default and optional MySQL.
- Versioned bootstrap database configuration stored outside the application database.
- Dynamic `robots.txt` and `sitemap.xml`.
- Configurable SEO, Open Graph, Twitter/X, favicon, Apple touch icon, and JSON-LD metadata.
- Branding media uploads for logo, dark logo, favicon, Apple touch icon, and social-share image.
- Configurable public footer.
- Read-only audit-log list, filters, and detail pages.
- Tailwind CSS and daisyUI build pipeline for embedded compiled CSS.
- First-party analytics with visitor sessions, public page-view tracking, redirect-event tracking, Android/iOS redirect detail tables, IP network/hash fields, device/browser/OS inference, referrer summaries, date filtering, CSV export, admin settings, HMAC or network-only IP handling, session detail pages, and manual retention cleanup.

### Changed

- Health endpoint exposes safe version metadata.
- Default development port changed to `8722` to avoid the more commonly occupied `8080`.

### Security

- Added signed session cookies, CSRF protection, bcrypt password hashing, login throttling, security headers, and URL validation.
- Added MySQL connection-test validation and signed bootstrap proof verification.
- Added media upload MIME/size validation with randomized filenames outside embedded assets.
- Added recursive audit metadata redaction.
