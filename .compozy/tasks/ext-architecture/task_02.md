---
status: completed
title: Shared subprocess lifecycle package
type: backend
complexity: high
dependencies: []
---

# Task 02: Shared subprocess lifecycle package

## Overview

Extract reusable subprocess lifecycle primitives from `internal/acp/client.go` into a new `internal/subprocess/` package. This provides the foundation for both ACP agent communication and extension subprocess management. The package handles process spawning, JSON-RPC 2.0 framing over stdio, initialize handshake with capability negotiation, health monitoring, and graceful shutdown with signal escalation — all conforming to the protocol spec in `_protocol.md`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/subprocess/` package with `Process` struct managing a single subprocess
- MUST implement `Launch(ctx, LaunchConfig) (*Process, error)` for process spawning with platform-specific process group setup
- MUST implement bidirectional JSON-RPC 2.0 framing over stdin/stdout (one JSON object per line)
- MUST implement `Call(ctx, method, params, result) error` for outbound requests
- MUST implement `HandleMethod(method, handler)` for inbound request routing
- MUST implement `Shutdown(ctx) error` with cooperative drain + signal escalation (SIGTERM → wait → SIGKILL) per protocol spec section 8
- MUST implement initialize handshake per protocol spec section 4 (capability negotiation, version check)
- MUST implement health check probing per protocol spec section 7 (interval, timeout, unhealthy threshold)
- MUST evaluate `sourcegraph/jsonrpc2` as the JSON-RPC library — use it if suitable, otherwise implement minimal framing
- MUST refactor `internal/acp/client.go` to import shared subprocess primitives where applicable without breaking existing ACP tests
- MUST NOT break any existing tests in `internal/acp/`
</requirements>

## Subtasks
- [x] 2.1 Create `internal/subprocess/` package with `Process` struct and `LaunchConfig`
- [x] 2.2 Implement JSON-RPC 2.0 transport layer (line-delimited, bidirectional, multiplexed)
- [x] 2.3 Implement initialize handshake with capability negotiation per protocol spec section 4
- [x] 2.4 Implement health check probing with configurable interval, timeout, and unhealthy threshold
- [x] 2.5 Implement graceful shutdown with signal escalation per protocol spec section 8
- [x] 2.6 Refactor `internal/acp/client.go` to use shared subprocess primitives where possible
- [x] 2.7 Write unit and integration tests for the subprocess lifecycle

## Implementation Details

New package `internal/subprocess/` with files: `process.go`, `transport.go`, `handshake.go`, `health.go`, `signals.go`.

See TechSpec "Core Interfaces" section for `Process` struct. See `_protocol.md` sections 1-4, 7-8 for normative wire-level contract.

The ACP refactor (subtask 2.6) should be incremental — extract what's cleanly shareable without forcing ACP into the extension protocol shape. The ACP uses `coder/acp-go-sdk` for its own JSON-RPC; the shared package provides an independent framing layer.

### Relevant Files
- `internal/acp/client.go` — Current subprocess lifecycle (Start, Stop, signal handling) to extract from
- `internal/acp/process_tree_unix.go` — Platform-specific process group setup (Setpgid, SIGTERM/SIGKILL)
- `internal/acp/process_tree_windows.go` — Windows process termination
- `internal/acp/types.go` — `StartOpts`, `AgentProcess` types
- `internal/procutil/procutil.go` — Process alive check and signal helpers

### Dependent Files
- `internal/acp/client.go` — Will be refactored to import shared subprocess primitives
- `internal/extension/manager.go` — Will use subprocess package to manage extension processes (task 06)
- `internal/extension/host_api.go` — Will use subprocess transport for Host API (task 07)

### Related ADRs
- [ADR-004: Generalize ACP as Subprocess Extension Protocol](adrs/adr-004.md) — This task implements the shared lifecycle
- [ADR-001: Two-Tier Extension Model](adrs/adr-001.md) — L3 subprocess tier

## Deliverables
- New `internal/subprocess/` package with process management, JSON-RPC transport, handshake, health, signals
- Refactored `internal/acp/client.go` using shared primitives where applicable
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for subprocess launch → handshake → call → shutdown lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `Launch()` spawns process and connects stdin/stdout
  - [x] `Call()` sends JSON-RPC request and receives response
  - [x] `Call()` with context cancellation returns error before timeout
  - [x] `HandleMethod()` routes inbound requests to correct handler
  - [x] Initialize handshake succeeds with compatible versions
  - [x] Initialize handshake fails with `-32602` for unsupported protocol version
  - [x] Health check marks extension unhealthy after 2 consecutive probe failures
  - [x] Health check with `healthy: false` response marks unhealthy immediately
  - [x] Shutdown sends cooperative request then escalates signals
  - [x] Shutdown SIGKILL after timeout if process doesn't exit
  - [x] JSON-RPC framing handles one JSON object per line correctly
  - [x] Messages exceeding 10 MiB are rejected
- Integration tests:
  - [x] End-to-end: launch test subprocess → handshake → call → shutdown
  - [x] Crash recovery: subprocess exits unexpectedly → Process detects exit
  - [x] Concurrent requests: multiple outstanding requests resolve correctly
- Test coverage target: >=80%
- All existing `internal/acp/` tests must continue passing

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/subprocess/` package exists and compiles
- `internal/acp/` tests still pass after refactor
- `make verify` passes
