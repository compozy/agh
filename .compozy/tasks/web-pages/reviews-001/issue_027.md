---
status: pending
file: web/src/systems/network/lib/network-formatters.ts
line: 151
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:c95a5d732f19
review_hash: c95a5d732f19
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 027: Prefer an interface for the metric card shape instead of an inline object type.
## Review Comment

Use a named `interface` for this return shape to align with project TS conventions.

As per coding guidelines, `web/**/*.ts?(x)`: Use `interface` for defining object shapes in TypeScript (pattern is in Zod schemas and types).

## Triage

- Decision: `UNREVIEWED`
- Notes:
