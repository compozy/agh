---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/cli/json_flags_test.go
line: 32
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:8c0c61c77da4
review_hash: 8c0c61c77da4
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 035: Strengthen invalid JSON assertion to validate the actual error path.
## Review Comment

This subtest currently accepts any non-nil error. Add a specific assertion so it fails if the code returns the wrong error class.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

## Triage

- Decision: `valid`
- Root cause: the invalid JSON subtest only checks for a non-nil error, so any failure path would pass without proving that `parseRequiredJSONRawMessage` returned the expected JSON parsing error class.
- Fix plan: assert the concrete decode error path, likely via `errors.As(..., *json.SyntaxError)` or the exact EOF-style parse error returned by `encoding/json`.
- Resolution: implemented and verified with focused Go tests, race-enabled package tests, and full `rtk make verify`.
