---
status: pending
file: web/src/components/design-system-showcase.tsx
line: 297
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:3eefbf5b72d3
review_hash: 3eefbf5b72d3
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 010: Avoid hardcoding token values in the showcase metadata.
## Review Comment

These new swatches duplicate literal color values that already live in `packages/ui/src/tokens.css`, but the current tests only verify token names. If the token file changes later, the showcase docs can silently drift. Consider deriving the displayed value from the token source or adding a sync assertion against `tokens.css`.

As per coding guidelines, `web/src/**/*.{tsx,ts,css}`: Design system tokens (colors, fonts, radius, spacing, motion) MUST be pulled from `DESIGN.md` in the repository root — never invent tokens or use ad-hoc hex values in components.

## Triage

- Decision: `UNREVIEWED`
- Notes:
