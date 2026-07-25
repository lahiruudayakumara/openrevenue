# Data architecture

PostgreSQL uses one logical schema per bounded context. Tenant and country scope are explicit columns, with row-level security planned as defense in depth. Migrations are forward-only. Money is stored as `bigint` minor units plus a three-character currency. Submitted returns persist payload, form version, and rule version. Ledger and audit rows are append-only; corrections add reversals and replacements. Backups are encrypted and routinely restored in an isolated environment.

Every scoped primary/unique key and query predicate includes tenant and
jurisdiction. Currency rows also retain the applicable minor-unit definition,
and periods retain inclusive start and exclusive end dates. Adapters validate
stored primitives through canonical constructors; invalid legacy data fails
closed instead of being silently normalized.
