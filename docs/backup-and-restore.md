# Backup and Restore

SQLite uses WAL mode, so use SQLite's backup command for live backups.

## Backup

```sh
sudo sqlite3 /var/lib/ty2shorten/ty2shorten.db ".backup '/var/lib/ty2shorten/ty2shorten.$(date -u +%Y%m%d%H%M%S).db'"
```

## Restore

```sh
sudo systemctl stop ty2shorten
sudo cp /path/to/backup.db /var/lib/ty2shorten/ty2shorten.db
sudo chown ty2shorten:ty2shorten /var/lib/ty2shorten/ty2shorten.db
sudo chmod 0640 /var/lib/ty2shorten/ty2shorten.db
sudo systemctl start ty2shorten
```

## Notes

- Do not commit database files.
- Test restore procedures before production launch.
- Keep backups outside the application directory when possible.
