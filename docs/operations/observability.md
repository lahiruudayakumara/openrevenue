# Observability

Emit structured redacted logs, Prometheus metrics, and OpenTelemetry traces with tenant-safe correlation. Monitor availability, latency, errors, saturation, outbox lag, notification failures, reconciliation exceptions, database health, and backup freshness. Alerts link to owned runbooks and actionable service-level objectives.

Monitor `openrevenue_domain_context_rejections_total` for ingress requests
rejected before application dispatch. Logs and traces may contain a correlation
ID but must not contain taxpayer identifiers, authorization headers, raw tenant
claims, or request payloads. Money mismatch and overflow failures are security-
and integrity-relevant application errors; alert on recurrence without recording
the underlying sensitive values.
