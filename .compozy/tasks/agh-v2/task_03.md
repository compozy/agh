---
status: pending
domain: Runtime
type: Feature Implementation
scope: Full
complexity: high
dependencies:
  - task_01
---

# Task 03: ACP Package

## Overview

Implement the `internal/acp` package — the ACP (Agent Client Protocol) client that spawns agent subprocesses and communicates via JSON-RPC 2.0 over stdio. This package implements the `AgentDriver` interface consumed by `session/`, handles bidirectional ACP messages (including agent requests like `fs/readTextFile`, `terminal/create`, `request_permission`), and enforces the permission model with path sandboxing.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use `github.com/coder/acp-go-sdk` as the ACP client library
- MUST spawn agent subprocess using provider command string (parsed into executable + args)
- MUST perform ACP `initialize` handshake and capture agent capabilities (modes, models, loadSession support)
- MUST call `session/new` with `cwd` and `mcpServers` parameters
- MUST call `session/prompt` and stream back `session/update` notifications as `AgentEvent` channel
- MUST support `session/load` for resume (when agent supports it)
- MUST support `session/cancel` for cooperative cancellation
- MUST handle agent-to-client requests: `fs/readTextFile`, `fs/writeTextFile`, `terminal/create`, `terminal/output`, `terminal/waitForExit`, `terminal/kill`, `request_permission`
- MUST enforce permission policy (deny-all, approve-reads, approve-all) with path sandboxing (cwd boundary)
- MUST detect process crash via `cmd.Wait()` and report it
- MUST capture token usage from `PromptResponse.usage` (nullable fields)
- MUST capture `usage_update` notifications when available (unstable ACP feature)
</requirements>

## Subtasks
- [ ] 3.1 Define ACP client types (AgentProcess, StartOpts, PromptRequest, AgentEvent, ACPCaps, TokenUsage)
- [ ] 3.2 Implement subprocess spawning from provider command string (split command, set env, pipe stdio)
- [ ] 3.3 Implement ACP initialize handshake and capability capture
- [ ] 3.4 Implement session/new with cwd and mcpServers
- [ ] 3.5 Implement session/prompt with streaming AgentEvent channel from session/update notifications
- [ ] 3.6 Implement session/load for resume and session/cancel for cancellation
- [ ] 3.7 Implement agent-to-client request handlers (fs/*, terminal/*, request_permission)
- [ ] 3.8 Implement permission policy enforcement with path sandboxing
- [ ] 3.9 Implement process crash detection and cleanup
- [ ] 3.10 Implement token usage capture from PromptResponse and usage_update

## Implementation Details

Create the following files:
- `internal/acp/client.go` — ACP client, subprocess lifecycle, JSON-RPC communication
- `internal/acp/handlers.go` — Agent-to-client request handlers (fs, terminal, permission)
- `internal/acp/permission.go` — Permission policy enforcement, path sandboxing
- `internal/acp/types.go` — AgentProcess, AgentEvent, StartOpts, ACPCaps, TokenUsage

The `acp` package exports types that satisfy the `session.AgentDriver` interface without importing `session/`.

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — ACP Integration Points, Permission Model, Data Models

### Old Project Reference
- `.old_project/internal/drivers/claude/claude.go` — Previous driver implementation (PTY-based, but useful for spawn patterns)
- `.old_project/internal/pty/process.go` — Subprocess spawning and I/O handling patterns
- `.old_project/internal/pty/manager.go` — Process lifecycle management
- `.old_project/internal/drivers/codex/codex.go` — Alternative driver patterns

### Related ADRs
- [ADR-003: ACP Internally, HTTP/SSE Externally](../adrs/adr-003.md) — ACP as internal protocol
- [ADR-005: Built-In Provider Registry With ACP Commands](../adrs/adr-005.md) — Provider command resolution
- [ADR-008: Direct Interfaces and Notifier Pattern](../adrs/adr-008.md) — AgentDriver interface pattern

## Deliverables
- `internal/acp/` package with full ACP client implementation
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration test with mock ACP server **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Parse provider command string into executable + args correctly
  - [ ] Permission policy: deny-all rejects all operations
  - [ ] Permission policy: approve-reads allows readTextFile, denies writeTextFile
  - [ ] Permission policy: approve-all allows all operations
  - [ ] Path sandboxing: paths within cwd allowed
  - [ ] Path sandboxing: paths outside cwd denied (including ../ traversal)
  - [ ] Token usage: parse PromptResponse.usage with all fields
  - [ ] Token usage: handle nil/missing fields gracefully
  - [ ] AgentEvent channel receives streamed session/update notifications
  - [ ] Process crash detected and reported correctly
- Integration tests:
  - [ ] Full ACP round-trip with mock server: initialize → session/new → session/prompt → session/update → done
  - [ ] Mock server sending fs/readTextFile request, client responds correctly
  - [ ] Mock server sending request_permission, client applies policy
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Can spawn a subprocess and complete ACP initialize handshake
- Permission enforcement works correctly for all three modes
- Path sandboxing prevents access outside session cwd
