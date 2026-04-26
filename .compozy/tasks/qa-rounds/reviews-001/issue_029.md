---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 79
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:f4ce083715a2
review_hash: f4ce083715a2
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 029: Avatar palette uses hardcoded hex values.
## Review Comment

The `AVATAR_PALETTE` contains ad-hoc color tokens. Per coding guidelines, colors should be pulled from `DESIGN.md` rather than invented. Consider defining these as CSS variables in `tokens.css` or confirming they align with the design system's avatar color specification.

As per coding guidelines: "Pull every color, font, radius, spacing step, and motion value from `DESIGN.md` — never invent tokens"

---

## Triage

- Decision: `valid`
- Notes: `AVATAR_PALETTE` uses literal hex colors in the component. AGH's design system already exposes the needed semantic tint and text variables, so the fix is to express avatar background/foreground pairs with existing CSS token variables rather than invented literals.
