# Regression Suite: Bridge Adapters V1

**Date:** 2026-04-15
**Version:** 1.0
**Feature:** Provider-Scoped Bridge Adapters

---

## Suite Overview

This regression suite validates the complete Bridge V1 implementation across all layers: shared SDK, bridge domain, 8 provider extensions, daemon wiring, and API surfaces.

---

## Suite Tiers

### Smoke Suite (10-15 min)

**Purpose:** Quick gate — if any smoke test fails, stop and fix before proceeding.
**Frequency:** Every build, every commit, before any detailed testing.
**Command:** `make test` (unit tests only, which cover smoke-level scenarios)

| ID        | Test Case                                          | Covers                      |
| --------- | -------------------------------------------------- | --------------------------- |
| SMOKE-001 | Bridge SDK Runtime Boots Successfully              | SDK handshake, session init |
| SMOKE-002 | Bridge Instance CRUD Round-Trip                    | Registry persistence        |
| SMOKE-003 | Webhook Signature Verification                     | Ingress security gate       |
| SMOKE-004 | Inbound Message Ingestion Through Host API         | Provider→daemon flow        |
| SMOKE-005 | Delivery Pipeline Completes START to FINAL         | Delivery correctness        |
| SMOKE-006 | Error Classification Maps Provider Failures        | Error recovery mapping      |
| SMOKE-007 | Lifecycle State Machine Rejects Invalid Transition | State machine integrity     |
| SMOKE-008 | All Eight Providers Compile and Pass Unit Tests    | Build stability             |

**Pass Criteria:** All 8 smoke tests pass. Any failure blocks further testing.

---

### Targeted Regression Suite (30-45 min)

**Purpose:** Test areas impacted by specific changes.
**Frequency:** Per change, per PR.
**Command:** `go test -race ./internal/bridgesdk/... ./internal/bridges/... ./extensions/bridges/...`

#### Area A: SDK Infrastructure (if `internal/bridgesdk/` changed)

| Priority | Test Cases                                                                           |
| -------- | ------------------------------------------------------------------------------------ |
| P0       | TC-FUNC-013 (Error Classification), TC-PERF-001 (Dedup Bounds)                       |
| P1       | TC-FUNC-008 (Typed Interactions), TC-PERF-003 (Batching), TC-PERF-004 (Rate Limiter) |
| P1       | TC-SEC-008 (Rate Limit Attack), TC-SEC-009 (In-Flight Limits)                        |

#### Area B: Bridge Domain (if `internal/bridges/` changed)

| Priority | Test Cases                                                                                                |
| -------- | --------------------------------------------------------------------------------------------------------- |
| P0       | TC-FUNC-001 (Create), TC-FUNC-005 (State Machine), TC-FUNC-009 (Delivery Ordering), TC-FUNC-016 (Routing) |
| P1       | TC-FUNC-011 (Edit), TC-FUNC-012 (Delete), TC-FUNC-014 (Degradation), TC-FUNC-017 (Target Resolution)      |
| P1       | TC-INT-005 (Recovery), TC-INT-011 (Coalescing)                                                            |

#### Area C: Provider Extensions (if `extensions/bridges/<provider>/` changed)

| Priority | Test Cases                                                                              |
| -------- | --------------------------------------------------------------------------------------- |
| P0       | TC-SEC-001 or TC-SEC-002 (Signature for changed provider), TC-INT-002 (Webhook Ingress) |
| P1       | TC-INT-004 (Delivery E2E), TC-SEC-005 (DM Policy), TC-SEC-007 (Secret Isolation)        |
| P2       | TC-INT-012 (Conformance Matrix)                                                         |

#### Area D: API / CLI (if `internal/api/` or `internal/cli/` changed)

| Priority | Test Cases                                          |
| -------- | --------------------------------------------------- |
| P0       | TC-INT-007 (HTTP CRUD), TC-INT-009 (CLI Commands)   |
| P1       | TC-INT-008 (UDS Operations), TC-FUNC-004 (List/Get) |

