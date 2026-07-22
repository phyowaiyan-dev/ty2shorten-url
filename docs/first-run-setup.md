# First-Run Setup

First-run setup has two stages.

## Stage A: Database Bootstrap

When no bootstrap config exists at `APP_CONFIG_PATH`, the server allows only:

- `GET /setup/database`
- `POST /setup/database/test`
- `POST /setup/database`
- `GET /health`
- `GET /static/*`

Other browser routes redirect to `/setup/database`.

SQLite is selected by default. MySQL can be selected when the operator provides host, port, database name, username, password, TLS mode, and optional parameters. Final submission requires a signed connection-test proof for the exact configuration.

Known limit: after saving bootstrap config, the current process asks for restart instead of hot-swapping into the application setup router.

## Stage B: Application Setup

When the selected application database has no completed settings row, the setup gate allows only:

- `GET /setup`
- `POST /setup`
- `GET /health`
- `GET /static/*`

Other browser routes redirect to `/setup`.

## Required Fields

- Site name
- Site description
- Administrator name
- Administrator email
- Administrator password
- Password confirmation
- Android app URL
- Apple app URL

Optional fields:

- Default URL
- Support email

The public base URL is initialized from `BASE_URL`.

## Persistence

Setup creates the first administrator and settings row in one database transaction. `IsSetupCompleted` is set only after the setup data is stored.

## Lockout

After setup completes, `/setup` redirects to `/admin/login` or rejects repeated POST attempts.

## Reset Considerations

No supported reset command exists. Manual database edits or deletion can destroy production data and should only occur during development or a planned maintenance process.
