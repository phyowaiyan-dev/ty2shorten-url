# Audit Logs

Audit logs are append-only through the application UI and are visible at:

```text
GET /admin/audit-logs
GET /admin/audit-logs/:id
```

The UI is read-only. There are no POST, PUT, or DELETE audit-log routes.

Stored fields include:

- Actor snapshot fields for future multi-admin support
- Action
- Resource type and ID
- Human-readable summary
- Redacted metadata JSON
- IP address
- User agent
- Request ID
- Creation time

Metadata redaction is recursive and masks keys containing sensitive words such as password, secret, token, cookie, or DSN.

Current audited actions include general settings, SEO settings, branding settings, footer settings, and selected earlier authentication/logging paths.

Known limits:

- Direct database administrators can still modify raw rows.
- The log is not cryptographically tamper-proof.
- Not every mutation is currently written in the same transaction as the business change.
