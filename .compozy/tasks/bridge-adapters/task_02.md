---
status: pending
title: "Redesign provider-scoped bridge runtime handshake and daemon lifecycle"
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 02: Redesign provider-scoped bridge runtime handshake and daemon lifecycle

## Overview

Replace the old "one bridge instance per extension process" launch contract with the approved provider-scoped runtime model. This task moves daemon boot and subprocess negotiation to a provider runtime context that can represent multiple managed bridge instances at once.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST redesign `InitializeBridgeRuntime` and related manager/daemon wiring so bridge-capable extensions negotiate a provider-scoped runtime context instead of a single assigned instance.
2. MUST remove the runtime assumption in daemon extension loading that exactly one enabled bridge instance exists per extension.
3. MUST preserve daemon ownership of lifecycle, route ownership, and secret binding resolution while allowing one provider runtime to manage many bridge instances internally.
4. SHOULD keep the new runtime context versioned and clone-safe so extension manager state, restart handling, and tests can retain it safely.
</requirements>

## Subtasks
- [ ] 2.1 Replace the instance-scoped handshake payload with provider-scoped runtime context and managed-instance snapshots
- [ ] 2.2 Refactor daemon bridge runtime resolution to stop failing on multiple enabled instances for one provider
- [ ] 2.3 Update extension manager runtime injection and restart paths for provider-scoped bridge sessions
- [ ] 2.4 Add integration coverage for provider runtime launch, restart, and multi-instance ownership

## Implementation Details

Follow the TechSpec sections "Approved Architecture", "Provider Runtime Model", and "Host API and Runtime Changes". This task should stop at runtime negotiation and lifecycle composition; it should not yet add new Host API methods or shared SDK ingress hardening.

### Relevant Files
- `internal/subprocess/handshake.go` — Current initialize payload still carries a single `Instance`
- `internal/daemon/bridges.go` — `instanceForExtension()` currently errors when multiple enabled instances share one extension
- `internal/extension/manager.go` — Bridge runtime resolution and subprocess initialize wiring assume instance-scoped bridge sessions
- `internal/extensiontest/bridge_adapter_harness.go` — Current harness materializes an old single-instance runtime contract

### Dependent Files
- `internal/extension/host_api_bridges.go` — Host API authorization later needs the provider-scoped runtime context
- `sdk/examples/telegram-reference/main.go` — Reference adapter/runtime later needs to consume the new handshake
- `internal/daemon/daemon_integration_test.go` — Existing bridge-runtime integration tests will need updated runtime assertions

### Reference Sources (.resources/)
- `.resources/chat/packages/chat/src/chat.ts` — Chat-SDK `Chat` class manages multiple adapters in one process with handler routing; shows multi-adapter multiplexing model
- `.resources/goclaw/internal/channels/dispatch.go` — GoClaw channel manager dispatch loop routing to/from many channels by name in one process
- `.resources/hermes/gateway/run.py` — Hermes `GatewayRunner` bootstraps multiple platform adapters, injects session handlers, coordinates lifecycle; reference for provider runtime boot with multiple owned instances
- `.resources/openclaw/src/channels/plugins/types.plugin.ts` — OpenClaw `ChannelPlugin` contract with `lifecycle?: ChannelLifecycleAdapter` (startup/shutdown hooks)

### Related ADRs
- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Defines the provider-scoped runtime contract this task must implement

## Deliverables
- Provider-scoped bridge initialize payload and clone helpers
- Refactored daemon and extension-manager lifecycle for many bridge instances per provider process
- Updated bridge runtime integration harness for the new launch model
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for multi-instance provider runtime boot and restart **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] provider-scoped initialize payload validation rejects empty provider identity and invalid managed-instance snapshots
  - [ ] cloned runtime contexts do not alias mutable slices or maps from the source runtime
  - [ ] daemon bridge runtime resolution no longer errors when multiple enabled instances share one extension
- Integration tests:
  - [ ] one bridge-capable extension launches successfully when two enabled bridge instances reference the same provider extension
  - [ ] restarting a provider-scoped runtime preserves daemon-owned bridge state and rehydrates the runtime context
  - [ ] provider-scoped runtime launch still fails cleanly when no enabled bridge instances exist for the extension
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Bridge runtime negotiation is provider-scoped rather than instance-scoped
- One provider process can launch cleanly with multiple owned bridge instances
