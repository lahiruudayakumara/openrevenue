# Release management

OpenRevenue releases are built from reviewed commits on `main`. Pull-request
workflows only validate source and build non-published test artifacts; they have
read-only repository permissions and cannot invoke the release workflow.

## Release prerequisites

Before the first release, create a protected GitHub environment named
`production`. Require manual approval and restrict deployment branches and tags
to protected release tags. Do not add long-lived cloud credentials: provenance
uses GitHub's short-lived OIDC token.

The release operator must confirm:

1. The intended commit is on `main` and all required checks are successful.
2. Release notes describe security, privacy, migration, API, event, database,
   country-pack, and operational effects, including “none” where appropriate.
3. Database changes have forward and rollback/forward-fix instructions.
4. The version is unused and follows `vMAJOR.MINOR.PATCH`, with an optional
   SemVer pre-release or build suffix.

## Create a release

Create an annotated tag from the verified `main` commit:

```bash
git switch main
git pull --ff-only origin main
git tag -a v0.1.0 -m "OpenRevenue v0.1.0"
git push origin v0.1.0
```

The tag-only workflow verifies that the tagged commit belongs to `main`, reruns
the backend release gates, and then:

- cross-compiles the API, worker, and scheduler for Linux, macOS, and Windows;
- records the tag, commit, repository, and workflow-run URL;
- produces a source SPDX JSON SBOM;
- produces `SHA256SUMS` for every attached file;
- creates GitHub build-provenance attestations using OIDC;
- creates immutable GitHub release notes and uploads the bundle.

The `production` environment approval is the publication boundary. Reject it if
the tag, source commit, checks, or release impact is unclear.

## Verify downloaded artifacts

```bash
sha256sum --check SHA256SUMS
gh attestation verify openrevenue-api-linux-amd64 \
  --repo opencorex-org/openrevenue
```

Use `shasum -a 256 -c SHA256SUMS` on macOS. Retain the workflow URL and
`BUILD-METADATA.txt` with operational change records.

## Failure and recovery

Never move or overwrite a published tag. If validation or build fails, correct
the source on a pull request, merge it, and create a new patch version. A failed
job may be rerun only when the failure is demonstrably transient and the commit,
action revisions, and inputs are unchanged. Never rerun publication after any
release asset has been partially published; inspect and remove a draft release,
then use a new version if consumers could have observed it.

Production rollback means promoting a previously verified immutable artifact
when database and event compatibility allow it. Otherwise perform a forward fix.
Validate health probes, metrics, migrations, event consumers, audit logging, and
critical taxpayer journeys after promotion. Deployment itself is intentionally
outside this foundation workflow.
