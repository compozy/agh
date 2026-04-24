---
status: resolved
file: internal/api/core/interfaces.go
line: 106
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151559901,nitpick_hash:3aca0e96c0c4
review_hash: 3aca0e96c0c4
source_review_id: "4151559901"
source_review_submitted_at: "2026-04-22T01:22:21Z"
---

# Issue 006: Refresh the NetworkStore doc comment.
## Review Comment

It now owns channel metadata CRUD as well, so "audit and timeline queries" undersells the actual contract and makes the interface easier to misuse.

## Triage

- Decision: `valid`
- Reasoning: the `NetworkStore` interface now exposes persisted channel CRUD in addition to audit and timeline reads. The existing comment underspecifies the contract and can mislead callers about what the store owns.
- Fix plan: refresh the interface doc comment so it accurately describes audit, channel metadata CRUD, and timeline responsibilities.
- Resolution: refreshed the `NetworkStore` doc comment so it now describes audit access, channel metadata CRUD, and timeline responsibilities.
- Verification: `go test ./internal/api/core` and `make verify`
