---
provider: coderabbit
pr: "113"
round: 2
round_created_at: 2026-05-06T21:09:03.43169Z
status: resolved
file: web/src/systems/agent/components/stories/agent-command-select.stories.tsx
line: 41
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4239579182,nitpick_hash:7a502b2ddcff
review_hash: 7a502b2ddcff
source_review_id: "4239579182"
source_review_submitted_at: "2026-05-06T21:08:32Z"
---

# Issue 001: Replace the ad-hoc width value with a design token.
## Review Comment

At Line 41, `w-[420px]` is an invented spacing value; please switch to a token-backed width utility/value from the design system.

As per coding guidelines, "Pull every color, font, radius, spacing step, and motion value from `DESIGN.md` — never invent tokens" and "`web/**/*.{css,tsx}`: Tokens live in `packages/ui/src/tokens.css`; never override with ad-hoc hex values in components".

---

## Triage

- Decision: `valid`
- Notes:
  - The story frame currently uses `w-[420px]`, which is an ad-hoc width outside the design token system.
  - The fix is constrained to the scoped story file: replace the hard-coded width with a standard responsive max-width utility already used across the web surface.
  - Verification after the fix passed with `make web-lint`, `make web-typecheck`, `make web-test`, and the full `make verify` gate.
