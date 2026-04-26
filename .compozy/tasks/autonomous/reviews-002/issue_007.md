---
status: resolved
file: internal/api/spec/spec.go
line: 1438
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tk,comment:PRRC_kwDOR5y4QM67YhqL
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Document the real 403/503 failure modes on the new agent routes.**

The new `/api/agent/*` operations mostly advertise only `401`/`404`/`500`, but the current implementation already emits other statuses. `/api/agent/me` returns `403 Forbidden` on workspace mismatch and `503 Service Unavailable` when the session service is missing in `internal/api/udsapi/agent_identity_test.go`, and the handlers in `internal/api/core/agent_channels.go` return `503` when `AgentContextService` or the network service is unavailable. Generated clients will miss supported failure modes if the spec stays narrower than the handlers.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/spec/spec.go` around lines 1212 - 1438, The OpenAPI specs for
the agent endpoints (OperationIDs like getAgentMe, getAgentContext,
listAgentChannels, receiveAgentChannelMessages, sendAgentChannelMessage,
replyAgentChannelMessage, claimNextAgentTask, heartbeatAgentTaskRun,
completeAgentTaskRun, failAgentTaskRun, releaseAgentTaskRun, spawnAgentSession,
getAgentCoordinatorConfig) omit real failure modes; update each ResponseSpec to
include 403 (e.g., "Forbidden — workspace or permission mismatch") and 503
(e.g., "Service unavailable — dependent service missing") entries with Body:
contract.ErrorPayload{} so generated clients reflect handlers that return 403
and 503. Ensure /api/agent/me definitely includes 403 and 503 and mirror the
same additions on channel/task/spawn/coordinator endpoints that can return
service-unavailable or permission-denied errors.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The OpenAPI response specs for the new agent routes omit status codes the handlers already emit. Identity resolution can return `403` for workspace mismatch and `503` for unavailable session lookup; channel/context routes can return `503` when dependent services are missing; task/spawn/coordinator routes can return permission or service-unavailable failures through shared status mapping. The fix is to document `403` and `503` error payloads on the agent operations and regenerate the derived OpenAPI/client artifacts because this touches `internal/api/spec`.
- Resolution: Added the missing agent-route error responses, regenerated derived OpenAPI/TypeScript contracts, and verified with `make codegen-check`, web checks, and full `make verify`.
