---
status: resolved
file: packages/ui/src/components/connection-indicator.tsx
line: 35
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:221ebfab8b3b
review_hash: 221ebfab8b3b
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 014: Consider status semantics for live connection changes.
## Review Comment

For dynamic status updates, adding `role="status"` and `aria-live="polite"` on the container improves screen-reader announcements.

## Triage

- Decision: `valid`
- Reasoning: `ConnectionIndicator` is the component that surfaces live connection state to the operator UI. Without live-region semantics, status transitions may not be announced to assistive technology users.
- Root cause: The rendered container exposes the visible state label but no status/live-region semantics.
- Fix plan: Add polite status semantics on the container while preserving the existing visual API.

## Resolution

- Added default `role="status"` and `aria-live="polite"` semantics in `packages/ui/src/components/connection-indicator.tsx`.
- Added a regression in `packages/ui/src/components/connection-indicator.test.tsx` because the existing companion test file lives outside the initial code-file list, and verified with `make verify`.
