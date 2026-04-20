---
status: resolved
file: packages/ui/src/components/kind-chip.tsx
line: 17
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:e73d1b692f42
review_hash: e73d1b692f42
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 022: Prevent overriding internal data-slot / data-kind markers.
## Review Comment

Because `{...props}` is applied last, callers can override internal markers (e.g., `data-slot`, `data-kind`). If these are intended to be stable hooks, make internal attributes win.

## Triage

- Decision: `valid`
- Notes:
  - `KindChip` applies `{...props}` after internal `data-slot` and `data-kind`, so callers can override the component’s stable selectors and semantic marker.
  - Fix by moving the prop spread ahead of internal markers so the component-owned attributes always win, and add a regression test that external overrides are ignored.
