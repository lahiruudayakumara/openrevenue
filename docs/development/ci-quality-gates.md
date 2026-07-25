# CI quality gates

The repository applies the same format, lint, type-check, test, and build
commands locally and in GitHub Actions. Run the complete baseline before pushing:

```bash
make quality
bash scripts/ci/validate-workflows.sh
```

## Required pull-request checks

Configure the `main` ruleset or branch-protection rule to require the branch to
be current and require these exact job names:

- `Backend quality`
- `Frontend quality`
- `Contract validation`
- `Documentation links`
- `Dependency vulnerability and license policy`
- `CodeQL (go)`
- `CodeQL (javascript-typescript)`
- `Container build and vulnerability scan`
- `Repository vulnerabilities, secrets, and misconfigurations`
- `Reproducible environment`
- `Workflow security policy`

Also require pull requests, resolved conversations, linear history, signed
commits where the contributor tooling supports them, and at least one approving
review. For a one-person bootstrap phase, use GitHub's bypass list only for
emergency recovery and record the reason; do not weaken or remove required
checks. Enable the dependency graph, Dependabot alerts, secret scanning, push
protection, CodeQL default/configured setup as appropriate, and private
vulnerability reporting in repository security settings.

Use squash merge and automatically delete merged branches. Create each branch
from current `main`; avoid a long-lived `dev` integration branch because it
causes repeated dependency and lockfile conflicts.

## Permissions and supply-chain policy

Workflows declare read-only permissions unless a job needs a specific write
capability. All reusable actions are pinned to full commit SHAs, with the
human-readable release in a comment. Dependabot proposes weekly updates, and the
workflow-policy check rejects mutable action references.

Dependency Review fails on moderate-or-higher known vulnerabilities and rejects
AGPL-1.0 and SSPL-1.0 dependencies. Exceptions require a security review,
documented legal approval, an expiry date, and a narrowly scoped configuration
change. Trivy checks the repository and container for high or critical
vulnerabilities, secrets, and configuration errors. CodeQL covers Go and
JavaScript/TypeScript.

## Failure ownership and reruns

The author owns the first investigation. CODEOWNERS identifies the reviewing
team for contracts, database, workflow, release, and security changes:

| Check | Primary owner | First response |
| --- | --- | --- |
| Backend or frontend | Change author/module owner | Reproduce with the matching `make` target |
| Contracts | API maintainers | Run Redocly/AJV and inspect compatibility |
| Dependency or security | Security maintainers | Confirm advisory, secret, license, or false-positive evidence |
| Container, environment, workflow policy | Platform maintainers | Inspect build inputs, pins, cache, and runner logs |
| Documentation | Change author | Correct or explicitly exclude non-public/local links |
| Release | Platform and security maintainers | Stop publication and follow the release runbook |

Open the failed job, expand the first failing step, and reproduce its documented
command locally. Do not rerun deterministic failures. Rerun a job once only for
an identified transient runner, registry, or network failure. If it fails again,
open an issue with the run URL, commit SHA, failing step, redacted log excerpt,
and owner. Never paste tokens, credentials, taxpayer records, or production
configuration into logs or issues.

Caches are disposable performance aids. If corruption is suspected, rerun with
the cache removed or a changed cache key; never loosen a gate to make a run pass.
Workflow artifacts expire after the configured retention period, while release
assets and attestations follow the release retention policy.
