---
status: resolved
file: internal/network/audit_test.go
line: 238
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:afa8df668a50
review_hash: afa8df668a50
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 018: Wrap this case in a t.Run("Should...") subtest to match test conventions.
## Review Comment

The test logic is good, but the new case should follow the project’s required subtest naming pattern.

As per coding guidelines "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Root cause: the new capability-audit regression was added as a top-level assertion block instead of the required `Should...` subtest structure.
- Fix plan: wrap the case in a `Should...` subtest and keep the audit assertions unchanged.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
