---
status: pending
title: Operational hardening and reliability sweep
type: backend
complexity: high
dependencies:
  - task_05
  - task_06
  - task_07
  - task_08
  - task_09
---

# Task 10: Operational hardening and reliability sweep

## Overview

Close the remaining runtime risk items after the core feature is in place by hardening reconnect behavior, queue resilience, diagnostics, and full-stack reliability. This task is the final polish pass that proves the integrated feature behaves correctly under realistic failure and concurrency conditions.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST close the known risk items from the tech spec that remain after initial feature implementation, especially reconnect, queue pressure, and busy-session delivery behavior
- MUST ensure structured logs, metrics, and diagnostics remain accurate across reconnects, shutdowns, retries, and overflow conditions
- MUST validate the final end-to-end user and agent workflows through real integration paths rather than isolated mocks alone
- MUST not introduce workaround logic that bypasses the guardrails or provenance model established by earlier tasks
</requirements>

## Subtasks
- [ ] 10.1 Harden reconnect, re-greet, queue overflow, and shutdown-drain behavior against real runtime edge cases
- [ ] 10.2 Tighten observability so logs, metrics, and status surfaces reflect degraded and recovered states correctly
- [ ] 10.3 Exercise the full agent workflow from CLI control plane to inbound session delivery and retry behavior
- [ ] 10.4 Fix any integration defects uncovered by the hardening pass without weakening tests or guardrails

## Implementation Details

Use this task to converge the fully integrated feature, not to create alternate code paths. The goal is to remove ambiguity and prove the final runtime under real daemon/session/network flows.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Known risks, testing approach, and monitoring sections
- `internal/network/transport.go` - Reconnect and drain behavior need final hardening
- `internal/network/delivery.go` - Queue pressure and delivery sequencing need real stress coverage
- `internal/network/manager.go` - Manager lifecycle and recovery behavior are finalized here
- `internal/observe/observer.go` - Metrics and health reporting are verified and tightened here
- `internal/daemon/daemon_integration_test.go` - Full-stack daemon behavior should be exercised here
- `internal/cli/cli_integration_test.go` - Agent-facing and user-facing workflows should be proven here

### Dependent Files
- `internal/api/udsapi/udsapi_integration_test.go` - Transport and handler integration coverage will likely expand here
- `internal/session/manager_integration_test.go` - Busy-session and resume/rejoin flows should be validated here

### Related ADRs
- [ADR-001: Embedded NATS Server as Transport Layer](adrs/adr-001.md) - Reliability depends on correct embedded transport lifecycle handling
- [ADR-003: CLI + Bundled Skill for Agent Network Communication](adrs/adr-003.md) - Final workflows must validate the intended CLI-plus-skill model
- [ADR-004: Network Manager as Boot-Phase Observer](adrs/adr-004.md) - Recovery and shutdown correctness depend on manager lifecycle ownership

## Deliverables
- Hardened reconnect, queue, and shutdown behavior across the integrated runtime
- Final observability and diagnostics polish for network operation and recovery
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for real end-to-end network workflows and failure recovery **(REQUIRED)**

## Tests
- Unit tests:
- [ ] Reconnect and re-greet paths preserve peer visibility and delivery readiness
- [ ] Queue overflow, retry, and shutdown-drain logic preserve the configured invariants
- [ ] Observability surfaces reflect degraded and recovered network states correctly
- [ ] Busy-session delivery semantics remain FIFO and provenance-safe under stress
- Integration tests:
- [ ] Full CLI -> UDS -> daemon -> network -> session flows succeed for direct, broadcast, retry, resume, and reconnect scenarios
- [ ] Failure injection cases surface the correct diagnostics without bypassing runtime guardrails
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The integrated AGH Network runtime behaves reliably under reconnect, queue pressure, and busy-session conditions
- The final feature can be verified end to end without hidden manual steps or degraded safety controls
