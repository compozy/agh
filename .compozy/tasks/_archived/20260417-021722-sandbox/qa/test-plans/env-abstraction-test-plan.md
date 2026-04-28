# Test Plan: Execution Sandbox Abstraction + Daytona Provider

**Version:** 1.0
**Date:** 2026-04-16
**Feature:** Execution Sandbox Abstraction Layer (tasks 01-08)
**Branch:** ext-refac
**Author:** QA (automated)

---

## Executive Summary

This test plan covers the Execution Sandbox Abstraction feature, which decouples AGH's ACP runtime from the local OS by introducing `Provider`, `Launcher`, `ToolHost`, and `Handle` interfaces. The feature ships a local provider (preserving current behavior) and a Daytona remote provider (SSH transport, tar-over-SSH sync, snapshot-aware sandbox lifecycle). It also integrates sandbox lifecycle into session management, daemon boot reconciliation, config profiles, workspace resolution, and extension hooks.

**Key Objectives:**
- Verify zero behavior regression for existing local sessions after ACP extraction (tasks 02-04)
- Verify sandbox profile configuration, validation, and merge (task 01)
- Verify session sandbox lifecycle: prepare -> sync-to -> launch -> sync-from -> destroy (task 04)
- Verify daemon restart reconciliation cleans up orphaned remote sandboxes (task 07)
- Verify environment extension hooks fire at correct lifecycle points with correct payloads (task 08)
- Verify Daytona provider SSH transport, tar sync safety, env var allowlist, and network policy (tasks 05-06)
- Verify security boundaries: no secret leakage, tar path traversal rejection, permission enforcement

**Key Risks:**
- ACP Launcher/ToolHost extraction regresses existing agent communication (HIGH impact, LOW likelihood)
- Tar extraction path traversal bypasses safety checks (HIGH impact, LOW likelihood)
- Environment variable leakage to remote sandbox (HIGH impact, LOW likelihood)
- SSH token expiry during long sessions causes agent disconnect (MEDIUM impact, MEDIUM likelihood)
- Large workspace sync causes excessive session start latency (MEDIUM impact, MEDIUM likelihood)
- Daemon crash leaves orphaned billable Daytona sandboxes (MEDIUM impact, MEDIUM likelihood)

---

## Scope

### In-Scope

| Area | Components | Tasks |
|------|-----------|-------|
| Core types & interfaces | `internal/sandbox/types.go`, `registry.go` | 01 |
| Config profiles | `internal/config/config.go`, `merge.go` | 01 |
| Workspace resolution | `internal/workspace/`, `internal/store/globaldb/` | 01 |
| ACP extraction | `internal/acp/launcher.go`, `tool_host.go` | 02 |
| Local provider | `internal/sandbox/local/` | 03 |
| Session integration | `internal/session/sandbox.go`, `manager_start.go` | 04 |
| SSH validation | `internal/sandbox/daytona/ssh_validation_test.go` | 05 |
| Daytona provider | `internal/sandbox/daytona/` (19 files) | 06 |
| Daemon reconciliation | `internal/daemon/boot.go`, `sandbox_reconcile.go` | 07 |
| Extension hooks + Host API | `internal/hooks/`, `internal/extension/` | 08 |
| API contracts | `internal/api/contract/`, `internal/api/core/` | 01, 04 |
| CLI flags | `internal/cli/workspace.go` | 01 |
| DB schema | sessions + workspaces sandbox columns | 01, 04 |

### Out-of-Scope

- Web UI rendering of sandbox info (no frontend changes in this feature)
- E2B or other future providers (reserved but not implemented)
- Turn-level sync (`SyncModeTurnBidirectional` reserved)
- Automated snapshot creation/mutation
- Process manager / PTY streaming (future follow-up)
- Extensibility-parity runtime internals (separate feature)

---

## Test Strategy

### Automated Testing

