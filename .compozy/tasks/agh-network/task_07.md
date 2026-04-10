---
status: pending
title: Network manager and daemon boot integration
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
  - task_04
  - task_06
---

# Task 07: Network manager and daemon boot integration

## Overview

Wire the network runtime into the daemon as a boot-phase service with explicit lifecycle ownership. This task introduces the network manager, late-bound session integration, and daemon diagnostics while preserving the existing package-boundary rules.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create a network manager that owns transport, routing, presence, delivery, and session-facing lifecycle callbacks
- MUST add a `bootNetwork` phase between `bootRuntime` and `bootHooks` and keep boot/shutdown ordering aligned with the ADR
- MUST integrate with `session.Manager` through post-construction setters or equivalent late-bound interfaces instead of constructor-time package coupling
- MUST expose daemon diagnostics for network status and listener information without publishing credentials
</requirements>

## Subtasks
- [ ] 7.1 Create the top-level network manager and runtime interfaces it implements
- [ ] 7.2 Add daemon boot, shutdown, and diagnostic wiring for the network service
- [ ] 7.3 Late-bind session lifecycle callbacks for join, leave, turn-end, and inbound delivery integration
- [ ] 7.4 Add daemon-level tests for startup, shutdown, and optional disabled-network mode

## Implementation Details

This task is the composition-root layer for the feature. Keep `internal/network` unaware of daemon internals, and keep the daemon responsible for passing dependencies downward.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Integration points, daemon boot sequence, and manager design sections
- `internal/network/manager.go` - New orchestrator for transport, router, presence, and delivery
- `internal/daemon/boot.go` - Add `bootNetwork` and shutdown sequencing
- `internal/daemon/daemon.go` - Extend runtime deps and late-bound integrations
- `internal/daemon/info.go` - Surface network diagnostics in daemon info/status
- `internal/daemon/hooks_bridge.go` - Reuse notifier-style integration patterns where appropriate

### Dependent Files
- `internal/api/udsapi/routes.go` - Network APIs will depend on booted runtime services
- `internal/cli/network.go` - CLI commands will call the daemon through the new network surface
- `internal/session/manager.go` - Session manager will receive late-bound callbacks but must not import `internal/network`

### Related ADRs
- [ADR-001: Embedded NATS Server as Transport Layer](adrs/adr-001.md) - Daemon owns the embedded broker lifecycle
- [ADR-004: Network Manager as Boot-Phase Observer](adrs/adr-004.md) - Governs boot sequencing and late-bound integration
- [ADR-005: Runtime-Created Spaces with Explicit Session Opt-In](adrs/adr-005.md) - Session participation is driven from stored opt-in metadata

## Deliverables
- New daemon-owned network manager integration
- Boot and shutdown sequencing updates for the optional network runtime
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for daemon boot and network lifecycle **(REQUIRED)**

## Tests
- Unit tests:
- [ ] Disabled network mode leaves the daemon operational without booting transport or manager services
- [ ] Runtime dependency wiring rejects incomplete network manager setup cleanly
- [ ] Daemon info/status surfaces network diagnostics without exposing credentials
- [ ] Session callbacks are late-bound without introducing package import cycles
- Integration tests:
- [ ] Full daemon startup and shutdown with network enabled drains transport and delivery workers cleanly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The network runtime boots and stops as a first-class daemon service
- Session integration is wired through composition-root interfaces instead of package coupling
