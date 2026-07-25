#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

fail() {
  printf 'Workflow policy violation: %s\n' "$1" >&2
  exit 1
}

workflow_count=0
while IFS= read -r workflow; do
  workflow_count=$((workflow_count + 1))
  grep -Eq '^permissions:' "$workflow" ||
    fail "$workflow must declare top-level permissions"
  grep -Eq '^concurrency:' "$workflow" ||
    fail "$workflow must cancel or serialize duplicate runs"
  grep -Eq '^[[:space:]]+timeout-minutes:' "$workflow" ||
    fail "$workflow must set a timeout for every job"

  while IFS= read -r reference; do
    [[ "$reference" == docker://* ]] &&
      fail "$workflow uses a container action instead of a commit-pinned action: $reference"
    [[ "$reference" =~ @[0-9a-f]{40}$ ]] ||
      fail "$workflow has a mutable action reference: $reference"
  done < <(sed -nE 's/^[[:space:]]*-[[:space:]]*uses:[[:space:]]*([^[:space:]#]+).*$/\1/p' "$workflow")
done < <(find .github/workflows -type f \( -name '*.yml' -o -name '*.yaml' \) | sort)
((workflow_count > 0)) || fail "no workflows were found"

release=.github/workflows/release.yml
grep -Fq "tags: ['v*.*.*']" "$release" ||
  fail "release tags must use the version-tag pattern"
grep -Fq 'attestations: write' "$release" ||
  fail "release provenance permission is missing"
grep -Fq 'id-token: write' "$release" ||
  fail "release OIDC permission is missing"
grep -Fq 'SHA256SUMS' "$release" ||
  fail "release checksums are missing"
grep -Fq 'openrevenue-source.spdx.json' "$release" ||
  fail "release SBOM generation is missing"

dependency_review=.github/workflows/dependency-review.yml
grep -Fq 'deny-licenses:' "$dependency_review" ||
  fail "dependency-review license policy is missing"

if grep -Eq 'pull_request:' "$release"; then
  fail "pull requests must never invoke the release workflow"
fi

printf 'Workflow policy baseline is valid for %d workflows.\n' "$workflow_count"
