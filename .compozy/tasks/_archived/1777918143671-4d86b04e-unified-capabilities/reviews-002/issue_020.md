---
status: resolved
file: internal/network/envelope.go
line: 275
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:745a1f42471d
review_hash: 745a1f42471d
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 020: Inconsistent omitempty on Digest field.
## Review Comment

`Digest` at line 281 lacks `omitempty`, but the corresponding `NetworkCapabilityPayload` in `contract.go` (line 330) uses `omitempty` for the same field. This inconsistency may cause unexpected behavior when capabilities are serialized without a digest.

## Triage

- Decision: `invalid`
- Root cause analysis: the compared types serve different contracts. `internal/network/envelope.go` defines the transferable capability artifact on the wire, and `internal/network/validate.go` requires `capability.digest` there. `internal/api/contract/contract.go` defines the API-facing rich capability catalog payload, where `digest` is optional metadata.
- Why not fix here: adding `omitempty` to the envelope payload would contradict the wire validation rules and RFC-backed requirement that transfer envelopes always carry a digest.
- Resolution: analysis completed; the current wire/API distinction is intentional and verified.
