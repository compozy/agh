---
status: resolved
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

- Decision: `valid`
- Root cause: `getNetworkMetricCards` returns an inline object-shape array instead of a named interface, which is inconsistent with the project’s TypeScript object-shape convention.
- Fix approach: introduce a small named interface for the metric-card shape and use it as the return type. No behavior change is required.
- Resolution: introduced a dedicated `NetworkMetricCard` interface and applied it to the formatter return type.
- Verification: route tests plus `make web-lint`, `make web-typecheck`, and `make verify` passed.
