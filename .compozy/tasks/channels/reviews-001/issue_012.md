---
status: resolved
file: internal/api/udsapi/channels_integration_test.go
line: 77
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:b0d7b7b1cfae
review_hash: b0d7b7b1cfae
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 012: Consider logging or handling the Close error.
## Review Comment

Per coding guidelines, errors should not be ignored with `_`. While ignoring `body.Close()` errors is common in test cleanup (where recovery isn't meaningful), you could log it for visibility:

As per coding guidelines: "Never ignore errors with _ — every error must be handled or have a written justification."

## Triage

- Decision: `valid`
- Notes:
  - `mustReadAll(...)` currently discards `body.Close()` errors with `_`, and this helper is the single cleanup path for these responses.
  - I will keep the helper behavior the same but report close failures through the test so the cleanup error is no longer silently ignored.
  - Resolution: `mustReadAll(...)` now reports close failures in [internal/api/udsapi/channels_integration_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/udsapi/channels_integration_test.go:75); verified with `make verify`.
