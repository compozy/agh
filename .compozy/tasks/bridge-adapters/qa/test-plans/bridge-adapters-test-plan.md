# Test Plan: Provider-Scoped Bridge Adapters (Bridge V1)

**Date:** 2026-04-15
**Version:** 1.0
**Feature:** Bridge V1 — Eight Provider-Scoped Bridge Adapters
**TechSpec:** `.compozy/tasks/bridge-adapters/_techspec.md`

---

## Executive Summary

This test plan covers the complete Bridge V1 implementation: eight provider-scoped bridge adapters (Slack, Discord, Telegram, Teams, WhatsApp, Google Chat, GitHub, Linear), the shared `internal/bridgesdk` provider SDK, the daemon-owned bridge runtime (registry, delivery broker, routing), and all API surfaces (HTTP, UDS, CLI).

### Objectives

- Validate that all eight providers correctly ingest inbound platform events and deliver outbound messages
- Verify the provider-scoped runtime model (one subprocess per provider, many bridge instances multiplexed)
- Confirm webhook ingress hardening (signature verification, rate limiting, body size limits, in-flight limits)
- Validate delivery pipeline correctness (progressive streaming, edit/delete, recovery/resume)
- Verify bridge instance lifecycle state machine transitions
- Confirm error classification and structured degradation reporting
- Validate DM policy enforcement (open, allowlist, pairing)
- Verify multi-instance and multi-tenant provider scenarios
- Confirm adapter-local dedup and inbound batching behavior
- Validate API contract correctness across HTTP, UDS, and CLI surfaces

### Key Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Webhook signature bypass allows unauthorized ingestion | Low | Critical | TC-SEC-001 through TC-SEC-008 cover all provider signature schemes |
| Delivery ordering violation causes garbled output | Medium | High | TC-FUNC-009 through TC-FUNC-012 validate START→DELTA→FINAL sequencing |
| Provider subprocess crash loses in-flight deliveries | Medium | High | TC-INT-005 validates recovery/resume with delivery snapshots |
| Rate limit storms degrade daemon performance | Medium | Medium | TC-PERF-001 through TC-PERF-003 validate rate limiting and backoff |
| Multi-instance config collision causes cross-tenant routing | Low | Critical | TC-INT-003, TC-INT-004 validate instance isolation |
| Dedup cache memory growth under sustained load | Low | Medium | TC-PERF-004 validates TTL eviction and max-size bounds |
| State machine allows invalid transitions | Low | High | TC-FUNC-005 validates all valid/invalid transition pairs |

---

## Scope

### In-Scope

- **Shared SDK** (`internal/bridgesdk`): Runtime, peer, webhook guards, dedup, batching, error classification, host API client, instance cache
- **Bridge Domain** (`internal/bridges`): Registry, delivery broker, routing, target resolution, lifecycle state machine, managed sync, types/validation
- **Provider Extensions** (`extensions/bridges/*`): All 8 providers — webhook ingestion, message mapping, delivery execution, signature verification, instance config resolution
- **Daemon Wiring** (`internal/daemon`): Bridge runtime composition, secret binding, provider enumeration, lifecycle management
- **Extension Manager** (`internal/extension`): Provider-scoped runtime handshake, Host API bridge methods, delivery notifier
- **Subprocess Protocol** (`internal/subprocess`): Initialize/shutdown handshake with bridge runtime payload
- **API Surfaces**: HTTP bridge endpoints, UDS bridge handlers, CLI bridge commands
- **Observability** (`internal/observe`): Bridge health metrics, status counts, delivery backlog tracking

### Out-of-Scope

- Modal lifecycle orchestration
- Ephemeral delivery portability
- Typing indicators
- Approval UI flows
- Credential pool rotation
- Dual-lane reasoning/answer rendering
- Web UI visual rendering of bridge configuration
- External platform API integration testing (tests use mocks/stubs)

---

## Test Strategy

### Approach

Testing follows a layered strategy matching the architecture:

1. **Unit Tests** — Validate individual components in isolation (SDK helpers, domain types, error classification, state machine)
2. **Integration Tests** — Validate component interactions (provider→daemon flows, delivery pipeline, API→registry→store)
3. **Conformance Tests** — Validate all providers meet the bridge v1 contract (conformance matrix harness)
4. **Security Tests** — Validate ingress hardening, signature verification, secret isolation, DM policy
5. **Performance Tests** — Validate rate limiting, batching efficiency, dedup cache bounds, delivery throughput

### Test Levels

| Level | Scope | Tool | Build Tag |
|-------|-------|------|-----------|
| Unit | Package-internal | `go test -race` | (none) |
| Integration | Cross-package, real SQLite | `go test -race -tags integration` | `integration` |
| Conformance | Provider subprocess harness | `go test -race -tags integration` | `integration` |
| Security | Ingress validation, policy enforcement | `go test -race` | (none) |
| Performance | Load characteristics, resource bounds | `go test -race -bench` | (none) |

### Verification Gate

All tests must pass `make verify` (fmt → lint → test → build) with zero warnings and zero errors.

---

## Environment Requirements

- **OS:** macOS (darwin) or Linux
- **Go:** Version matching `go.mod` toolchain
- **Build:** `make build` produces single binary
- **SQLite:** Via `t.TempDir()` for test isolation
- **Network:** Localhost only (webhook servers bind to `127.0.0.1`)
- **External Dependencies:** None — all platform APIs are mocked/stubbed in tests

---

## Entry Criteria

- [ ] All 8 provider extensions compile without errors
- [ ] `internal/bridgesdk` package compiles and passes existing unit tests
- [ ] `internal/bridges` package compiles and passes existing unit tests
- [ ] `make build` succeeds
- [ ] `make lint` reports zero issues
- [ ] Test data and fixtures are available in test files

## Exit Criteria

- [ ] `make verify` passes (fmt + lint + test + build)
- [ ] All P0 test cases pass
- [ ] 90%+ of P1 test cases pass
- [ ] No Critical or High severity bugs remain open
- [ ] Conformance matrix validates all 8 providers
- [ ] 80%+ code coverage per package maintained
- [ ] No race conditions detected (`-race` flag)

---

## Test Case Summary

| Category | Prefix | Count | Priority Breakdown |
|----------|--------|-------|--------------------|
| Functional | TC-FUNC | 20 | 8 P0, 8 P1, 4 P2 |
| Integration | TC-INT | 12 | 5 P0, 5 P1, 2 P2 |
| Security | TC-SEC | 10 | 6 P0, 3 P1, 1 P2 |
| Performance | TC-PERF | 6 | 2 P0, 3 P1, 1 P2 |
| Smoke | SMOKE | 8 | 8 P0 |
| **Total** | | **56** | **29 P0, 19 P1, 8 P2** |

---

## Timeline and Deliverables

| Phase | Deliverable | Status |
|-------|------------|--------|
| Planning | This test plan | Complete |
| Test Case Design | TC-FUNC, TC-INT, TC-SEC, TC-PERF, SMOKE cases | Complete |
| Regression Suite | Tiered regression suite document | Complete |
| Execution | Run via `qa-execution` or `make verify` | Pending |
| Reporting | Verification report with pass/fail matrix | Pending |

---

## References

- TechSpec: `.compozy/tasks/bridge-adapters/_techspec.md`
- ADR-001: Provider-Scoped Bridge SDK and Runtime Model
- ADR-002: Hardened Webhook + REST Provider Communication
- ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity
- Conformance Harness: `internal/extensiontest/bridge_adapter_harness.go`
- Conformance Matrix: `internal/extensiontest/bridge_conformance_matrix.go`
