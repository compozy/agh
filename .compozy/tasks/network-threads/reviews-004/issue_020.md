---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/hooks/dispatch_events_test.go
line: 64
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:842d2cfd7cd6
review_hash: 842d2cfd7cd6
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 020: Assert the exact turn ID here.
## Review Comment

These subtests only prove that `TurnIDFromPayload()` returns *some* trimmed non-empty string. If the implementation starts pulling the wrong field, this still passes. Comparing against the exact trimmed turn ID would make the test regression-sensitive.

As per coding guidelines, "MUST test meaningful business logic, not trivial operations" and "Check tests can fail when business logic changes."

## Triage

- Decision: `valid`
- Notes: `TestTurnIDFromPayloadTrimsSupportedPayloads` currently proves only that the returned value is non-empty and trimmed. It does not fail if `TurnIDFromPayload` starts reading the wrong field. Assert the exact trimmed turn id for each payload row.
