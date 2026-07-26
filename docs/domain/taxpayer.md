# Taxpayer

The taxpayer context owns parties, jurisdiction identifiers, individuals,
organizations, and authorized representatives. An application request must
carry canonical tenant, jurisdiction, actor, correlation, and causation
context.

Creation is idempotent. Replaying the same key and request returns the original
taxpayer; reusing a key for different input is rejected. Normalized identifiers
are unique by tenant, jurisdiction, and identifier scheme. Country packs can
replace the conservative demonstration identifier normalizer with a versioned
scheme-specific validator.

The `TaxpayerCreated` event and corresponding immutable audit event are written
with the actor and request context. Public API errors use problem details and
do not return repository, authorization-provider, or database error text.
