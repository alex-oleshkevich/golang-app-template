#!/usr/bin/env bash
set -euo pipefail
#
# deploy.sh — Build the image and deploy it to the Uncloud cluster.

cd "$(dirname -- "$0")/.." || exit 1

: "${NISABA_DOMAIN:?set NISABA_DOMAIN}"

if ! command -v uc >/dev/null 2>&1; then
  printf '%s\n' "uc is not installed" >&2
  exit 1
fi

exec uc deploy "$@"
