---
status: resolved
file: packages/ui/src/components/chat-message-bubble.test.tsx
line: 130
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:34fc96c8c655
review_hash: 34fc96c8c655
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 007: Add align-override cases for non-user roles (or explicitly lock the API down)
## Review Comment

Line 130 currently validates `align` override only for `role="user"`. Since `align` is part of the public props, add coverage for `agent`/`tool`/`diff` (and clarify `system` behavior) so API intent is explicit.

## Triage

- Decision: `valid`
- Reasoning: `align` is a public prop on `ChatMessageBubble`, but the tests only exercise an override for `role="user"`. That leaves non-user roles unprotected despite exposing the same API.
- Root cause: Coverage stops at the default user branch and does not verify the public override contract for other roles.
- Fix plan: Add explicit align-override coverage for the non-user roles affected by the component logic, including the supported system behavior.

## Resolution

- Added non-user align-override coverage in `packages/ui/src/components/chat-message-bubble.test.tsx` for `agent`, `tool`, `diff`, and explicit system centering behavior.
- Verified with `make verify` after all batch changes.
