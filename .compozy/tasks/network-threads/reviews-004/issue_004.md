---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/api/core/network.go
line: 302
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:683851d3231a
review_hash: 683851d3231a
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 004: Redundant surface check results in unreachable code.
## Review Comment

At lines 324-327, the condition `surface == ""` is checked with an inner `if` for `threadID != "" || directID != ""`, but both branches return the same error. The inner check at line 324-325 makes the standalone return at line 327 unreachable when `threadID` or `directID` is set. The logic should be simplified.

## Triage

- Decision: `valid`
- Notes: `validateNetworkSendConversation` returns the same `"surface is required"` error from both branches when `surface == ""`. The inner `threadID/directID` check is redundant and obscures the real validation flow. Simplify the branch without changing behavior.
