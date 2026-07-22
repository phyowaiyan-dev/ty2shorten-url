# Database Migration

The application currently uses GORM `AutoMigrate` at startup for both SQLite and MySQL.

Current migrated models:

- `AdminUser`
- `AppSetting`
- `ShortLink`
- `AuditLog`

`AutoMigrate` is convenient for early alpha development and additive schema changes, but it is not a complete replacement for versioned production migrations. Before frequent production upgrades, add explicit migration files with rollback notes, preflight checks, and backup requirements.

Compatibility notes:

- SQLite is configured with WAL mode, foreign keys, a busy timeout, and a conservative connection pool.
- MySQL uses driver configuration helpers for DSN escaping, `utf8mb4`, `parseTime=true`, safe timezone handling, and bounded connection pools.
- Boolean fields are represented through GORM and covered by the existing SQLite test suite.
- Click counters use GORM atomic updates and should be rechecked under MySQL integration tests before a production MySQL launch.

Before any production migration:

1. Back up the database.
2. Run the release binary against a staging copy.
3. Run `make check`.
4. Run optional MySQL integration tests when MySQL is used.
5. Verify `/health`, setup/login, redirects, and admin settings.