#### Area E: Daemon Wiring (if `internal/daemon/bridges.go` changed)

| Priority | Test Cases                                                                           |
| -------- | ------------------------------------------------------------------------------------ |
| P0       | TC-INT-001 (Multi-Instance Launch), TC-INT-003 (Routing Isolation)                   |
| P1       | TC-INT-006 (Auth Cycle), TC-INT-010 (Managed Sync), TC-FUNC-018 (Source Distinction) |

---

### Full Regression Suite (2-3 hours)

**Purpose:** Comprehensive validation before releases or after large changes.
**Frequency:** Weekly, pre-release, after major refactors.
**Command:** `make verify && go test -race -tags integration ./...`

#### Execution Order

**Phase 1: Smoke (10 min)**
Run all SMOKE-001 through SMOKE-008. If any fail, STOP.

**Phase 2: P0 Critical (30 min)**

| Category       | Test Cases                                                                                                          |
| -------------- | ------------------------------------------------------------------------------------------------------------------- |
| Functional P0  | TC-FUNC-001, TC-FUNC-002, TC-FUNC-003, TC-FUNC-005, TC-FUNC-007, TC-FUNC-009, TC-FUNC-010, TC-FUNC-013, TC-FUNC-016 |
| Integration P0 | TC-INT-001, TC-INT-002, TC-INT-003, TC-INT-005, TC-INT-009                                                          |
| Security P0    | TC-SEC-001, TC-SEC-002, TC-SEC-003, TC-SEC-004, TC-SEC-005, TC-SEC-006                                              |
| Performance P0 | TC-PERF-001, TC-PERF-002                                                                                            |

**Phase 3: P1 High (45 min)**

| Category       | Test Cases                                                                                             |
| -------------- | ------------------------------------------------------------------------------------------------------ |
| Functional P1  | TC-FUNC-004, TC-FUNC-006, TC-FUNC-008, TC-FUNC-011, TC-FUNC-012, TC-FUNC-014, TC-FUNC-015, TC-FUNC-017 |
| Integration P1 | TC-INT-004, TC-INT-006, TC-INT-007, TC-INT-008, TC-INT-010                                             |
| Security P1    | TC-SEC-007, TC-SEC-008, TC-SEC-009                                                                     |
| Performance P1 | TC-PERF-003, TC-PERF-004, TC-PERF-005                                                                  |

**Phase 4: P2 Medium (20 min)**

| Category       | Test Cases                            |
| -------------- | ------------------------------------- |
| Functional P2  | TC-FUNC-018, TC-FUNC-019, TC-FUNC-020 |
| Integration P2 | TC-INT-011, TC-INT-012                |
| Security P2    | TC-SEC-010                            |
| Performance P2 | TC-PERF-006                           |

**Phase 5: Exploratory (30 min)**

- Unscripted testing of unusual provider configurations
- Multi-tenant scenarios with mixed DM policies
- Rapid instance create/delete cycles
- Concurrent delivery and ingestion under the same route

---

## Pass/Fail Criteria

### PASS

- All SMOKE tests pass
- All P0 tests pass
- 90%+ of P1 tests pass
- No Critical or High severity bugs open
- `make verify` passes (fmt + lint + test + build)
- No race conditions detected

### FAIL (Block Release)

- Any SMOKE test fails
- Any P0 test fails
- Critical bug discovered (data loss, security bypass, crash)
- Security vulnerability in webhook ingress
- Delivery ordering violation
- Cross-instance routing leakage

### CONDITIONAL PASS

- P1 failures with documented workarounds
- Known issues documented with fix plan
- Non-critical degradation reporting gaps
- Minor CLI output formatting issues

---

## Test Case Priority Summary

