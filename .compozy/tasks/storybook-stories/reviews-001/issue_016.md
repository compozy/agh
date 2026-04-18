---
status: resolved
file: web/src/components/ui/stories/switch.stories.tsx
line: 4
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:90d3a79dbd3b
review_hash: 90d3a79dbd3b
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 016: Use @/* alias for local imports.
## Review Comment

Lines 4-5 should use alias-based imports instead of relative paths.

As per coding guidelines, `web/src/**/*.{ts,tsx}`: "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - `switch.stories.tsx` imports both `field` and `switch` via relative paths inside `web/src`.
  - The active web instructions require `@/*` imports for these modules.
  - Fix approach: rewrite the local imports to alias-based paths and verify with the web checks.
