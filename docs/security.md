# Security

This document describes implemented controls and known limitations. Vulnerability reporting instructions are in the root [SECURITY.md](../SECURITY.md).

## Implemented Controls

- bcrypt password hashing.
- Strong password minimum of 10 characters.
- Signed auth session cookies.
- `HttpOnly`, `SameSite=Lax`, and production `Secure` cookie settings.
- Session replacement after login.
- Session clearing after logout.
- Session rotation after password change.
- Signed-cookie CSRF tokens for HTML POST forms.
- Login throttling with an in-memory limiter.
- URL validation for `http` and `https` schemes.
- Reserved slug validation.
- GORM parameterized queries.
- `html/template` auto-escaping.
- Request body size limit.
- Security headers and CSP.
- Trusted proxy configuration.
- Generic user-facing errors.
- No password or hash logging.
- MySQL bootstrap config is written outside the app database with `0600` permissions.
- MySQL DSNs are built with driver helpers instead of unsafe password concatenation.
- Database connection-test proofs are signed, short lived, and tied to a hash of the tested config.
- Bootstrap host validation rejects URL schemes, empty/malformed hosts, link-local addresses, cloud metadata IPs, multicast addresses, and unspecified addresses.
- Media uploads validate detected MIME types, reject SVG, enforce file-size limits, and use random filenames.
- Public media serving rejects nested paths and traversal attempts.
- SEO, social metadata, XML sitemap, and JSON-LD are generated through escaping/encoding APIs.
- Audit metadata redacts sensitive keys recursively before storage.
- SQLite WAL and busy timeout.
- HTTP timeouts and graceful shutdown.
- Gin recovery middleware.

## Known Limitations

- Login throttling is in-memory and not shared across instances.
- Privacy and Terms pages are placeholders.
- No formal dependency vulnerability gate beyond GitHub workflow additions.
- No external penetration test has been performed.
- No real VPS deployment has been verified in this audit.
- Connection testing can still reach allowed loopback/private database hosts because local MySQL is a supported deployment pattern.
- Uploaded media replacement does not yet delete unreferenced old files.
- Audit logs are append-only through the app UI, but direct database administrators can modify raw rows.
- Audit entries are not cryptographically tamper-proof.
- Not every important mutation is audited in the same transaction as the business change yet.
