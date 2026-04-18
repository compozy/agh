---
status: resolved
file: web/src/systems/session/mocks/fixtures.ts
line: 295
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:020b7d0ce48d
review_hash: 020b7d0ce48d
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 031: Avoid duplicate message IDs in uiMessageFixtures.
## Review Comment

Line 299 clones `bashToolMessageFixture` but keeps its original `id`, so this array contains duplicate IDs. If rendered with `key={message.id}`, it can produce unstable UI and misleading Storybook behavior.

## Triage

- Decision: `valid`
- Notes: `uiMessageFixtures` currently includes a cloned copy of `bashToolMessageFixture` as a `tool_result` message without assigning a new `id`, so the array contains duplicate IDs. That can produce unstable keyed rendering and inaccurate Storybook behavior. Fix by assigning a distinct ID to the cloned tool-result fixture and add a regression check for message-ID uniqueness.

## Resolution

- Assigned a distinct `tool_bash_result` id to the cloned tool-result fixture and added a regression test that asserts `uiMessageFixtures` ids stay unique.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-typecheck`, `make web-test`, and `make verify`.
