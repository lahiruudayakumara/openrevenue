# Event guidelines

Events are immutable past-tense facts inside the standard envelope. Include identity, version, time, correlation, causation, actor, tenant, country, data, and metadata. Publish with the outbox, handle at least-once delivery idempotently, avoid sensitive payloads, and make same-version evolution additive.

Envelope tenant, jurisdiction, actor, and correlation fields use the canonical
domain formats. Consumers validate them before dispatch and include tenant plus
jurisdiction in idempotency keys. Producers use the injected application clock
for `occurredAt`; event handlers never substitute receipt time for decision
time.

`contracts/events/envelope.schema.json` is the canonical JSON Schema 2020-12
envelope. Event names are past-tense PascalCase facts and `eventVersion` starts
at 1. A version may add optional data or metadata only; removals, renamed
fields, changed meanings, and new required fields require a new event version
and a documented consumer migration. Examples live in
`contracts/events/examples` and are validated in CI. They must be fictional and
must not contain taxpayer identifiers or other production data.
