---
status: resolved
file: web/src/routes/_app/-automation.integration.test.tsx
line: 59
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:8efcc154caff
review_hash: 8efcc154caff
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 039: Use an interface for mocked Link props shape.
## Review Comment

This works, but the inline object type for `params` should be promoted to an `interface` to match repo TS shape conventions.

As per coding guidelines, `web/**/*.ts?(x)`: Use `interface` for defining object shapes in TypeScript.

## Triage

- Decision: `valid`
- Notes:
  The mocked `Link` props shape is currently an inline object type. This file is
  under the web TypeScript conventions that prefer `interface` for object
  shapes, so the mock should promote the shape to a named interface.

## Resolution

- Promoted the mocked `Link` prop shape into named interfaces for reuse and
  clearer TypeScript test code.
