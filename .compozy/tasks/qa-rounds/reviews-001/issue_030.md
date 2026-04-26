---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 432
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:3dc982df56e7
review_hash: 3dc982df56e7
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 030: Inline color value for hover state.
## Review Comment

`hover:bg-white/[0.014]` uses an ad-hoc opacity value. Consider using a CSS variable-based hover surface token for consistency with the flat depth model.

As per coding guidelines: "Tokens live in `packages/ui/src/tokens.css`; never override with ad-hoc hex values in components"

## Triage

- Decision: `valid`
- Notes: The message-row hover state uses `hover:bg-white/[0.014]`, an ad-hoc opacity value outside the token system. The fix is to use the existing hover surface token via `var(--color-hover)`.
