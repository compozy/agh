---
status: resolved
file: packages/ui/src/components/kind-chip.test.tsx
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:7d1cc452f926
review_hash: 7d1cc452f926
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 024: Potential brittleness in style comparison.
## Review Comment

`style.background` may return normalized values (e.g., `rgb()` format) depending on the test environment, while `KIND_DOT_COLORS.receipt` is `var(--color-success)`. This should work in jsdom since inline styles aren't computed, but be aware if tests become flaky after environment changes.

## Triage

- Decision: `valid`
- Notes: The current assertion compares `dot.style.background` directly to the token string. That happens to work in the current jsdom environment but couples the test to one CSSOM serialization path. The fix is to assert the inline style through the jest-dom style matcher, which is the existing test stack's style assertion API.
