#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID} -ne 0 ]]; then
  echo "Run as root: sudo $0 /path/to/ty2shorten" >&2
  exit 1
fi

BINARY_SOURCE=${1:-}
APP_USER=ty2shorten
APP_GROUP=ty2shorten
APP_DIR=/opt/ty2shorten
DATA_DIR=/var/lib/ty2shorten
CONFIG_DIR=/etc/ty2shorten
SERVICE_FILE=/etc/systemd/system/ty2shorten.service
ENV_FILE=${CONFIG_DIR}/ty2shorten.env

if [[ -z "${BINARY_SOURCE}" || ! -f "${BINARY_SOURCE}" ]]; then
  echo "Usage: sudo $0 /path/to/ty2shorten" >&2
  exit 1
fi

if ! id "${APP_USER}" >/dev/null 2>&1; then
  useradd --system --home-dir "${APP_DIR}" --shell /usr/sbin/nologin "${APP_USER}"
fi

install -d -o "${APP_USER}" -g "${APP_GROUP}" -m 0750 "${APP_DIR}" "${DATA_DIR}"
install -d -o root -g "${APP_GROUP}" -m 0750 "${CONFIG_DIR}"
install -o root -g "${APP_GROUP}" -m 0750 "${BINARY_SOURCE}" "${APP_DIR}/ty2shorten"
install -o root -g root -m 0644 "$(dirname "$0")/ty2shorten.service" "${SERVICE_FILE}"

if [[ ! -f "${ENV_FILE}" ]]; then
  SESSION_SECRET=$(openssl rand -base64 48)
  cat > "${ENV_FILE}" <<EOF
APP_ENV=production
APP_HOST=127.0.0.1
APP_PORT=8722
DATABASE_PATH=/var/lib/ty2shorten/ty2shorten.db
APP_CONFIG_PATH=/var/lib/ty2shorten/config.json
MEDIA_STORAGE_PATH=/var/lib/ty2shorten/media
SESSION_SECRET=${SESSION_SECRET}
BASE_URL=https://app.phyowaiyan.com
TRUSTED_PROXIES=127.0.0.1,::1
APP_VERSION=production
EOF
  chown root:"${APP_GROUP}" "${ENV_FILE}"
  chmod 0640 "${ENV_FILE}"
fi

systemctl daemon-reload
systemctl enable ty2shorten.service
systemctl restart ty2shorten.service
systemctl --no-pager status ty2shorten.service