| Layer | Tool | Coverage Target | Gate |
|-------|------|----------------|------|
| Unit tests | `go test -race` | >= 80% per package | `make test` |
| Integration tests | `go test -tags integration -race` | Critical paths | `make test-integration` |
| Lint | `golangci-lint` | Zero issues | `make lint` |
| Build | `go build` | Clean compile | `make build` |
| Full gate | All above | All pass | `make verify` |

### Manual / Semi-Automated Testing

| Area | Method | When |
|------|--------|------|
| Daytona E2E | Integration test with `DAYTONA_API_KEY` | Before merge, requires API key |
| SSH non-PTY validation | Spike test against live Daytona | Before merge, requires API key |
| CLI workspace flags | Manual `agh workspace add --sandbox` | After local build |
| Session list/info API | Manual `agh session list`, `agh session info` | After local build |

---

## Environment Requirements

| Environment | Purpose | Requirements |
|-------------|---------|-------------|
| Local dev | Unit + integration tests | Go 1.24+, macOS/Linux, SQLite |
| Daytona staging | Remote provider E2E | `DAYTONA_API_KEY`, network access to Daytona API |
| CI | Automated gate | `make verify`, no Daytona key (skip tagged tests) |

---

## Entry Criteria

- [x] All 6 completed tasks (01-04, 07-08) have code merged to `ext-refac`
- [x] `make verify` passes with zero warnings/errors
- [x] All existing ACP tests pass unmodified (regression gate)
- [x] Go module dependencies resolved (`go.sum` updated)
- [ ] Tasks 05-06 (Daytona provider) code reviewed and ready

---

## Exit Criteria

- All P0 test cases pass
- All P1 test cases pass (90%+ threshold)
- No Critical or High severity bugs open
- `make verify` passes
- Zero lint warnings
- Test coverage >= 80% per package
- Daytona E2E integration test passes (when API key available)

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| ACP extraction regresses agent communication | Low | High | Existing ACP test suite is the regression gate; TC-REG-001 through TC-REG-004 |
| Tar path traversal escapes destination | Low | High | TC-SEC-001 with malicious archive payloads |
| Env var leakage (`DAYTONA_API_KEY`) to sandbox | Low | High | TC-SEC-002 validates allowlist enforcement |
| SSH token expires mid-session | Medium | Medium | TC-FUNC-017 validates 50% refresh and retry |
| Large workspace sync causes latency | Medium | Medium | TC-PERF-001 measures sync duration |
| Daemon crash orphans billable sandboxes | Medium | Medium | TC-FUNC-020 through TC-FUNC-024 validate reconciliation |
| Concurrent sessions cause data corruption | Low | High | TC-INT-005 validates last-write-wins semantics |
| Config merge drops sandbox profile fields | Low | Medium | TC-FUNC-005 validates overlay merge |
| Hook deny patch ignored by session manager | Low | High | TC-FUNC-026 through TC-FUNC-028 validate deny semantics |

---

## Test Case Summary

| Category | Count | Priority Distribution |
|----------|-------|-----------------------|
| Smoke (SMOKE-*) | 8 | All P0 |
| Functional (TC-FUNC-*) | 30 | 12 P0, 10 P1, 8 P2 |
| Integration (TC-INT-*) | 8 | 4 P0, 3 P1, 1 P2 |
| Regression (TC-REG-*) | 6 | 4 P0, 2 P1 |
| Security (TC-SEC-*) | 5 | 4 P0, 1 P1 |
| Performance (TC-PERF-*) | 3 | 1 P1, 2 P2 |
| **Total** | **60** | **24 P0, 16 P1, 11 P2, 0 P3** |

---

## Timeline

| Phase | Duration | Deliverables |
|-------|----------|-------------|
| Test plan creation | Complete | This document |
| Test case generation | Complete | 60 test cases in `qa/test-cases/` |
| Smoke suite execution | 15 min | Automated via `make verify` |
| Full regression execution | 30-60 min | `make test` + manual CLI checks |
| Daytona E2E (gated) | 15 min | `make test-integration` with API key |
| Bug reporting | As found | `qa/issues/` |
| Final verification report | After execution | `qa/verification-report.md` |
