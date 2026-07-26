# Calculation

Calculation evaluates immutable country-pack rules and never owns filing or
assessment state. The calculator interface accepts explicit inputs and returns
an explainable result without clocks, storage, network calls, or ambient
configuration.

The fictional `fictional-flat-rate-v1` example applies 1,000 basis points using
integer minor-unit arithmetic. It rounds half away from zero to the currency's
minor unit and records the basis, rate, rounded result, input hash, rule version,
and result hash. Repeating the same input produces a byte-equivalent result.

Production country packs must introduce a new rule version for behavior changes;
an existing version is immutable. The example rule and XCR currency are
fictional and are not authoritative tax values.
