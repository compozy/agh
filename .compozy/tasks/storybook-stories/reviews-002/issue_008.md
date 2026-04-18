---
status: resolved
file: web/src/systems/session/mocks/fixtures.ts
line: 149
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133343018,nitpick_hash:6071b1dbaca6
review_hash: 6071b1dbaca6
source_review_id: "4133343018"
source_review_submitted_at: "2026-04-18T02:56:43Z"
---

# Issue 008: Make multiHunkEditToolMessageFixture represent a real before/after edit.
## Review Comment

`new_string` currently mirrors `old_string`, so this fixture does not actually exercise edit-result rendering differences.

## Triage

- Decision: `valid`
- Notes:
  - Verified in `web/src/systems/session/mocks/fixtures.ts`.
  - `multiHunkEditToolMessageFixture.toolInput.new_string` currently duplicates `old_string`, so the fixture does not model an actual multi-hunk edit result.
  - Root cause: the fixture was copied without changing the replacement hunk content.
  - Fix approach: make `new_string` represent the post-edit content for both hunks so renderers exercise before/after differences correctly.
  - Resolved by making the `old_string` and `new_string` content distinct and asserting that difference in the scoped test.
