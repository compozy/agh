---
status: resolved
file: web/src/routes/_app/stories/-agents.$name.stories.tsx
line: 68
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:706081bfac4b
review_hash: 706081bfac4b
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 005: Consider adding a play function to verify loading state.
## Review Comment

The `SessionsLoading` story lacks a `play` function assertion to verify the loading UI renders. While the infinite delay ensures the loading state persists, adding an assertion would make the test more explicit.

## Triage

- Decision: `VALID`
- Notes:
  - The `SessionsLoading` route story pins `/api/sessions` with an infinite MSW delay but does not assert that the loading branch actually renders.
  - Without a play assertion, Storybook test coverage could pass even if the story stopped exercising the intended state.
  - Fix by adding a play function that waits for the sessions loading test id.
  - Resolution: added a `play` function that asserts `agent-sessions-loading` renders.
  - Verification: targeted Vitest passed; `make verify` passed.
