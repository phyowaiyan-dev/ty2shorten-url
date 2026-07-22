# systemd Service

The provided unit is `deployments/ty2shorten.service`.

## Service User

The install script creates a dedicated `ty2shorten` system user.

## Important Paths

- Working directory: `/opt/ty2shorten`
- Binary: `/opt/ty2shorten/ty2shorten`
- Environment: `/etc/ty2shorten/ty2shorten.env`
- Writable data: `/var/lib/ty2shorten`

## Hardening

The unit uses:

- `NoNewPrivileges=true`
- `PrivateTmp=true`
- `ProtectSystem=strict`
- `ProtectHome=true`
- `ReadWritePaths=/var/lib/ty2shorten`
- `StateDirectory=ty2shorten`
- restrictive `UMask`

These settings should still allow SQLite to write to the configured data directory.
