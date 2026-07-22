# Apache Reverse Proxy

The provided config is `deployments/apache.conf`.

## Required Modules

```sh
sudo a2enmod proxy proxy_http headers ssl rewrite
```

## Install Config

```sh
sudo cp deployments/apache.conf /etc/apache2/sites-available/ty2shorten.conf
sudo a2ensite ty2shorten.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
```

## Proxy Target

```text
http://127.0.0.1:8080
```

## Required Headers

- `Host`
- `X-Forwarded-For`
- `X-Forwarded-Host`
- `X-Forwarded-Proto`
- `X-Forwarded-Port`

When Apache runs on the same host, use:

```env
TRUSTED_PROXIES=127.0.0.1,::1
```

Do not trust all proxies.
