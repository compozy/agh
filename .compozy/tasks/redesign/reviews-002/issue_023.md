---
status: resolved
file: packages/ui/src/components/metric.test.tsx
line: 35
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:cfd56318a70c
review_hash: cfd56318a70c
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 023: Fix selector mismatch in the tone test block.
## Review Comment

On Line 35, `root` queries `metric-value` instead of the root `metric` slot, so the assertion is not validating what the variable name implies.

## Triage

- Decision: `valid`
- Notes:
  - The tone test assigns `root` from `[data-slot="metric-value"]` instead of the root `[data-slot="metric"]`, so the variable name and asserted target do not match.
  - Fix by querying the actual metric root node in that test block.
