---
status: completed
title: Agent Caller Identity Layer
type: backend
complexity: medium
dependencies:
  - task_01
  - task_02
---

# Task 05: Agent Caller Identity Layer

## Overview
Add the identity resolution layer used by agent-facing UDS and CLI commands. Operator commands remain explicit, while agent commands infer the caller from daemon-issued environment variables and validate that the session is known, active, and authorized.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, and ADR-010 before changing CLI/UDS identity behavior
- REFERENCE TECHSPEC for env names, JSON/JSONL conventions, and exit-code taxonomy
- FOCUS ON "WHAT" - caller identity and audit context, not individual task/channel verbs
- MINIMIZE CODE - centralize identity resolution instead of copying env parsing into every command
- TESTS REQUIRED - missing, stale, invalid, and mismatched identity paths must be covered
- NO WORKAROUNDS - do not trust raw environment values without daemon validation
</critical>

<requirements>
- MUST resolve agent caller identity from `AGH_SESSION_ID`, `AGH_AGENT`, and daemon session status.
- MUST distinguish agent-facing identity inference from operator-facing explicit commands.
- MUST produce task actor/origin contexts for agent UDS operations without allowing privilege escalation.
- MUST define stable JSON/JSONL output conventions and exit-code mapping for agent namespaces.
- MUST reject missing, stale, stopped, mismatched, or unauthorized session identity with actionable errors.
- MUST not remove or degrade existing user-created task and user-started session flows.
</requirements>

## Subtasks
- [x] 5.1 Add shared CLI/UDS caller identity resolver and typed error mapping.
- [x] 5.2 Add daemon/client helpers for session lookup and actor/origin derivation.
- [x] 5.3 Add JSON/JSONL output and exit-code conventions for future `me`, `ch`, `task`, and `spawn` commands.
- [x] 5.4 Add tests for missing env, stale session, stopped session, agent mismatch, workspace mismatch, and valid identity.
- [x] 5.5 Confirm manual operator commands still require explicit flags and do not infer identity accidentally.

## Implementation Details
Keep identity resolution close to transport/CLI boundaries and use existing `session.Status`, task actor types, and UDS client patterns. This is a security boundary: invalid env input must fail closed.

### Relevant Files
- `internal/cli/root.go` - command dependency injection and env helpers.
- `internal/cli/client.go` - UDS client interface and request methods.
- `internal/api/core/interfaces.go` - session/task service surfaces.
- `internal/api/udsapi/server.go` and `internal/api/udsapi/routes.go` - UDS transport binding.
- `internal/session/session.go` - session info/state model.
- `internal/task/types.go` - actor/origin types used by task authority.
- `.resources/paperclip/doc/plans/2026-02-18-agent-authentication.md` - reference for agent authentication semantics.
- `.resources/paperclip/cli/src/__tests__/agent-jwt-env.test.ts` - reference for env-derived agent identity tests.
- `.resources/claude-code/setup.ts` - reference for session environment setup.

### Dependent Files
- `internal/cli/task.go`, `internal/cli/network.go` - later agent verbs consume the identity layer.
- `internal/api/udsapi/*` - later agent endpoints consume resolved caller identity.
- `internal/daemon/task_runtime.go` - later task actors use agent session identity.

### Related ADRs
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - CLI-first identity inference.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual operator flows remain explicit.

## Deliverables
- Shared agent caller identity resolver for CLI/UDS.
- Actor/origin derivation helpers for agent-session calls.
- Output/exit-code conventions for agent namespaces.
- Unit tests with 80%+ coverage for identity resolution **(REQUIRED)**.
- Integration tests proving operator flows do not accidentally infer agent identity **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Missing `AGH_SESSION_ID` returns a specific identity-required error.
  - [x] Unknown or stopped session returns a stale identity error.
  - [x] `AGH_AGENT` mismatch against session agent name fails closed.
  - [x] Valid session identity yields `ActorKindAgentSession` and `OriginKindUDS`/CLI as appropriate.
  - [x] JSON and JSONL output helpers render stable machine-readable errors.
- Integration tests:
  - [x] Existing `agh task create --workspace ...` remains operator-explicit even with agent env variables present.
  - [x] UDS handler tests prove invalid caller identity cannot access agent endpoints.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Agent commands have one validated caller identity path.
- Manual commands remain first-class and explicit.
