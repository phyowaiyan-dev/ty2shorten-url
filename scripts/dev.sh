#!/bin/sh
set -u

air_pid=""
css_pid=""
status=0

stop_children() {
  trap - INT TERM EXIT

  for pid in "$air_pid" "$css_pid"; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
    fi
  done

  for pid in "$air_pid" "$css_pid"; do
    if [ -n "$pid" ]; then
      wait "$pid" 2>/dev/null || true
    fi
  done

  exit "$status"
}

on_interrupt() {
  status=130
  stop_children
}

on_term() {
  status=143
  stop_children
}

on_exit() {
  status=$?
  stop_children
}

command -v air >/dev/null 2>&1 || {
  echo "Air is not installed."
  echo "Install it with: make install-dev-tools"
  exit 1
}

command -v npm >/dev/null 2>&1 || {
  echo "npm is not installed or is not on PATH."
  echo "Install Node.js dependencies with: npm install"
  exit 1
}

trap on_interrupt INT
trap on_term TERM
trap on_exit EXIT

air &
air_pid=$!

npm run css:watch &
css_pid=$!

while :; do
  if ! kill -0 "$air_pid" 2>/dev/null; then
    wait "$air_pid" || status=$?
    break
  fi

  if ! kill -0 "$css_pid" 2>/dev/null; then
    wait "$css_pid" || status=$?
    break
  fi

  sleep 1
done

stop_children
