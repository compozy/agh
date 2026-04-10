---
status: pending
title: Transport, config, and audit foundation
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Transport, config, and audit foundation

## Overview

Build the infrastructure layer that boots embedded NATS, carries the daemon-only broker credential in memory, and persists network audit records. This task also introduces the new `[network]` configuration surface and home-path entries required by the runtime.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `NetworkConfig` to the merged AGH config with defaults and validation matching the tech spec
- MUST implement embedded NATS server lifecycle and daemon-side client connection with broker token retained only in daemon memory
- MUST extend `HomePaths` and storage surfaces for the network audit file without publishing any broker credential path
- MUST add persistent audit storage in `globaldb` for network events while keeping spaces and presence runtime-only
</requirements>

## Subtasks
- [ ] 2.1 Add `[network]` configuration structs, defaults, merge behavior, and validation tests
- [ ] 2.2 Implement embedded transport startup, shutdown, publish, subscribe, and reconnect-ready plumbing
- [ ] 2.3 Add network audit writer surfaces for append-only file output and `globaldb` persistence
- [ ] 2.4 Cover transport boot, config validation, and audit persistence with tests

## Implementation Details

Keep credential handling inside daemon-owned runtime state. This task should not expose direct broker connectivity to CLI, skills, or extensions, and it should not create a persisted space catalog.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Transport, config, audit, and token-auth sections
- `internal/config/config.go` - Add `NetworkConfig` to the merged configuration model
- `internal/config/merge.go` - Extend config overlay behavior for the new network section
- `internal/config/home.go` - Add filesystem paths for audit output
- `internal/store/globaldb/global_db.go` - Extend schema creation with network audit persistence
- `internal/network/transport.go` - New embedded NATS transport boundary
- `internal/network/audit.go` - New audit writer and storage adapter surface

### Dependent Files
- `internal/network/router.go` - Router depends on transport publish and subscribe surfaces
- `internal/network/manager.go` - Manager depends on transport lifecycle and audit adapters
- `internal/daemon/boot.go` - Daemon boot will consume config and transport readiness later
- `internal/api/contract/contract.go` - Status surfaces will eventually expose network diagnostics

### Related ADRs
- [ADR-001: Embedded NATS Server as Transport Layer](adrs/adr-001.md) - Governs transport ownership and credential handling
- [ADR-004: Network Manager as Boot-Phase Observer](adrs/adr-004.md) - Requires boot-phase infrastructure readiness
- [ADR-005: Runtime-Created Spaces with Explicit Session Opt-In](adrs/adr-005.md) - Confirms that only audit history is persisted

## Deliverables
- Config and home-path updates for the network runtime
- Embedded transport and audit storage primitives under `internal/network`
- Schema updates and storage helpers for `network_audit_log`
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for transport lifecycle and audit persistence **(REQUIRED)**

## Tests
- Unit tests:
- [ ] `[network]` defaults and validation reject invalid ports, payload limits, heartbeat values, and default-space names
- [ ] Transport setup fails safely when required runtime inputs are missing
- [ ] Audit writer normalizes records consistently for file and database sinks
- [ ] Home path resolution includes the new audit file path
- Integration tests:
- [ ] Embedded NATS can start, accept an in-process daemon connection, drain, and stop cleanly
- [ ] Audit events are persisted to `globaldb` and append-only file output without leaking broker credentials
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- AGH has a valid `[network]` config model and audit storage foundation
- Embedded transport lifecycle works without exposing broker credentials outside the daemon
