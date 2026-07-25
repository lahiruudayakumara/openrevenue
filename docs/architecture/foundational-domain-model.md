# Foundational domain model

The canonical shared kernel is deliberately small. It contains values that every
bounded context must interpret identically; it does not contain tax policy or
framework code.

## Invariants

| Value | Canonical invariant |
| --- | --- |
| Tenant | Lowercase, 2–63 character opaque key; never inferred from payload data |
| Jurisdiction | Uppercase country or country-subdivision code; selected before application dispatch |
| Actor | Explicit `USER`, `SERVICE`, or `SYSTEM` kind plus a safe opaque subject |
| Correlation | Safe 1–128 character identifier, accepted or generated at ingress |
| Taxpayer identifier | Tenant- and jurisdiction-scoped scheme/value, normalized by a versioned scheme |
| Date | Valid Gregorian UTC calendar date |
| Period | Half-open interval: start inclusive, end exclusive |
| Financial year | Annual rule whose start exists every year; calculated without ambient time |
| Currency | Three-letter uppercase code plus 0–6 minor units |
| Money | Signed `int64` minor units and one exact currency definition |
| Clock | Explicit system or fixed UTC time source |

Tenant and jurisdiction form the isolation boundary. Actor and correlation form
the accountability boundary. They travel together as a domain context and are
mandatory for every application command and query. In-memory keys currently use
both isolation dimensions; persistence adapters must also include both in
primary/unique keys and predicates. Row-level security is additional defense,
not a substitute for scoped queries.

Money has no floating-point constructor. Addition and subtraction require equal
currency codes and equal minor-unit definitions. All arithmetic checks signed
64-bit overflow. A bounded context may reject negative or zero values for a
specific operation (for example, ledger postings and payments require positive
amounts), while the shared value remains signed to support adjustments and
reversals.

Periods do not use local timestamps. `2026-04-01` means a calendar boundary, not
an instant in an operator's timezone. A financial-year rule deterministically
maps a date to one period and handles leap years. A country pack may supply
monthly, quarterly, 4-4-5, or other calendars through an application port, but
the resulting canonical period must still have explicit boundaries.

## Extension points

- Country packs own identifier normalizers/check digits and scheme versions.
- Country packs own the allowed currency catalog and minor-unit definitions.
- Country packs own tax-period calendars and labels.
- Identity adapters map verified OIDC/workload claims to canonical actors.
- Transport adapters may generate correlation IDs when callers omit them.
- Persistence adapters encode/decode private values through constructors and
  reject invalid stored data.

No extension may infer a tenant from a taxpayer identifier, silently switch
jurisdiction, change a currency's minor units within an operation, use an
unversioned normalization rule, or read the system clock directly in domain
behavior.

## Failure behavior and observability

Invalid ingress context returns an RFC 9457-style `400` response before
application dispatch. The API increments
`openrevenue_domain_context_rejections_total` without logging tenant,
identifier, token, or taxpayer values. Alert on sustained increases, correlate
with authentication and gateway telemetry, and investigate configuration or
abuse without copying sensitive headers into tickets.

Currency mismatch and overflow are explicit domain errors. Callers must reject
the operation atomically; they must not clamp, wrap, convert, or partially post
financial state.

## Impact review

- **Security:** mandatory scoped context and scoped idempotency/storage keys
  reduce confused-deputy and cross-tenant access risk. Authorization remains a
  separate deny-by-default decision after context construction.
- **Privacy:** identifiers are normalized but never emitted to metrics or error
  details. Synthetic values are used in tests and examples.
- **Accessibility:** no user-interface behavior changes in this foundation.
  Future validation messages must remain programmatically associated with input
  fields and must not rely only on color.
- **Migration:** there is no production persistence implementation yet, so this
  change creates no database migration. Future repositories require
  forward-only tenant, jurisdiction, period-boundary, currency, and minor-unit
  fields before production data exists.
- **API and events:** authenticated API calls now require canonical context
  headers. Event envelopes use `jurisdiction` instead of the ambiguous
  `country`; this is an intentional pre-v0.1 contract correction.
- **Operations:** invalid context is observable through a redacted counter.
  Currency mismatch and overflow fail atomically and should be investigated as
  integrity signals.
