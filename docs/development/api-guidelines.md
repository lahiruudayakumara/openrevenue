# API guidelines

External routes live under `/api/v1`; operational probes remain unversioned. Design OpenAPI first, reject unknown fields, cap request bodies, authenticate then authorize, and return RFC 9457 problems. Mutation retries require idempotency keys. Propagate correlation and trace IDs, paginate collections, use UTC ISO-8601 times, and deprecate rather than silently break fields.

Authenticated application routes require `X-Tenant-ID`,
`X-Jurisdiction-Code`, and `X-Actor-ID`; identity infrastructure derives these
from trusted claims in production rather than accepting arbitrary browser
values. `X-Actor-Type` defaults to `USER`. `X-Correlation-ID` may be supplied or
is generated at ingress. Middleware validates the complete canonical context
before calling an application service.

## Contract governance

`contracts/openapi/openapi.yaml` is the canonical HTTP contract. Every
implemented `/api/v1` route needs a unique camel-case `operationId`, documented
success and RFC 9457 `application/problem+json` error responses, and examples
that contain fictional data only. Run `make contracts` after changing a route.

CI compares pull-request contracts with the base commit. Removing an operation,
schema, or property, or making an existing property required, fails the
compatibility gate. Intentional breaking changes require a new versioned API
surface and a recorded migration decision; the check must not be bypassed.

The TypeScript smoke client is committed at
`clients/typescript/openapi-client.ts`. Regenerate it with `make generate`.
Contract validation fails when generated output differs, ensuring generation is
reproducible.
