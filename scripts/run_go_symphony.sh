#!/usr/bin/env sh
set -u

if [ "$#" -lt 1 ]; then
  echo "usage: scripts/run_go_symphony.sh <symphony-command> [args...]" >&2
  exit 2
fi

subcommand="$1"
shift
repo_root="$(pwd)"
repo_go_cache="$repo_root/.gocache"
repo_go_mod_cache="$repo_root/.gomodcache"

# Symphony commands must not inherit host-global Go caches. The daemon tears down
# and repairs workspaces independently, so every Go-backed command uses caches
# scoped to this checked-out repository regardless of the caller's environment.
export GOCACHE="$repo_go_cache"
export GOMODCACHE="$repo_go_mod_cache"
export GOFLAGS="${GOFLAGS:+$GOFLAGS }-modcacherw"

cleanup_module_cache_permissions() {
  if [ -n "${GOMODCACHE:-}" ] && [ -d "$GOMODCACHE" ]; then
    chmod -R u+w "$GOMODCACHE" 2>/dev/null || true
  fi
}

trap cleanup_module_cache_permissions EXIT

go run ./cmd/symphony "$subcommand" "$@"
