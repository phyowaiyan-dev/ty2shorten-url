# AWS Ubuntu VPS Setup With Apache

This guide explains how to deploy Ty2Shorten URL on an AWS Ubuntu VPS with Apache as the public reverse proxy. It covers both SQLite and MySQL database setups.

The examples use:

- Domain: `app.example.com`
- App user: `ty2shorten`
- App binary: `/opt/ty2shorten/ty2shorten`
- App data: `/var/lib/ty2shorten`
- App env file: `/etc/ty2shorten/ty2shorten.env`
- Local app listener: `127.0.0.1:8722`
- Public proxy: Apache on ports `80` and `443`

Replace `app.example.com` with the real domain before running production commands.

## Minimal System Requirements

Recommended minimum for a small production deployment:

| Component | SQLite setup | MySQL setup |
| --- | --- | --- |
| Instance size | 1 vCPU, 1 GB RAM | 1 vCPU, 2 GB RAM |
| Disk | 10 GB SSD | 20 GB SSD |
| OS | Ubuntu 22.04 LTS or 24.04 LTS | Ubuntu 22.04 LTS or 24.04 LTS |
| Runtime | Single Go binary | Single Go binary plus MySQL |
| Proxy | Apache 2.4 | Apache 2.4 |
| TLS | Certbot | Certbot |

Practical AWS starting points:

- SQLite: `t3.micro`, `t3.small`, or similar.
- MySQL on the same VPS: `t3.small` or better.
- MySQL on Amazon RDS: app VPS can stay smaller, but database cost is separate.

Use more CPU, RAM, and disk when traffic, analytics retention, media uploads, or short-link volume grows.

## AWS Preparation

1. Create an EC2 instance.
2. Choose Ubuntu 22.04 LTS or Ubuntu 24.04 LTS.
3. Assign an Elastic IP if this is a long-running production service.
4. Configure the security group:

| Type | Port | Source |
| --- | --- | --- |
| SSH | `22` | Your IP only |
| HTTP | `80` | `0.0.0.0/0`, `::/0` |
| HTTPS | `443` | `0.0.0.0/0`, `::/0` |

Do not expose the app port `8722` publicly. Apache should be the only public entry point.

5. Point DNS to the EC2 public IP:

```text
app.example.com -> A record -> your Elastic IP
```

## Server Base Setup

SSH into the server:

```sh
ssh ubuntu@app.example.com
```

Update packages:

```sh
sudo apt update
sudo apt upgrade -y
```

Install required packages:

```sh
sudo apt install -y \
  apache2 \
  certbot \
  python3-certbot-apache \
  curl \
  git \
  build-essential \
  sqlite3 \
  ca-certificates \
  openssl
```

Enable Apache:

```sh
sudo systemctl enable --now apache2
sudo systemctl status apache2 --no-pager
```

Enable Apache modules:

```sh
sudo a2enmod proxy proxy_http headers ssl rewrite
sudo systemctl reload apache2
```

## Build The Binary

You can build on the VPS or build on another Linux AMD64 machine and copy the binary to the server.

### Option A: Build On The VPS

Install Go 1.26.1 or a newer compatible version. Then:

```sh
git clone https://github.com/phyowaiyan-dev/ty2shorten-url.git
cd ty2shorten-url
go mod download
```

Build:

```sh
mkdir -p dist
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -trimpath \
  -ldflags "-s -w -X main.version=production -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dist/ty2shorten-linux-amd64 ./cmd/server
```

### Option B: Copy A Prebuilt Binary

From your local machine:

```sh
scp dist/ty2shorten-linux-amd64 ubuntu@app.example.com:/tmp/ty2shorten
```

On the server, place it somewhere readable:

```sh
chmod +x /tmp/ty2shorten
```

## Install The App

From inside the repository on the VPS:

```sh
sudo deployments/install.sh ./dist/ty2shorten-linux-amd64
```

If you copied a prebuilt binary:

```sh
sudo ./deployments/install.sh /tmp/ty2shorten
```

The installer creates:

- System user: `ty2shorten`
- App directory: `/opt/ty2shorten`
- Data directory: `/var/lib/ty2shorten`
- Environment directory: `/etc/ty2shorten`
- systemd unit: `ty2shorten.service`

Edit the environment file:

```sh
sudo nano /etc/ty2shorten/ty2shorten.env
```

Use this production shape:

