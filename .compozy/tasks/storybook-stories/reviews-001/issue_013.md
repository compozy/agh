---
status: resolved
file: web/src/components/ui/stories/popover.stories.tsx
line: 11
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:23a8371a70c8
review_hash: 23a8371a70c8
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 013: Use @/* alias instead of relative import
## Review Comment

Line 11 uses a relative import (`../popover`), which violates the web import convention and makes refactors more brittle.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - The popover story imports `../popover` instead of using the web alias.
  - That is a direct mismatch with the `@/*` import requirement for `web/src`.
  - Fix approach: switch to `@/components/ui/popover` and confirm resolution via the standard web checks.
