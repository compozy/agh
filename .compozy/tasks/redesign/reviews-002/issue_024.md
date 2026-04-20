---
status: resolved
file: packages/ui/src/components/mono-badge.tsx
line: 48
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:7f5727757d06
review_hash: 7f5727757d06
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 024: Prevent overriding internal data attributes via trailing prop spread.
## Review Comment

Placing `{...props}` last allows consumers to override `data-slot`/`data-tone`, which can desync the component’s internal contract and selectors.

## Triage

- Decision: `valid`
- Notes:
  - `MonoBadge`'s trailing prop spread lets callers override `data-tone`, which can desynchronize the semantic marker from the selected visual variant.
  - `data-slot` is intentionally caller-owned for composite slot integration (for example, `ToolCallCard` exposes its badge via a custom slot), so the safe fix is to keep `data-slot` passthrough while making `data-tone` component-owned.
  - Regression coverage requires touching adjacent existing test file `packages/ui/src/components/mono-badge.test.tsx`, which is outside the listed batch code files but is the colocated test surface for this component.
