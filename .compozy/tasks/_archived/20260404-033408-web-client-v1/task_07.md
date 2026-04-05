---
status: completed
title: Backend — Interactive Permission Approval Endpoint
type: ""
complexity: high
dependencies:
    - task_05
---

# Task 07: Backend — Interactive Permission Approval Endpoint

## Overview

Implement the `POST /api/sessions/:id/approve` endpoint in the Go backend to support interactive permission approval from the web client. Currently, the ACP permission handler decides immediately based on policy and returns a response to the agent without waiting. This task adds a channel-based blocking mechanism so that when a permission mode requires user input, the handler waits for the user's decision via the HTTP endpoint before responding to the agent.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE internal/acp/handlers.go `handleRequestPermission()` for current auto-decide flow
- REFERENCE internal/acp/permission.go `selectPermissionOutcome()` for decision → outcome mapping
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- MUST run `make verify` (fmt → lint → test → build) before completion
</critical>

<requirements>
- MUST implement `POST /api/sessions/:id/approve` accepting JSON body: `{ "turn_id": string, "decision": string }` where decision is one of: `"allow-once"`, `"allow-always"`, `"reject-once"`, `"reject-always"`
- MUST add a pending permission storage mechanism to `AgentProcess` — a thread-safe map of pending permission requests indexed by a request ID, each holding a response channel
- MUST modify `handleRequestPermission()` in `internal/acp/handlers.go` to: when permission mode is interactive (not auto-decided), emit the permission event to the SSE stream AND block on a response channel until the user POSTs approval or a timeout occurs
- MUST add a timeout for pending permissions (e.g., 5 minutes) — if no approval arrives, auto-deny and unblock the handler
- MUST add `ApprovePermission(ctx context.Context, id string, req ApproveRequest) error` method to the session Manager interface and implementation
- MUST route approval from HTTP handler → SessionManager → Session → AgentProcess, resolving the pending channel with the user's decision
- MUST handle error cases: session not found (404), no pending permission for turn_id (409 Conflict), session not active (400), invalid decision value (400)
- MUST emit a follow-up permission event to the SSE stream after approval with the final decision so the frontend can update UI
- MUST be thread-safe — multiple permissions could theoretically be pending (though rare), and approval must match the correct pending request
- MUST include the permission options (allow_once, allow_always, reject_once, reject_always) in the SSE permission event `raw` field so the frontend knows which options are available
- MUST include a unique `request_id` in the SSE permission event payload (alongside turn_id) so the frontend can disambiguate concurrent permission requests. The approve endpoint should accept `request_id` (preferred) or fall back to `turn_id` for matching
- MUST enrich the SSE permission event with parsed `tool_input` (from the ACP request's raw data) so the frontend can display what the tool wants to do without parsing raw JSON
- MUST pass `make verify` (fmt, lint, test, build) with zero issues
</requirements>

## Subtasks
- [x] 7.1 Add pending permission storage (channel map) to `AgentProcess` in `internal/acp/types.go`
- [x] 7.2 Modify `handleRequestPermission()` in `internal/acp/handlers.go` to block on response channel when interactive approval is needed
- [x] 7.3 Add `ResolvePermission()` method to `AgentProcess` to unblock pending permission with user decision
- [x] 7.4 Add `ApprovePermission()` to session Manager interface and implementation
- [x] 7.5 Implement `approveSession` HTTP handler in `internal/httpapi/sessions.go`
- [x] 7.6 Add timeout mechanism for pending permissions (auto-deny after 5 min)
- [x] 7.7 Add tests and verify `make verify` passes

## Implementation Details

The core change is in `internal/acp/handlers.go` `handleRequestPermission()`. Currently (lines 208-243), it:
1. Decides based on policy → emits permission event → returns response immediately

The new flow for interactive mode:
1. Checks if permission mode requires user input (e.g., `deny-all` mode needs approval for all tools)
2. Creates a response channel, stores it in `AgentProcess.pendingPermissions[requestID]`
3. Emits `EventTypePermission` event to SSE stream (includes options in `Raw` field)
4. Blocks on the response channel (with timeout via `select` + `time.After`)
5. When user POSTs to `/api/sessions/:id/approve`, it calls `AgentProcess.ResolvePermission(requestID, decision)`
6. `ResolvePermission` sends the decision to the channel, unblocking the handler
7. Handler converts decision to `acpsdk.RequestPermissionOutcome` via `selectPermissionOutcome()`
8. Returns outcome to agent

The request ID should come from the ACP SDK's permission request (check if `RequestPermissionRequest` has an ID field, otherwise use turn_id + tool name as composite key).

See `internal/acp/permission.go:183-207` for `selectPermissionOutcome()` which maps decisions to ACP SDK outcomes.

### Relevant Files
- `internal/acp/handlers.go` — `handleRequestPermission()` at line ~208 — modify to block on channel
- `internal/acp/permission.go` — `selectPermissionOutcome()`, `permissionDecision` type, `permissionPolicy` interface
- `internal/acp/types.go` — `AgentProcess` struct (add pendingPermissions map), `activePromptState`
- `internal/httpapi/sessions.go` — `approveSession()` stub at line ~177 — implement handler
- `internal/httpapi/server.go` — `SessionManager` interface at line ~37 — add `ApprovePermission` method
- `internal/session/manager.go` — Manager implementation — add `ApprovePermission` method
- `internal/session/session.go` — Session struct — add pass-through to agent process
- `internal/session/interfaces.go` — `AgentDriver` interface — may need extension

### Dependent Files
- `internal/httpapi/server_test.go` — Tests for HTTP handler
- `internal/acp/handlers_test.go` — Tests for permission blocking flow
- `internal/session/manager_test.go` — Tests for ApprovePermission routing
- Task 08 (frontend permissions) depends on this endpoint working

## Deliverables
- Working `POST /api/sessions/:id/approve` endpoint accepting `{ turn_id, decision }`
- Channel-based blocking mechanism in ACP permission handler
- Timeout for unresolved permissions (5 min auto-deny)
- Thread-safe pending permission storage
- `ApprovePermission` method through the full stack (HTTP → Manager → Session → AgentProcess)
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for the approval flow **(REQUIRED)**
- `make verify` passing (fmt → lint → test → build)

## Tests
- Unit tests:
  - [x] `approveSession` handler returns 400 for missing `decision` field
  - [x] `approveSession` handler returns 400 for invalid `decision` value (e.g., "maybe")
  - [x] `approveSession` handler returns 404 for non-existent session
  - [x] `approveSession` handler returns 409 when no pending permission matches `turn_id`
  - [x] `approveSession` handler returns 200 and resolves permission on valid request
  - [x] `AgentProcess.ResolvePermission()` unblocks waiting handler with correct decision
  - [x] `AgentProcess.ResolvePermission()` returns error for unknown request ID
  - [x] Permission timeout auto-denies after configured duration
  - [x] Concurrent permission resolve is thread-safe (no races with `-race` flag)
  - [x] `selectPermissionOutcome` maps "allow-once" to `PermissionOptionKindAllowOnce`
  - [x] `selectPermissionOutcome` maps "reject-always" to `PermissionOptionKindRejectAlways`
- Integration tests:
  - [x] Full flow: permission request → SSE event emitted → POST approve → agent receives outcome
  - [x] Permission timeout: request → no approval → auto-deny after timeout
- Test coverage target: >=80%
- All tests must pass
- `make verify` passes with zero issues

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt → lint → test → build) with zero issues
- `POST /api/sessions/:id/approve` returns 200 on valid approval
- Permission blocking works: agent waits for user decision before proceeding
- Timeout works: unresolved permissions auto-deny after 5 minutes
- No race conditions (verified by `-race` flag in tests)
