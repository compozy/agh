---
status: resolved
file: web/src/systems/session/components/stories/copy-button.stories.tsx
line: 6
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:1deda152ec11
review_hash: 1deda152ec11
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 026: Use alias import for local module.
## Review Comment

Line 6 should use the `@/*` path alias instead of a relative path.

As per coding guidelines, `web/src/**/*.{ts,tsx}`: "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `valid`
- Notes: The story imports `CopyButton` via a relative path from inside `web/src`, which conflicts with the active `@/*` alias policy. Fix by switching the component import to the aliased session-system path.

## Resolution

- Switched the `CopyButton` story to the aliased component import and kept the touched stories explicit with `args: {}`.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
