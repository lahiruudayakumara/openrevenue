# Filing

Filing owns return drafts, validation, immutable form and rule references,
submission, and amendment chains. A draft records its selected `formVersion`
and `ruleVersion`; neither can be silently upgraded.

Validation returns deterministic field-level codes rather than localized
messages or rejected values. The initial codes are `FIELD_REQUIRED`,
`FIELD_CODE_INVALID`, `FIELD_NOT_IN_FORM`, `FIELD_DUPLICATE`,
`AMOUNT_NEGATIVE`, `FORM_VERSION_INCOMPATIBLE`, and `RETURN_NOT_DRAFT`.
Clients translate these stable codes for display.

Submission requires successful validation and calculation over the same payload
hash. It rejects a changed payload or stale form/rule version and records a
frozen payload hash. Submitted returns are immutable.

An amendment creates a new draft with a new identifier and incremented revision.
It records `originalReturnId` and `supersedesId`; the submitted original remains
unchanged and retrievable in the complete revision history.
