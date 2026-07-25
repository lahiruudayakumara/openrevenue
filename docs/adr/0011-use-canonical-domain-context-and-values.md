# 0011: Use canonical domain context and value objects

**Status:** Accepted

## Context

Every revenue decision is made for one tenant and jurisdiction, by an actor, and
within a traceable request. Raw strings, floating-point amounts, ambient clocks,
and unscoped repository keys make cross-tenant disclosure, inconsistent country
policy, arithmetic corruption, and irreproducible decisions more likely.

## Decision

Maintain a framework-free shared kernel in `pkg/domain` for tenant,
jurisdiction, actor, correlation, taxpayer identifier, Gregorian date/period,
financial-year rule, currency, money, and clock values.

Application operations require an already validated domain `Context`. Storage
and idempotency lookups are keyed by tenant and jurisdiction. Domain aggregates
retain that scope. HTTP transport constructs context only after authentication
and rejects missing or invalid dimensions before dispatch.

Money uses signed 64-bit minor units and an explicit currency definition.
Arithmetic rejects currency/minor-unit mixing and overflow. Periods are
half-open `[start, end)` UTC calendar-date intervals. Business time comes from an
injected clock.

Taxpayer identifier syntax, currency catalogs, and non-Gregorian or non-annual
period calendars remain jurisdiction-pack extension points. Packs normalize and
validate external values before constructing canonical values; they cannot
weaken tenant or jurisdiction scope.

## Consequences

- Application APIs are intentionally context-heavy and cannot perform unscoped
  reads or writes.
- Transport and persistence adapters must reconstruct values through validated
  constructors rather than struct literals.
- Currency minor units and identifier schemes must come from versioned,
  validated jurisdiction configuration.
- Existing persistence will require forward-only tenant, jurisdiction,
  currency-minor-unit, and period-boundary columns when repositories are added.
- Tests use fixed clocks and synthetic context values.

## Alternatives considered

Ambient request context, primitive strings, decimal floating point, global
currency tables, and direct `time.Now` calls were rejected because they hide
critical invariants and make deterministic testing or jurisdiction extension
harder.
