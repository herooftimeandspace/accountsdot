#!/usr/bin/env sh
set -u

if [ "$#" -lt 1 ]; then
  echo "usage: scripts/run_go_symphony.sh <symphony-command> [args...]" >&2
  exit 2
fi

subcommand="$1"
shift
repo_root="$(pwd)"

export GOCACHE="${GOCACHE:-"$repo_root/.gocache"}"
export GOMODCACHE="${GOMODCACHE:-"$repo_root/.gomodcache"}"
export GOFLAGS="${GOFLAGS:+$GOFLAGS }-modcacherw"

cleanup_module_cache_permissions() {
  if [ -n "${GOMODCACHE:-}" ] && [ -d "$GOMODCACHE" ]; then
    chmod -R u+w "$GOMODCACHE" 2>/dev/null || true
  fi
}

trap cleanup_module_cache_permissions EXIT

go run ./cmd/symphony "$subcommand" "$@"
