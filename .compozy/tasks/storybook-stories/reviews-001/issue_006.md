---
status: resolved
file: web/src/components/ui/stories/command.stories.tsx
line: 21
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:62381109d541
review_hash: 62381109d541
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 006: Replace relative component import with @/* alias.
## Review Comment

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - The story imports its local command primitives through `../command`, which violates the `@/*` import rule for `web/src`.
  - This is the same root cause as the other alias comments in this batch: newly added stories were authored with relative sibling imports.
  - Fix approach: convert the story to `@/components/ui/command` imports and verify via web lint/typecheck/build.
