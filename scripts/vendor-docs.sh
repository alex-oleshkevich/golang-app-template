#!/usr/bin/env bash
set -euo pipefail
#
# vendor-docs.sh — Download the Swagger UI and Scalar files pinned in
# internal/web/assets/docs/VERSIONS (name=version per line).

cd "$(dirname -- "$0")/.." || exit 1

readonly DOCS_DIR=internal/web/assets/docs
readonly CDN=https://cdn.jsdelivr.net/npm

die() {
    printf '%s\n' "$1" >&2
    exit 1
}

fetch() {
    curl -fsSL -o "$2" -- "$1"
}

# Each UI lives in DOCS_DIR/<name>/: a hand-written index.html plus the vendored files under <version>/.
# Downloads into a scratch dir and swaps it in, so a failed fetch keeps the old version.
vendor() {
    local name="$1" ver="$2" ui_dir="${DOCS_DIR}/$1" tmp old
    [[ "${ver}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "bad version for ${name}: ${ver}"
    [[ -f "${ui_dir}/index.html" ]] || die "missing ${ui_dir}/index.html"

    tmp=$(mktemp -d -p "${ui_dir}") || exit 1
    trap "rm -rf -- '${tmp}'" EXIT
    chmod 755 -- "${tmp}"

    case "${name}" in
        swagger)
            fetch "${CDN}/swagger-ui-dist@${ver}/swagger-ui-bundle.js" "${tmp}/swagger-ui-bundle.js"
            fetch "${CDN}/swagger-ui-dist@${ver}/swagger-ui.css" "${tmp}/swagger-ui.css"
            ;;
        scalar)
            fetch "${CDN}/@scalar/api-reference@${ver}/dist/browser/standalone.js" "${tmp}/standalone.js"
            ;;
        *) die "unknown entry in VERSIONS: ${name}" ;;
    esac

    for old in "${ui_dir}"/*/; do
        [[ "${old%/}" == "${tmp}" ]] || rm -rf -- "${old}"
    done
    mv -- "${tmp}" "${ui_dir}/${ver}"
    trap - EXIT

    # index.html links its files by relative "<version>/..." paths; point them at the new version.
    sed -E -i "s#((src|href)=\")[0-9]+\.[0-9]+\.[0-9]+/#\\1${ver}/#g" "${ui_dir}/index.html"
    printf 'vendored %s %s\n' "${name}" "${ver}"
}

main() {
    local name ver
    while IFS='=' read -r name ver; do
        [[ -z "${name}" || "${name}" == \#* ]] && continue
        [[ -n "${ver}" ]] || die "missing version for ${name} in VERSIONS"
        vendor "${name}" "${ver}"
    done < "${DOCS_DIR}/VERSIONS"
}

main "$@"
