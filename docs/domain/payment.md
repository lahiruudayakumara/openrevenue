# Payment

Payment owns receipts, idempotency references, reconciliation, unapplied credit,
and allocations. Receipt appends a balanced cash-control to unapplied-cash
posting. It does not imply that an assessment has been paid.

Allocation moves the lesser of the requested amount, available payment credit,
and assessment outstanding value from unapplied cash to taxpayer receivables.
This supports partial payments, exact payments, overpayments, and completely
unapplied receipts.

Every payment has an optimistic concurrency version. An allocation succeeds
only when its expected version matches; the version then increments atomically.
Competing requests therefore cannot spend the same payment credit twice.
