# Deployment

Target deployment is an Ubuntu VPS behind Apache.

## Build

```sh
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w -X main.version=v0.3.0 -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dist/ty2shorten-linux-amd64 ./cmd/server
```

## Install

```sh
sudo deployments/install.sh ./dist/ty2shorten-linux-amd64
```

The script creates or uses:

- User: `ty2shorten`
- Binary: `/opt/ty2shorten/ty2shorten`
- Database directory: `/var/lib/ty2shorten`
- Environment file: `/etc/ty2shorten/ty2shorten.env`
- Service: `ty2shorten.service`

## Apache and TLS

See [apache.md](apache.md). Use Certbot:

```sh
sudo certbot --apache -d app.phyowaiyan.com
```

## Health Verification

```sh
curl -f http://127.0.0.1:8722/health
curl -f https://app.phyowaiyan.com/health
```

## Updates and Rollback

```sh
sudo deployments/update.sh ./dist/ty2shorten-linux-amd64
```

The script backs up the binary and SQLite database, installs the new binary, restarts the service, checks health, and rolls back the binary when health fails.

## Production Validation Still Needed

The deployment files have been reviewed and documented, but this repository audit did not validate them on a real VPS.
