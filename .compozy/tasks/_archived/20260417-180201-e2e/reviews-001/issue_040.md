---
status: resolved
file: web/src/systems/automation/components/automation-detail-panel.test.tsx
line: 6
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:bb21175ddbb7
review_hash: bb21175ddbb7
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 040: Promote mocked Link props shape to an interface.
## Review Comment

The inline object shape works, but it violates the repo TypeScript shape convention.

As per coding guidelines, `web/**/*.ts?(x)`: Use `interface` for defining object shapes in TypeScript.

## Triage

- Decision: `valid`
- Notes:
  Same root cause as issue 039: the mocked `Link` props use an inline object
  type in a web TypeScript test file. Promote that shape to a named interface
  to match the repository convention.

## Resolution

- Promoted the mocked `Link` prop shape into named interfaces for reuse and
  clearer TypeScript test code.
