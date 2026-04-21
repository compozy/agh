---
status: resolved
file: internal/session/manager_integration_test.go
line: 198
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:3d21f008ef30
review_hash: 3d21f008ef30
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 026: Finish the blocked fake prompt with a terminal done event.
## Review Comment

This stub closes the ACP stream for `turn-1` without ever emitting `acp.EventTypeDone`. That means the test is proving that EOF unblocks the queued synthetic turn, not that the real turn-completion path does. Sending a final `done` event here makes the ordering assertion match the production contract more closely.

Based on learnings: Check dependent package APIs before writing integration code or tests.

## Triage

- Decision: `valid`
- Root cause: the fake ACP stream for the blocked first prompt currently closes without emitting `acp.EventTypeDone`, so the test proves queue draining on EOF rather than on the real terminal-turn contract that production code relies on.
- Fix approach: update the fake prompt hook to emit a final `done` event before closing the stream, then keep the existing ordering assertions.
