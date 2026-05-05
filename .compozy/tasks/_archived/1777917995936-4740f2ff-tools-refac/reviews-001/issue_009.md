---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/core/tools_test.go
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4204955814,nitpick_hash:345402853d55
review_hash: 345402853d55
source_review_id: "4204955814"
source_review_submitted_at: "2026-04-30T12:11:10Z"
---

# Issue 009: Add coverage for the session-search and single-toolset handlers.
## Review Comment

`newToolCoreEngine` wires `/sessions/:id/tools/search` and `/toolsets/:id`, but this suite never hits either path. Those two handlers have different scoping/parameter behavior than the operator list/search cases, so they can drift without any regression signal.

As per coding guidelines, `**/*_test.go`: `Focus on critical paths: workflow execution, state management, error handling`.

Also applies to: 236-248

## Triage

- Decision: `VALID`
- Notes: `newToolCoreEngine` registers `/sessions/:id/tools/search` and `/toolsets/:id`, but the existing core handler test only exercises operator search, session list, and toolset list. Add coverage for session-scoped search and single-toolset retrieval so their route-specific scope/parameter behavior cannot drift.
