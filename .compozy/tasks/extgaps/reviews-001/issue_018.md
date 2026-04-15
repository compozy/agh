---
status: resolved
file: internal/store/globaldb/global_db_bridges_test.go
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110509614,nitpick_hash:e01b9d37a776
review_hash: e01b9d37a776
source_review_id: "4110509614"
source_review_submitted_at: "2026-04-15T03:04:56Z"
---

# Issue 018: Consider asserting source default semantics, not only column presence.
## Review Comment

This test currently verifies existence, but not the expected default (`dynamic`) and NOT NULL behavior that migrations depend on.

## Triage

- Decision: `valid`
- Root cause: the migration/schema test only proves that the `source` column exists. It does not verify the critical migration contract that legacy inserts get the `dynamic` default and that the column is effectively required.
- Fix plan: extend the test to insert a bridge row without `source`, then assert the persisted value is `dynamic` and non-empty after readback.
- Resolution: extended the bridge schema test to insert a row without `source` and assert the persisted default is `dynamic`.
- Verification: updated `internal/store/globaldb/global_db_bridges_test.go` and passed `go test ./internal/store/globaldb` plus `make verify`.
