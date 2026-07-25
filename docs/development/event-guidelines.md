# Event guidelines

Events are immutable past-tense facts inside the standard envelope. Include identity, version, time, correlation, causation, actor, tenant, country, data, and metadata. Publish with the outbox, handle at least-once delivery idempotently, avoid sensitive payloads, and make same-version evolution additive.

Envelope tenant, jurisdiction, actor, and correlation fields use the canonical
domain formats. Consumers validate them before dispatch and include tenant plus
jurisdiction in idempotency keys. Producers use the injected application clock
for `occurredAt`; event handlers never substitute receipt time for decision
time.
