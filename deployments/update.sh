#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID} -ne 0 ]]; then
  echo "Run as root: sudo $0 /path/to/new/ty2shorten" >&2
  exit 1
fi

NEW_BINARY=${1:-}
APP_USER=ty2shorten
APP_GROUP=ty2shorten
APP_DIR=/opt/ty2shorten
DATA_DIR=/var/lib/ty2shorten
BINARY=${APP_DIR}/ty2shorten
BACKUP_DIR=${APP_DIR}/backups
DB_PATH=${DATABASE_PATH:-${DATA_DIR}/ty2shorten.db}
HEALTH_URL=${HEALTH_URL:-http://127.0.0.1:8080/health}
STAMP=$(date -u +%Y%m%d%H%M%S)

if [[ -z "${NEW_BINARY}" || ! -f "${NEW_BINARY}" ]]; then
  echo "Usage: sudo $0 /path/to/new/ty2shorten" >&2
  exit 1
fi

install -d -o root -g "${APP_GROUP}" -m 0750 "${BACKUP_DIR}"

if [[ -f "${BINARY}" ]]; then
  cp -a "${BINARY}" "${BACKUP_DIR}/ty2shorten.${STAMP}"
fi

if [[ -f "${DB_PATH}" ]]; then
  sqlite3 "${DB_PATH}" ".backup '${BACKUP_DIR}/ty2shorten.${STAMP}.db'"
  chown root:"${APP_GROUP}" "${BACKUP_DIR}/ty2shorten.${STAMP}.db"
  chmod 0640 "${BACKUP_DIR}/ty2shorten.${STAMP}.db"
fi

TMP_BINARY="${APP_DIR}/.ty2shorten.${STAMP}.new"
install -o root -g "${APP_GROUP}" -m 0750 "${NEW_BINARY}" "${TMP_BINARY}"
mv "${TMP_BINARY}" "${BINARY}"

systemctl restart ty2shorten.service
sleep 2

if curl --fail --silent --show-error "${HEALTH_URL}" >/dev/null; then
  echo "Update completed and health check passed."
  exit 0
fi

echo "Health check failed; rolling back binary." >&2
if [[ -f "${BACKUP_DIR}/ty2shorten.${STAMP}" ]]; then
  cp -a "${BACKUP_DIR}/ty2shorten.${STAMP}" "${BINARY}"
  systemctl restart ty2shorten.service
fi
exit 1
