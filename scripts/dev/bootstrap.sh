#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

if [[ ! -f .env ]]; then
  cp .env.example .env
  printf 'Created .env from the fictional local-development template.\n'
else
  printf 'Preserving existing .env.\n'
fi

"$repo_root/scripts/dev/doctor.sh"
go mod download
corepack pnpm install --frozen-lockfile
"$repo_root/scripts/dev/validate-config.sh"

printf '\nBootstrap complete. Run make services-up, then make api.\n'