```env
APP_ENV=production
APP_HOST=127.0.0.1
APP_PORT=8722
DATABASE_PATH=/var/lib/ty2shorten/ty2shorten.db
APP_CONFIG_PATH=/var/lib/ty2shorten/config.json
MEDIA_STORAGE_PATH=/var/lib/ty2shorten/media
SESSION_SECRET=replace-with-a-long-random-secret
BASE_URL=https://app.example.com
TRUSTED_PROXIES=127.0.0.1,::1
APP_VERSION=production
```

Generate a new secret when needed:

```sh
openssl rand -base64 48
```

Create the media directory:

```sh
sudo install -d -o ty2shorten -g ty2shorten -m 0750 /var/lib/ty2shorten/media
```

Restart:

```sh
sudo systemctl daemon-reload
sudo systemctl restart ty2shorten
sudo systemctl status ty2shorten --no-pager
```

Check local health:

```sh
curl -f http://127.0.0.1:8722/health
```

## Apache Reverse Proxy

Create the Apache site:

```sh
sudo nano /etc/apache2/sites-available/ty2shorten.conf
```

Paste this config:

```apache
<VirtualHost *:80>
    ServerName app.example.com

    ProxyPreserveHost On
    ProxyRequests Off

    RequestHeader set X-Forwarded-Proto "http"
    RequestHeader set X-Forwarded-Host "app.example.com"
    RequestHeader set X-Forwarded-Port "80"

    ProxyPass / http://127.0.0.1:8722/
    ProxyPassReverse / http://127.0.0.1:8722/

    ErrorLog ${APACHE_LOG_DIR}/ty2shorten-error.log
    CustomLog ${APACHE_LOG_DIR}/ty2shorten-access.log combined
</VirtualHost>
```

Enable the site:

```sh
sudo a2dissite 000-default.conf
sudo a2ensite ty2shorten.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
```

Check the public HTTP endpoint:

```sh
curl -I http://app.example.com/health
```

## TLS With Certbot

Issue the certificate:

```sh
sudo certbot --apache -d app.example.com
```

After Certbot updates Apache, verify that the HTTPS virtual host forwards the correct headers:

```apache
RequestHeader set X-Forwarded-Proto "https"
RequestHeader set X-Forwarded-Host "app.example.com"
RequestHeader set X-Forwarded-Port "443"
```

Reload Apache:

```sh
sudo apache2ctl configtest
sudo systemctl reload apache2
```

Verify:

```sh
curl -f https://app.example.com/health
```

## SQLite Setup

SQLite is the simplest setup and is recommended for small deployments.

### Environment

Keep these values:

```env
DATABASE_PATH=/var/lib/ty2shorten/ty2shorten.db
APP_CONFIG_PATH=/var/lib/ty2shorten/config.json
```

Restart the app:

```sh
sudo systemctl restart ty2shorten
```

### First Browser Setup

Open:

```text
https://app.example.com/setup/database
```

Choose:

```text
Database driver: SQLite
SQLite file path: /var/lib/ty2shorten/ty2shorten.db
```

Click test connection, save the database configuration, then restart:

```sh
sudo systemctl restart ty2shorten
```

Open:

```text
https://app.example.com/setup
```

Create the first administrator and complete site settings.

### SQLite Backups

Back up the database:

```sh
sudo install -d -m 0750 /opt/ty2shorten/backups
sudo sqlite3 /var/lib/ty2shorten/ty2shorten.db ".backup '/opt/ty2shorten/backups/ty2shorten-$(date -u +%Y%m%d%H%M%S).db'"
```

Back up uploads and config:

```sh
sudo tar -czf /opt/ty2shorten/backups/ty2shorten-files-$(date -u +%Y%m%d%H%M%S).tar.gz \
  /var/lib/ty2shorten/config.json \
  /var/lib/ty2shorten/media
```

## MySQL Setup

Use MySQL when you already operate MySQL, expect more writes, or want database tooling outside the VPS filesystem.

### Install MySQL On The Same VPS

```sh
sudo apt install -y mysql-server
sudo systemctl enable --now mysql
sudo mysql_secure_installation
```

Create a database and user:

```sh
sudo mysql
```

Inside MySQL:

