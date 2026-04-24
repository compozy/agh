---
status: resolved
file: web/src/systems/session/components/session-resume-failure.test.tsx
line: 92
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167424608,nitpick_hash:e264c0c0c4be
review_hash: e264c0c0c4be
source_review_id: "4167424608"
source_review_submitted_at: "2026-04-24T02:13:16Z"
---

# Issue 005: Add an explicit spinner assertion in the retrying-state test.
## Review Comment

The test currently verifies `disabled`; asserting the loading icon is present would lock down the intended feedback state too.

## Triage

- Decision: `valid`
- Notes:
- The component intentionally swaps `RefreshCw` for a spinning `Loader2` icon while `isRetrying` is true, but the current test only verifies the button is disabled.
- Existing tests in the same session component area already assert `animate-spin` on retry/stop buttons, so adding the spinner assertion matches local testing practice and closes a regression gap.
- Fix plan: extend the retrying-state test to assert that the retry button's rendered SVG carries the `animate-spin` class.
- Implemented: the retrying-state test now checks both the disabled button state and the rendered spinner class.
- Verified with targeted component Vitest execution and the full web/repository gates (`make web-test`, `make verify`).
