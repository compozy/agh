---
status: resolved
file: packages/ui/src/components/dialog.tsx
line: 175
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4136983224,nitpick_hash:e3bf9e7696ac
review_hash: e3bf9e7696ac
source_review_id: "4136983224"
source_review_submitted_at: "2026-04-20T02:30:11Z"
---

# Issue 017: Use DialogClose wrapper here for slot consistency.
## Review Comment

This path currently uses `DialogPrimitive.Close` directly, which skips your local wrapper conventions (`data-slot="dialog-close"`). Prefer reusing `DialogClose` for consistent markup and selectors.

## Triage

- Decision: `valid`
- Reasoning: The `DialogFooter` `showCloseButton` path uses `DialogPrimitive.Close` directly, which bypasses the local `DialogClose` wrapper and its stable `data-slot` convention.
- Root cause: The footer close button path is inconsistent with the component’s own wrapper API.
- Fix plan: Reuse `DialogClose` in the footer close path so markup and selectors stay consistent across close actions.

## Resolution

- Switched the footer close path in `packages/ui/src/components/dialog.tsx` to `DialogClose` and added a regression in `packages/ui/src/components/dialog.test.tsx` for the stable slot/close behavior.
- Verified with `make verify` after all batch changes.
