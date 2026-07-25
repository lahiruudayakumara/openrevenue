# API guidelines

External routes live under `/api/v1`; operational probes remain unversioned. Design OpenAPI first, reject unknown fields, cap request bodies, authenticate then authorize, and return RFC 9457 problems. Mutation retries require idempotency keys. Propagate correlation and trace IDs, paginate collections, use UTC ISO-8601 times, and deprecate rather than silently break fields.

Authenticated application routes require `X-Tenant-ID`,
`X-Jurisdiction-Code`, and `X-Actor-ID`; identity infrastructure derives these
from trusted claims in production rather than accepting arbitrary browser
values. `X-Actor-Type` defaults to `USER`. `X-Correlation-ID` may be supplied or
is generated at ingress. Middleware validates the complete canonical context
before calling an application service.
