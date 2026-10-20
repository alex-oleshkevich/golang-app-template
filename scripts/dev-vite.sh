#!/usr/bin/env bash
set -euo pipefail
#
# dev-vite.sh AIR_PID — air's pre_cmd: run the Vite dev server next to air.
# Vite starts once and survives rebuilds. When either Vite or air exits, the other is stopped.

air_pid="${1:?usage: dev-vite.sh AIR_PID}"
url="${APP_FRONTEND_DEV_SERVER:?start through just dev}"

# pre_cmd runs before every rebuild; after the first one Vite is already up.
if curl -s -o /dev/null "${url}"; then
  exit 0
fi

cd "$(dirname -- "$0")/../ui"
(
  node_modules/.bin/vite &
  vite_pid=$!
  while kill -0 "${air_pid}" 2>/dev/null && kill -0 "${vite_pid}" 2>/dev/null; do
    sleep 1
  done
  kill -INT "${air_pid}" "${vite_pid}" 2>/dev/null || true
) &
