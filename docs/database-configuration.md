# Database Configuration

Ty2Shorten uses a two-stage startup model.

When `APP_CONFIG_PATH` does not point to an existing bootstrap config, the server starts in database-bootstrap mode and redirects browser routes to `/setup/database`. Static assets and `/health` remain available. After a database configuration is tested and saved, restart the service or start a new process so it connects to the selected database and continues to `/setup`.

Default development path:

```text
./storage/config.json
```

Suggested production path:

```text
/var/lib/ty2shorten/config.json
```

The file is versioned JSON, written atomically, and created with `0600` permissions because MySQL configurations may contain credentials.

SQLite remains the default:

```json
{
  "version": 1,
  "database": {
    "driver": "sqlite",
    "sqlite": {
      "path": "./storage/ty2shorten.db"
    }
  }
}
```

MySQL is optional:

```json
{
  "version": 1,
  "database": {
    "driver": "mysql",
    "mysql": {
      "host": "127.0.0.1",
      "port": 3306,
      "name": "ty2shorten",
      "user": "ty2shorten",
      "password": "secret",
      "tls_mode": "",
      "parameters": {
        "charset": "utf8mb4",
        "parseTime": "true"
      }
    }
  }
}
```

Environment precedence:

- `APP_CONFIG_PATH` chooses the bootstrap config file path.
- `DATABASE_PATH` is only the default SQLite path offered before bootstrap config exists.
- Once bootstrap config exists, it controls the application database.
- The app does not silently fall back to SQLite when MySQL is explicitly configured.

Known limitation: the current process displays a completion page after saving database config and asks for restart instead of hot-swapping into the application setup router.
