---
status: resolved
file: internal/api/udsapi/handlers_test.go
line: 501
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:4727b916571b
review_hash: 4727b916571b
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 021: Add binding assertions for the new /api/agent/... routes.
## Review Comment

Line 513 extends the shared binding map, but the new agent endpoints added above are still only covered by route presence. A miswire on `AgentTaskClaimNext`, `AgentTaskComplete`, or `AgentChannelReply` would still pass this suite.

As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling".

## Triage

- Decision: `VALID`
- Notes: `TestRegisterTaskRoutesUseSharedHandlerBindings` only asserts shared task route bindings and omits the new agent task/channel routes. Route presence coverage earlier in this file would not catch a handler miswire for `AgentTaskClaimNext`, `AgentTaskComplete`, or `AgentChannelReply`.
- Fix: Extend the binding expectation map with the agent routes and expected handler names.
