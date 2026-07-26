# Payment allocation

```mermaid
sequenceDiagram
  participant Bank as Bank or Gateway
  participant API as Payment API
  participant Payment as Payment Module
  participant Ledger as Ledger Module
  participant Outbox
  participant Worker
  participant Notify as Notification Module
  Bank->>API: Payment reference and amount
  API->>Payment: Record idempotently
  Payment->>Ledger: Debit cash / credit unapplied cash
  API->>Payment: Allocate with expected payment version
  Payment->>Payment: Min(requested, unapplied, outstanding)
  Payment->>Ledger: Debit unapplied cash / credit receivable
  Payment-->>API: Incremented version + remaining credit
  Ledger->>Outbox: Commit events atomically
  Outbox-->>Worker: Publish
  Worker->>Notify: Send receipt
```
