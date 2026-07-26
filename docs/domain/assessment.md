# Assessment

Assessment owns liabilities produced from submitted returns or authorized
officer decisions. One return can create at most one assessment. Retrying
submission returns the existing assessment and posting.

Creation appends a balanced ledger posting: a debit to taxpayer receivables and
an equal credit to the revenue control account. The assessment stores its
original amount, outstanding amount, currency, source return, and posting
identifier. Allocation can reduce outstanding value but cannot make it
negative.
