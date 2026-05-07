---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/extension/model_source_test.go
line: 250
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:6288f563b57d
review_hash: 6288f563b57d
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 016: Assert the specific validation failure for each row case.
## Review Comment

Every case here passes on any non-nil error, so the table will miss regressions that reject a row for the wrong reason. Add an expected sentinel or substring per case and assert with `errors.Is`/`ErrorContains`.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Notes:
  - The invalid-row table in `internal/extension/model_source_test.go:250-344` passes on any non-nil error.
  - Each case is intended to validate a distinct failure reason, so generic `err != nil` hides regressions that fail for the wrong reason.
  - Fix: add an expected error substring per case and assert the specific validation message.
