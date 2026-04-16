---
status: completed
title: "Implement the reconcile driver runtime"
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Implement the reconcile driver runtime

## Overview

Create the named control-plane scheduler that drives boot rebuilds and post-commit reconcile in a topology-aware way. This task turns the spec's reconcile semantics into one explicit runtime component with bounded concurrency, timeout propagation, degraded-circuit behavior, and shutdown ownership.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement the `ReconcileDriver` contract with single-flight execution per kind, one queued rerun at most, bounded coalescing, and topology-aware dependency ordering.
2. MUST propagate per-kind deadlines into `Projector.Build` and `Projector.Apply`, and surface projector failures through health state, logs, and metrics rather than through already-committed writes.
3. MUST support reverse-dependency fan-out after writes, degraded-circuit behavior after repeated failures, and `Close(ctx)` shutdown semantics that do not leak goroutines.
4. MUST preserve the TechSpec distinction between build-time ordering (`DependsOn()`) and ownership fan-out so the driver does not create dependency cycles for bundle-generated records.
</requirements>

## Subtasks

- [x] 3.1 Implement the named reconcile driver and projector registration topology
- [x] 3.2 Add single-flight, coalescing, timeout, and degraded-circuit behavior per resource kind
- [x] 3.3 Wire boot-time `RunBoot()` ordering and post-commit reverse-dependency scheduling
- [x] 3.4 Add contract coverage for retries, shutdown, and topology correctness

## Implementation Details

Follow the TechSpec sections "Core Interfaces", "Integration Points", "Development Sequencing", and "Monitoring and Observability". This task should introduce the shared reconcile runtime only; it should not yet migrate any family-specific projector logic beyond the driver contract and test doubles.

### Relevant Files

- `internal/resources/` — New reconcile driver, projector registry, reverse dependency index, and scheduler tests
- `internal/daemon/boot.go` — Boot sequencing must call `RunBoot()` in dependency order for migrated kinds
- `internal/daemon/daemon.go` — Daemon lifecycle must own reconcile driver startup and shutdown
- `internal/daemon/extensions.go` — Post-commit triggers from extension-driven writes later feed the shared driver

### Dependent Files

- `internal/hooks/dispatch.go` — Hook-binding cutover later relies on atomic projector apply semantics
- `internal/automation/manager.go` — Automation migration later depends on full-snapshot reconcile triggers
- `internal/bridges/registry.go` — Bridge migration later depends on external-state projector timeout and degraded behavior

### Related ADRs

- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires explicit reconcile evidence for every cutover
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Makes full-snapshot reconcile the correctness path
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Defines the scheduler semantics this task implements
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Keeps raw dependency bags internal to the driver

## Deliverables

- A named reconcile driver with topology registration, single-flight scheduling, and shutdown ownership
- Boot-time and post-commit scheduling semantics aligned to the TechSpec
- Health, metric, and degraded-circuit hooks for projector failure handling
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for boot ordering, write-storm coalescing, and shutdown behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] concurrent triggers for the same kind produce at most one in-flight pass plus one queued rerun
  - [x] projector contexts inherit the configured timeout and cancel when the deadline elapses
  - [x] repeated projector failures open a degraded circuit and suppress busy-loop reruns until backoff or a new write arrives
  - [x] reverse dependencies are scheduled after the written kind without conflating `DependsOn()` with ownership fan-out
- Integration tests:
  - [x] `RunBoot()` rebuilds registered kinds in topological order and refuses an invalid dependency graph
  - [x] a write storm against one kind coalesces work within the bounded window instead of growing an unbounded queue
  - [x] `Close(ctx)` stops new triggers and drains or cancels in-flight work within the caller deadline
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- The daemon has one explicit reconcile scheduler instead of ad hoc post-write callbacks
- Boot rebuilds, post-commit fan-out, and projector failure handling are deterministic and observable