| Priority  | Count  | Categories                                            |
| --------- | ------ | ----------------------------------------------------- |
| P0        | 29     | 8 SMOKE + 8 TC-FUNC + 5 TC-INT + 6 TC-SEC + 2 TC-PERF |
| P1        | 19     | 8 TC-FUNC + 5 TC-INT + 3 TC-SEC + 3 TC-PERF           |
| P2        | 8      | 4 TC-FUNC + 2 TC-INT + 1 TC-SEC + 1 TC-PERF           |
| **Total** | **56** |                                                       |

---

## Existing Automated Coverage

The codebase already has extensive automated test coverage that maps to these test cases:

| Test Case Area       | Automated By                                                        | Location                                                            |
| -------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------- |
| SDK Runtime Flow     | `TestRuntimeServeInitializeDeliverHealthShutdownAndSync`            | `internal/bridgesdk/runtime_flow_test.go`                           |
| Error Classification | `TestClassifyErrorMapsRepresentativeProviderFailures`               | `internal/bridgesdk/errors_test.go`                                 |
| Dedup Cache          | `TestDedupCacheSuppressesDuplicatesWithinTTLAndReleasesAfterExpiry` | `internal/bridgesdk/dedup_test.go`                                  |
| Batching             | `TestInboundBatcherCoalescesShortBurstAndPreservesOrdering`         | `internal/bridgesdk/batching_test.go`                               |
| Webhook Guards       | `TestWebhookHandlerRejectsUnsupportedMethodBeforeHandler`           | `internal/bridgesdk/webhook_test.go`                                |
| Instance Cache       | `TestInstanceCacheSyncPreservesBoundSecrets`                        | `internal/bridgesdk/cache_test.go`                                  |
| Registry CRUD        | `TestBridgeHandlersCreateListGetAndUpdate`                          | `internal/api/core/bridges_test.go`                                 |
| Lifecycle            | `TestBridgeRuntimeTransition`                                       | `internal/daemon/bridges_test.go`                                   |
| Delivery Broker      | `TestBridgeDeliveryNotifierProjectsEventsAndForwardsLifecycle`      | `internal/extension/bridge_delivery_notifier_test.go`               |
| Delivery Ordering    | `TestBridgeDeliveryIntegrationShouldHandleDeliveryScenarios`        | `internal/extension/bridge_delivery_integration_test.go`            |
| Conformance Matrix   | `TestBuildConformanceMatrixAggregatesTargetsPerProvider`            | `internal/extensiontest/bridge_conformance_matrix_test.go`          |
| Conformance Harness  | `TestHarnessIntegrationTelegramReferenceConformance`                | `internal/extensiontest/bridge_adapter_harness_integration_test.go` |
| HTTP API             | `TestHTTPBridgeCreateReturnsPersistedPayload`                       | `internal/api/httpapi/bridges_integration_test.go`                  |
| UDS API              | `TestCreateBridgeHandlerReturnsPersistedPayload`                    | `internal/api/udsapi/bridges_integration_test.go`                   |
| CLI                  | `TestBridgeListRendersScopePlatformAndStatusInHumanOutput`          | `internal/cli/bridge_test.go`                                       |
| Health Metrics       | `TestHealthIncludesBridgeStatusCountsAndRouteSummary`               | `internal/observe/bridges_test.go`                                  |
| Provider Tests       | `provider_test.go` in each `extensions/bridges/<provider>/`         | Per-provider unit tests                                             |

---

## Maintenance

### Monthly Review

- Remove test cases for deprecated features
- Update test cases for changed APIs or contracts
- Add regression cases for bugs found in production
- Review priority assignments based on incident history

### After Each Release

- Update test data and expected values
- Fix broken tests from API changes
- Add regression cases for bugs discovered during release testing
- Archive execution reports

---

## References

- Test Plan: `qa/test-plans/bridge-adapters-test-plan.md`
- Test Cases: `qa/test-cases/TC-*.md` and `qa/test-cases/SMOKE-*.md`
- TechSpec: `.compozy/tasks/bridge-adapters/_techspec.md`
- Conformance Harness: `internal/extensiontest/bridge_adapter_harness.go`
