---
status: resolved
file: web/src/routes/_app/stories/-agents.$name.stories.tsx
line: 87
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:3187700db5e5
review_hash: 3187700db5e5
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 006: Consider adding a play function to verify agent loading state.
## Review Comment

Similar to `SessionsLoading`, the `AgentLoading` story would benefit from a `play` function to verify the loading UI is rendered.

---

## Triage

- Decision: `VALID`
- Notes:
  - The `AgentLoading` route story pins `/api/agents/:name` with an infinite MSW delay but does not assert that the agent detail loading branch renders.
  - The story should fail if the loading UI regresses instead of remaining a passive visual fixture.
  - Fix by adding a play function that waits for the agent detail loading test id.
  - Resolution: added a `play` function that asserts `agent-detail-loading` renders.
  - Verification: targeted Vitest passed; `make verify` passed.
