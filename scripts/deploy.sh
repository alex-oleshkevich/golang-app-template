#!/usr/bin/env bash
set -euo pipefail
#
# deploy.sh — Build the image and deploy it to the Uncloud cluster.

cd "$(dirname -- "$0")/.." || exit 1

: "${APP_DOMAIN:?set APP_DOMAIN}"

if ! command -v uc >/dev/null 2>&1; then
  printf '%s\n' "uc is not installed" >&2
  exit 1
fi

# compose.yml passes this to the image build, which stamps it into the binary.
APP_COMMIT="$(git rev-parse --short=7 HEAD)"
export APP_COMMIT

exec uc deploy "$@"
