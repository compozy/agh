---
status: resolved
file: packages/ui/src/components/avatar.test.tsx
line: 26
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:3a8e6d5da2d4
review_hash: 3a8e6d5da2d4
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 006: Image-slot test is asserting the wrong element.
## Review Comment

The case says it verifies image rendering, but it checks `avatar-fallback`. This can miss real regressions in image-slot rendering.

## Triage

- Decision: `valid`
- Reasoning: The "image slot" test looks up `data-slot="avatar-fallback"`, so it can pass even if the avatar image never renders.
- Root cause: The assertion targets the fallback slot instead of the image slot under test.
- Fix plan: Assert against the image slot and/or the rendered image element so the test fails on real image-rendering regressions.

## Resolution

- Reworked `packages/ui/src/components/avatar.test.tsx` to simulate a successful image load and assert the rendered `avatar-image` element instead of the fallback slot.
- Verified with `make verify` after all batch changes.