```sql
CREATE DATABASE ty2shorten CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'ty2shorten'@'localhost' IDENTIFIED BY 'replace-with-strong-password';
GRANT ALL PRIVILEGES ON ty2shorten.* TO 'ty2shorten'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

### MySQL With Amazon RDS

If using RDS:

- Create a MySQL-compatible RDS instance.
- Put EC2 and RDS in the same VPC.
- Allow inbound MySQL `3306` from the EC2 security group only.
- Do not expose RDS to the public internet unless you have a specific reason and firewall rules.

Use the RDS endpoint as the MySQL host during setup.

### First Browser Setup For MySQL

Open:

```text
https://app.example.com/setup/database
```

Choose:

```text
Database driver: MySQL
MySQL host: 127.0.0.1
MySQL port: 3306
MySQL database: ty2shorten
MySQL username: ty2shorten
MySQL password: replace-with-strong-password
MySQL TLS mode: leave empty for local MySQL, or use preferred/required for remote MySQL when configured
```

For RDS, use the RDS endpoint as host:

```text
MySQL host: your-rds-endpoint.region.rds.amazonaws.com
```

Click test connection, save the database configuration, then restart:

```sh
sudo systemctl restart ty2shorten
```

Open:

```text
https://app.example.com/setup
```

Create the first administrator and complete site settings.

### MySQL Backups

For local MySQL:

```sh
sudo install -d -m 0750 /opt/ty2shorten/backups
mysqldump -u ty2shorten -p ty2shorten | gzip > /opt/ty2shorten/backups/ty2shorten-$(date -u +%Y%m%d%H%M%S).sql.gz
```

For RDS, use automated RDS backups and snapshots.

Also back up:

```sh
sudo tar -czf /opt/ty2shorten/backups/ty2shorten-files-$(date -u +%Y%m%d%H%M%S).tar.gz \
  /var/lib/ty2shorten/config.json \
  /var/lib/ty2shorten/media
```

## Updating The App

Build or upload a new binary, then run:

```sh
sudo deployments/update.sh ./dist/ty2shorten-linux-amd64
```

If your health URL uses the default port:

```sh
sudo HEALTH_URL=http://127.0.0.1:8722/health deployments/update.sh ./dist/ty2shorten-linux-amd64
```

The update script backs up the current binary, backs up SQLite when `DATABASE_PATH` points to a local SQLite file, installs the new binary, restarts the service, and checks health.

For MySQL, handle database backups separately before updating.

## Useful Operations

Check service:

```sh
sudo systemctl status ty2shorten --no-pager
```

Restart service:

```sh
sudo systemctl restart ty2shorten
```

View app logs:

```sh
sudo journalctl -u ty2shorten -f
```

View Apache logs:

```sh
sudo tail -f /var/log/apache2/ty2shorten-access.log
sudo tail -f /var/log/apache2/ty2shorten-error.log
```

Check listening ports:

```sh
sudo ss -tulpn | grep -E ':(80|443|8722|3306)'
```

Validate Apache:

```sh
sudo apache2ctl configtest
```

## Troubleshooting

### Apache shows 502 proxy error

Check that the app is running:

```sh
sudo systemctl status ty2shorten --no-pager
curl -f http://127.0.0.1:8722/health
```

Confirm Apache points to the same port as `APP_PORT`.

### App fails in production with SESSION_SECRET error

Set a long secret:

```sh
sudo sed -i "s|^SESSION_SECRET=.*|SESSION_SECRET=$(openssl rand -base64 48)|" /etc/ty2shorten/ty2shorten.env
sudo systemctl restart ty2shorten
```

### SQLite permission error

Check ownership:

```sh
sudo chown -R ty2shorten:ty2shorten /var/lib/ty2shorten
sudo chmod 0750 /var/lib/ty2shorten
sudo systemctl restart ty2shorten
```

### MySQL connection fails

Check MySQL is listening:

```sh
sudo systemctl status mysql --no-pager
mysql -u ty2shorten -p -h 127.0.0.1 ty2shorten
```

For RDS, check:

- EC2 security group can reach RDS security group on `3306`.
- RDS endpoint is correct.
- Username, password, and database name match the setup form.
- TLS mode matches your MySQL/RDS configuration.

### Redirects use the wrong host or scheme

Check:

```env
BASE_URL=https://app.example.com
TRUSTED_PROXIES=127.0.0.1,::1
```

Then restart:

```sh
sudo systemctl restart ty2shorten
```

Also verify Apache forwards `X-Forwarded-Proto`, `X-Forwarded-Host`, and `X-Forwarded-Port`.

## Production Checklist

- DNS points to the VPS.
- Security group exposes only SSH, HTTP, and HTTPS.
- SSH is restricted to trusted IPs.
- Apache reverse proxy is enabled.
- TLS certificate is installed and auto-renewing.
- `APP_ENV=production`.
- `SESSION_SECRET` is long and private.
- `BASE_URL` is the final public HTTPS URL.
- `TRUSTED_PROXIES=127.0.0.1,::1`.
- App health works locally and publicly.
- Database setup is completed.
- First admin setup is completed.
- Backups are configured.
- Media directory is writable by `ty2shorten`.
- Apache and app logs are monitored.
