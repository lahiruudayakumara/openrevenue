#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
errors=0

pass() { printf 'ok    %s\n' "$1"; }
fail() {
  printf 'error %s\n' "$1" >&2
  errors=$((errors + 1))
}

require_command() {
  if command -v "$1" >/dev/null 2>&1; then
    pass "$1 is installed"
  else
    fail "$1 is required. $2"
  fi
}

require_command go "Install Go 1.26.5: https://go.dev/dl/"
require_command node "Install Node 22.23.1 using mise, asdf, fnm, or nvm."
require_command corepack "Install a Node distribution that includes Corepack."
require_command docker "Install Docker Engine/Desktop with Compose v2."

version_at_least() {
  local actual_major actual_minor minimum_major minimum_minor
  IFS=. read -r actual_major actual_minor _ <<< "${1#v}"
  IFS=. read -r minimum_major minimum_minor _ <<< "${2#v}"
  ((actual_major > minimum_major)) ||
    ((actual_major == minimum_major && actual_minor >= minimum_minor))
}

if command -v go >/dev/null 2>&1; then
  expected_go="$(awk '/^toolchain go/ { sub(/^toolchain go/, ""); print; exit }' "$repo_root/go.mod")"
  actual_go="$(cd "$repo_root" && go env GOVERSION 2>/dev/null || true)"
  if [[ "$actual_go" == "go$expected_go" ]]; then
    pass "Go toolchain is $expected_go"
  else
    fail "Go $expected_go is required; found ${actual_go:-unknown}. Run 'go env -w GOTOOLCHAIN=auto' or install the pinned toolchain."
  fi
fi

if command -v node >/dev/null 2>&1; then
  expected_node="$(tr -d '[:space:]' < "$repo_root/.node-version")"
  actual_node="$(node --version | sed 's/^v//')"
  if [[ "$actual_node" == "$expected_node" ]]; then
    pass "Node is $expected_node"
  else
    fail "Node $expected_node is required; found $actual_node. Run 'nvm use', 'fnm use', or 'mise install'."
  fi
fi

if command -v corepack >/dev/null 2>&1; then
  expected_pnpm="$(
    grep -Eo '"packageManager"[[:space:]]*:[[:space:]]*"pnpm@[^"]+"' "$repo_root/package.json" |
      cut -d@ -f2 |
      tr -d '"'
  )"
  if command -v pnpm >/dev/null 2>&1; then
    actual_pnpm="$(pnpm --version 2>/dev/null || true)"
  else
    actual_pnpm="$(cd "$repo_root" && corepack pnpm --version 2>/dev/null || true)"
  fi
  if [[ "$actual_pnpm" == "$expected_pnpm" ]]; then
    pass "pnpm is $expected_pnpm"
  else
    fail "pnpm $expected_pnpm is required; found ${actual_pnpm:-unavailable}. Run 'corepack enable' and retry."
  fi
fi

if command -v docker >/dev/null 2>&1; then
  if docker info >/dev/null 2>&1; then
    pass "Docker daemon is running"
    docker_version="$(docker version --format '{{.Server.Version}}')"
    if version_at_least "$docker_version" "27.0"; then
      pass "Docker Engine is $docker_version"
    else
      fail "Docker Engine 27.0+ is required; found $docker_version. Upgrade Docker Engine/Desktop."
    fi
  else
    fail "Docker is installed but its daemon is unavailable. Start Docker Desktop or the Docker service."
  fi

  if docker compose version >/dev/null 2>&1; then
    compose_version="$(docker compose version --short)"
    if version_at_least "$compose_version" "2.30"; then
      pass "Docker Compose is $compose_version"
    else
      fail "Docker Compose 2.30+ is required; found $compose_version. Upgrade the Compose plugin."
    fi
  else
    fail "Docker Compose v2 is required. Install the Docker Compose plugin."
  fi
fi

if [[ -f "$repo_root/.env" ]]; then
  pass ".env exists"
else
  fail ".env is missing. Run 'make bootstrap' to create it from .env.example."
fi

if (( errors > 0 )); then
  printf '\nDeveloper environment has %d problem(s).\n' "$errors" >&2
  exit 1
fi

printf '\nDeveloper environment is ready.\n'
