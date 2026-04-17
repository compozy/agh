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

# Issue 008: Strengthen environment assertions to cover all mapped fields.
## Review Comment

You set `Profile`, `State`, and `InstanceID` in the fixture but don’t assert them. Adding checks here would catch partial mapping regressions.

## Triage

- Decision: `VALID`
- Root cause: `TestSessionPayloadFromInfo` exercises multiple mapped session fields but only asserts a subset of them. The current payload includes `State` and `ACPSessionID`, and leaving those unchecked weakens the mapping regression coverage.
- Fix approach: Expand the assertions to cover the currently mapped fields that are present in the fixture and not yet verified.

## Resolution

- Extended [internal/api/core/conversions_parsers_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/core/conversions_parsers_test.go) to assert `State` and `ACPSessionID` in the mapped session payload.
