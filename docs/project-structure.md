# Project Structure

```text
.github/                 GitHub workflows, issue templates, and community helper files
cmd/server/              Application entry point
deployments/             systemd, Apache, install, and update files
docs/                    Project documentation
internal/config/         Environment configuration and validation
internal/database/       SQLite/GORM connection, pragmas, migrations, health ping
internal/handlers/       HTTP handlers
internal/middleware/     Gin middleware and route guards
internal/models/         GORM model definitions
internal/repositories/   Data-access objects
internal/server/         Router and http.Server composition
internal/services/       Business logic
internal/session/        Signed session and CSRF cookie management
internal/validation/     URL and slug validators
storage/                 Local SQLite storage path; only `.gitkeep` should be tracked
web/                     Embedded templates and static CSS
```

Root files include README, TODO, Makefile, license, changelog, governance, contribution, support, and security policy documents.
