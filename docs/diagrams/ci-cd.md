# CI/CD

```mermaid
flowchart LR
  PR["Pull request"] --> Q["Required quality checks"]
  Q --> F["Format, lint, types, tests"]
  Q --> C["Contracts and documentation"]
  Q --> S["CodeQL, dependency, secret, license, container scans"]
  F --> M["Reviewed squash merge to main"]
  C --> M
  S --> M
  M --> T["Protected SemVer tag"]
  T --> V["Revalidate tagged main commit"]
  V --> B["Cross-platform binaries"]
  B --> I["SBOM, checksums, build metadata"]
  I --> A["OIDC provenance"]
  A --> E["Production environment approval"]
  E --> R["Immutable GitHub release"]
```

Pull requests cannot enter the publication path. The production environment is
the explicit approval boundary, and release artifacts remain traceable to their
tag, source commit, workflow run, checksum, SBOM, and provenance attestation.
