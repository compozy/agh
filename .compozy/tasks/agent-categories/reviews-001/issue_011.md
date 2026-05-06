---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: web/src/systems/agent/lib/agent-category.ts
line: 3
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4239418533,nitpick_hash:d66345482b92
review_hash: d66345482b92
source_review_id: "4239418533"
source_review_submitted_at: "2026-05-06T20:41:36Z"
---

# Issue 011: Use interface for exported object-shape contracts.
## Review Comment

`AgentCategoryFolderNode` and `AgentCategoryLeafNode` are object-shape APIs and should be declared as `interface` to match repo TS conventions.

As per coding guidelines: `**/*.{ts,tsx}`: Use TypeScript `interface` (not `type`) for defining object shapes.

## Triage

- Decision: `valid`
- Notes:
  - `AgentCategoryFolderNode` and `AgentCategoryLeafNode` are still exported as `type` object literals.
  - Root cause: the file is using type aliases for object-shape contracts where the repo convention requires exported interfaces.
  - Fix approach: convert those exported object-shape contracts to `interface` declarations while keeping the union type for `AgentCategoryNode`.
