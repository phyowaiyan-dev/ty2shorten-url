# Configuration

Ty2Shorten URL reads configuration directly from environment variables. No `.env` library is required.

| Variable | Default | Required | Sensitive | Production recommendation |
| --- | --- | --- | --- | --- |
| `APP_ENV` | `development` | Yes | No | Set to `production`. |
| `APP_HOST` | `127.0.0.1` | Yes | No | Keep loopback behind Apache. |
| `APP_PORT` | `8722` | Yes | No | Keep loopback behind Apache; `8722` avoids many services that commonly use `8080`. |
| `DATABASE_PATH` | `./storage/ty2shorten.db` | Legacy/dev | No | Default SQLite path before bootstrap config exists. |
| `APP_CONFIG_PATH` | `./storage/config.json` | Yes | May contain secrets | Use `/var/lib/ty2shorten/config.json`; file mode is `0600`. |
| `MEDIA_STORAGE_PATH` | `./storage/media` | Yes | No | Use `/var/lib/ty2shorten/media`. |
| `SESSION_SECRET` | empty | Production | Yes | Long random value. |
| `BASE_URL` | `http://localhost:8722` | Yes | No | Public HTTPS origin. |
| `TRUSTED_PROXIES` | `127.0.0.1,::1` | Yes | No | Trust only Apache loopback addresses. |
| `APP_VERSION` | `dev` | No | No | Inject through build or environment. |
| `APP_COMMIT` | `unknown` | No | No | Inject through build or environment. |
| `APP_BUILD_TIME` | `unknown` | No | No | Inject through build or environment. |

## Development Example

```env
APP_ENV=development
APP_HOST=127.0.0.1
APP_PORT=8722
DATABASE_PATH=./storage/ty2shorten.db
APP_CONFIG_PATH=./storage/config.json
MEDIA_STORAGE_PATH=./storage/media
SESSION_SECRET=
BASE_URL=http://localhost:8722
TRUSTED_PROXIES=127.0.0.1,::1
APP_VERSION=dev
APP_COMMIT=unknown
APP_BUILD_TIME=unknown
```

## Production Example

```env
APP_ENV=production
APP_HOST=127.0.0.1
APP_PORT=8080
DATABASE_PATH=/var/lib/ty2shorten/ty2shorten.db
APP_CONFIG_PATH=/var/lib/ty2shorten/config.json
MEDIA_STORAGE_PATH=/var/lib/ty2shorten/media
SESSION_SECRET=replace-with-a-long-random-secret
BASE_URL=https://example.com
TRUSTED_PROXIES=127.0.0.1,::1
APP_VERSION=v0.3.0
APP_COMMIT=replace-with-commit
APP_BUILD_TIME=2026-07-22T00:00:00Z
```

Do not commit real production environment files.

See [database configuration](database-configuration.md) for how bootstrap config takes precedence over `DATABASE_PATH`.
