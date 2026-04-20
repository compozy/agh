---
status: resolved
file: packages/ui/src/components/sonner.test.tsx
line: 9
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:b9bdc1a50ec3
review_hash: b9bdc1a50ec3
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 035: Replace document.querySelector with semantic RTL queries.
## Review Comment

`screen.getByRole` aligns with React Testing Library best practices by making tests more robust against implementation changes and reflecting how users (including those using assistive technology) interact with the app.

## Triage

- Decision: `invalid`
- Notes:
  - The current selector already targets Sonner’s accessible notification container via its labelled section. Replacing it with `screen.getByRole(...)` is a test-style preference, not a defect in behavior coverage.
  - No correctness, accessibility, or stability bug is demonstrated by the current query.
