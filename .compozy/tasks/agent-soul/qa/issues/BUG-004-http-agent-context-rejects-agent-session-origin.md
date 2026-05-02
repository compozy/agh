# BUG-004: HTTP Agent Context Rejected Valid Agent-Session Origin

## Status

Fixed in this QA pass.

## Severity

P0 for HTTP/API parity, because `/api/agent/context` and `/api/agent/soul` are documented agent-facing runtime surfaces.

## Scenario

During TC-SCEN-003, a real provider-backed `reviewer` session was created successfully in the isolated lab. The UDS/CLI surface `agh me context` returned the expected Soul projection, but the HTTP agent-facing routes returned `500 Internal Server Error` with valid `X-AGH-Session-ID` and `X-AGH-Agent` headers.

## Root Cause

HTTP agent-facing requests were mapped to `task.OriginKindHTTP`, but `task.DeriveAgentSessionActorContextForOrigin` and actor/origin validation only allowed `cli`, `uds`, and `agent_session` origins for agent-session actors. The identity resolved correctly, then failed while deriving the actor context.

## Fix

- Allowed `OriginKindHTTP` for authenticated agent-session ingress in `internal/task/actors.go`.
- Added focused coverage in `internal/task/actors_test.go`.
- Added HTTP route coverage in `internal/api/httpapi/agent_context_test.go`.

## Evidence

- Failing runtime evidence before fix:
  - `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-003-agent-context-http-debug.log`
  - `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-003-agent-soul-http-debug.log`
- Focused regression evidence:
  - `.compozy/tasks/agent-soul/qa/evidence/BUG-004-http-agent-context-go-test.log`
  - `.compozy/tasks/agent-soul/qa/evidence/BUG-004-task-actors-test-conventions.log`
  - `.compozy/tasks/agent-soul/qa/evidence/BUG-004-http-agent-context-test-conventions.log`
- Real lab verification after rebuild/restart:
  - `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-003-agent-context-http-after-fix.log`
  - `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-003-agent-soul-http-after-fix.log`
