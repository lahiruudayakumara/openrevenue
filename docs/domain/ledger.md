# Ledger

Ledger is the sole owner of taxpayer financial entries. Every posting contains
at least two single-currency lines whose debits equal credits. PostgreSQL checks
this invariant with a deferred constraint trigger.

Entries and posting headers are append-only. Corrections append a balanced
posting that swaps every debit and credit and links each reversal line and
posting to its original. A posting can be reversed only once; corrected values
require a separate replacement posting.

Every line records tenant, jurisdiction, taxpayer, account, source reference,
actor, correlation, causation, effective time, and posting time. Receivable,
unapplied-credit, and net-due balances are projections and never mutable sources
of truth. See the [posting flow](../diagrams/ledger-posting.md).
