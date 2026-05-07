---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/api/contract/contract_test.go
line: 107
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:85136c755c50
review_hash: 85136c755c50
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 004: Assert the serialized config option values too.
## Review Comment

The new JSON-shape check only verifies `id`, `kind`, and `current`. If `config_options[0].values` is dropped or mis-encoded, this test still passes even though the selector becomes unusable to clients.

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/contract/contract_test.go` asserts the serialized config option `id`, `kind`, and `current`, but not the `values` payload.
  - A regression dropping or mangling `config_options[*].values` would currently pass this test.
  - Fix plan: assert the serialized option values explicitly so the JSON shape covers selector usability end-to-end.
  - Fixed in `internal/api/contract/contract_test.go` and verified with focused package tests plus `make verify`.
