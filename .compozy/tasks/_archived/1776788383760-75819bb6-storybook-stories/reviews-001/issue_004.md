---
status: resolved
file: web/src/components/ui/stories/combobox.stories.tsx
line: 3
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:f679d66ec4e6
review_hash: f679d66ec4e6
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 004: Switch combobox story imports to @/* aliases.
## Review Comment

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - `combobox.stories.tsx` uses sibling relative imports even though `web/src` files are expected to use the `@/*` alias.
  - This is a direct import-contract mismatch, not a speculative preference.
  - Fix approach: rewrite the combobox component imports to `@/components/ui/combobox` and let typecheck/build validate the resolution.
