---
status: resolved
file: web/src/systems/network/components/stories/network-channels-list-panel.stories.tsx
line: 6
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:10aaec52d1a5
review_hash: 10aaec52d1a5
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 024: Use @/* alias for component import
## Review Comment

Line 6 should import via alias rather than `../network-channels-list-panel`.

As per coding guidelines "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `valid`
- Notes: The story imports `NetworkChannelsListPanel` through a relative path even though `web/src` code is required to use the `@/*` alias. Fix by switching the local component import to the aliased system path.

## Resolution

- Updated the `NetworkChannelsListPanel` story to use the aliased component import and kept the touched stories explicit with `args: {}`.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.
