---
status: resolved
file: web/src/systems/network/components/stories/network-channels-list-panel.stories.tsx
line: 19
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:04e435ec2cee
review_hash: 04e435ec2cee
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 025: Import ComponentProps from React instead of using React.ComponentProps
## Review Comment

This file uses `React.ComponentProps` on line 20 but does not import React. With the `react-jsx` transform, the React namespace is unavailable, causing TypeScript errors. Import `ComponentProps` directly instead.

## Triage

- Decision: `valid`
- Notes: Current typecheck is green, so this is not an active compiler failure in the present config, but the file still relies on the ambient `React` namespace for `ComponentProps` without an explicit type import. That is weaker than the project’s explicit type-import style and makes the story less self-contained. Fix by importing `ComponentProps` from `react` and using it directly in the helper signature.

## Resolution

- Imported `ComponentProps` from `react` and replaced the ambient `React.ComponentProps` reference in the story frame helper.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-typecheck`, `make web-test`, and `make verify`.
