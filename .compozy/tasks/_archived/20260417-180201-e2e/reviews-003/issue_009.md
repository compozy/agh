---
status: resolved
file: internal/session/manager_prompt.go
line: 31
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:afabaedac5d2
review_hash: afabaedac5d2
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 009: Avoid silently dropping extra metadata values.
## Review Comment

`PromptNetwork` accepts `...acp.PromptNetworkMeta`, but only `meta[0]` is ever used. If a caller passes more than one value, the extras compile cleanly and disappear. Prefer a single metadata parameter, or reject `len(meta) > 1`.

## Triage

- Decision: `valid`
- Notes:
  - `PromptNetwork` accepts a variadic `...acp.PromptNetworkMeta` but only ever uses `meta[0]`, so extra values compile and are silently discarded.
  - The current call sites pass zero or one metadata object, but the API contract should reject impossible multi-meta calls instead of quietly truncating them.
  - Implemented: `PromptNetwork` now returns an explicit validation error when more than one metadata value is supplied, while keeping the zero-or-one call pattern unchanged.
  - Regression coverage: added a focused manager test to prove the method rejects multiple metadata values before the driver is called.
  - Verification: `go test ./internal/session -run 'Test(PromptWithOptsTracksTurnSourceAndClearsAfterPrompt|PromptNetworkRejectsMultipleMetadataValues)$' -count=1`; `make verify`.
