# Troubleshooting

## Missing Session Secret

In production, set `SESSION_SECRET` in `/etc/ty2shorten/ty2shorten.env`.

## Database Permission Denied

Check ownership and permissions:

```sh
sudo chown -R ty2shorten:ty2shorten /var/lib/ty2shorten
sudo chmod 0750 /var/lib/ty2shorten
```

## SQLite Locked

Confirm only one service instance uses the database. Check for stale local processes.

## Missing Templates

Templates are embedded. If missing-template errors appear, rebuild the binary and verify `web/assets.go` embed patterns.

## Secure Cookies Not Working

Behind HTTPS termination, confirm Apache sends `X-Forwarded-Proto: https` and the app trusts only the Apache proxy IP.

## Redirect URL Missing

Check `/admin/settings` for Android, Apple, default, and public base URL values.

## Setup Page Unavailable

If setup is complete, `/setup` redirects to `/admin/login`. There is no supported production reset command.

## CGO Build Errors

Install a compiler and SQLite development tooling. On Ubuntu:

```sh
sudo apt-get install -y gcc sqlite3 libsqlite3-dev
```

## Health Check Failures

Check service logs and database access:

```sh
sudo journalctl -u ty2shorten --since "30 minutes ago"
curl -v http://127.0.0.1:8080/health
```

## systemd Restart Loops

Run:

```sh
sudo systemctl status ty2shorten
sudo journalctl -u ty2shorten -n 100
```
