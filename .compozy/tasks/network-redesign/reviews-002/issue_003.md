---
status: resolved
file: internal/network/audit_test.go
line: 224
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167301360,nitpick_hash:f527ef25af56
review_hash: f527ef25af56
source_review_id: "4167301360"
source_review_submitted_at: "2026-04-24T01:39:58Z"
---

# Issue 003: Add a mirrored received-direct case for direction coverage.
## Review Comment

This subtest now validates only the sent direct path. Adding a received direct variant would better protect direction-specific timeline mapping.

---

## Triage

- Decision: `valid`
- Root cause: The current timeline coverage only exercises the sent direct path. Direct-message normalization is direction-sensitive, so leaving out the received variant means the `peer_from`/`peer_to` mapping for inbound direct envelopes is not protected by tests.
- Fix plan: Add a mirrored received-direct case to the same table-driven suite in `internal/network/audit_test.go` and assert the expected direct-message fields for the inbound path.
- Outcome: Added a received-direct case to the shared renderable-envelope suite and asserted inbound addressing metadata. Verified with `go test ./internal/network -count=1` and `make verify`.
