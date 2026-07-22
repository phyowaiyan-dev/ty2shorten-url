# Documentation

This directory contains project and operations documentation for Ty2Shorten URL.

- [Architecture](architecture.md): responsibilities, request flow, dependency direction, and deployment shape.
- [Project structure](project-structure.md): package and directory guide.
- [Configuration](configuration.md): environment variables and examples.
- [Local development](local-development.md): setup, run, test, and common local issues.
- [First-run setup](first-run-setup.md): setup wizard behavior and lockout.
- [Database configuration](database-configuration.md): SQLite/MySQL bootstrap config and precedence.
- [Database migration](database-migration.md): current AutoMigrate behavior and migration limits.
- [Authentication](authentication.md): sessions, cookies, login, logout, throttling, and password changes.
- [Redirect engine](redirect-engine.md): app-store redirects, device detection, and short-link resolution.
- [Short links](short-links.md): slug rules, validation, active state, clicks, and limitations.
- [SEO](seo.md): configurable metadata and structured data.
- [Robots and sitemap](robots-and-sitemap.md): dynamic crawler resources.
- [Social sharing](social-sharing.md): Open Graph and Twitter/X behavior.
- [Branding](branding.md): logo, favicon, and social image configuration.
- [Media uploads](media-uploads.md): storage, serving, validation, and limits.
- [Footer](footer.md): configurable public footer.
- [Admin dashboard](admin-dashboard.md): admin UI structure.
- [Audit logs](audit-logs.md): append-only application audit records and UI.
- [Database](database.md): SQLite behavior, schema, WAL, backups, and migrations.
- [Security](security.md): implemented controls and known limitations.
- [Testing](testing.md): automated checks and manual smoke tests.
- [Deployment](deployment.md): Ubuntu deployment flow.
- [Apache](apache.md): reverse-proxy configuration and headers.
- [systemd](systemd.md): service file and hardening.
- [CI/CD](ci-cd.md): GitHub Actions workflows.
- [Backup and restore](backup-and-restore.md): SQLite backup procedures.
- [Troubleshooting](troubleshooting.md): common problems and checks.
- [Release process](release-process.md): versioning and releases.
- [API routes](api-routes.md): route reference.
- [Coding standards](coding-standards.md): implementation conventions.
- [Decisions](decisions.md): lightweight architecture decision log.
- [Future vision](future-vision.md): possible future product directions.
