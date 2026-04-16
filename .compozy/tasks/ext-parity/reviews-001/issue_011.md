---
status: resolved
file: internal/api/core/conversions_parsers_test.go
line: 63
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:1210a1681875
review_hash: 1210a1681875
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 011: Strengthen environment assertions to cover all mapped fields.
## Review Comment

You set `Profile`, `State`, and `InstanceID` in the fixture but don’t assert them. Adding checks here would catch partial mapping regressions.

## Triage

- Decision: `VALID`
- Notes: The fixture in `TestSessionPayloadFromInfo` sets `Profile`, `State`, and `InstanceID`, but the assertions only check a subset of the mapped environment fields. Adding explicit assertions closes a real regression gap in the conversion test.
