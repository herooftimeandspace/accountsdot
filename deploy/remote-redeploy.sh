#!/usr/bin/env bash
set -euo pipefail

remote="${ACCOUNTSDOT_REMOTE:-origin}"
worktree_root="${ACCOUNTSDOT_WORKTREE_ROOT:-../accountsdot-deploy-worktrees}"
compose_file="${ACCOUNTSDOT_COMPOSE_FILE:-docker-compose.deploy.yml}"
branches="${ACCOUNTSDOT_DEPLOY_BRANCHES:-dev staging main}"

git fetch "$remote"
mkdir -p "$worktree_root"

for branch in $branches; do
	if ! git show-ref --verify --quiet "refs/remotes/$remote/$branch"; then
		printf 'Missing remote branch %s/%s. Create or push it before deploying this environment.\n' "$remote" "$branch" >&2
		exit 1
	fi

	destination="$worktree_root/$branch"
	if [ -e "$destination/.git" ]; then
		git -C "$destination" fetch "$remote"
		git -C "$destination" checkout --detach "$remote/$branch"
	else
		git worktree add --detach "$destination" "$remote/$branch"
	fi
done

docker compose -f "$compose_file" up -d --build --remove-orphans "$@"
