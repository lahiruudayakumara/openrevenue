# Observability

Emit structured redacted logs, Prometheus metrics, and OpenTelemetry traces with tenant-safe correlation. Monitor availability, latency, errors, saturation, outbox lag, notification failures, reconciliation exceptions, database health, and backup freshness. Alerts link to owned runbooks and actionable service-level objectives.

Monitor `openrevenue_domain_context_rejections_total` for ingress requests
rejected before application dispatch. Logs and traces may contain a correlation
ID but must not contain taxpayer identifiers, authorization headers, raw tenant
claims, or request payloads. Money mismatch and overflow failures are security-
and integrity-relevant application errors; alert on recurrence without recording
the underlying sensitive values.

The taxpayer-registration vertical slice exports low-cardinality counters for
successful taxpayer creation, registration submission, registration approval,
and safe request failures. Tenant identifiers, jurisdiction identifiers,
taxpayer identifiers, names, bearer tokens, and idempotency keys must never be
metric labels or log fields.

Operators should alert on a sustained increase in
`openrevenue_registration_vertical_slice_failures_total` relative to successful
submissions and approvals. Audit and integration events carry correlation and
causation identifiers for trace linkage without exposing taxpayer data.

Return lifecycle metrics report successful draft, validation, calculation,
submission, and amendment operations using fixed low-cardinality operation
labels. Alert on a sustained increase in
`openrevenue_return_lifecycle_failures_total`. Calculation explanations and
payload hashes may be retained with the return, but raw return lines and
validation inputs must not be emitted to logs, traces, or metric labels.
