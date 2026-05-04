# Network Redesign Round 2 Test Plan

**Created:** 2026-04-23
**Feature:** AGH Network Redesign
**Round:** 2
**Branch:** `feat/network-slack-redesign`
**Baseline reference:** `971c51cf` (`fix: address network redesign QA review findings round 1`)

## Executive Summary

This plan covers a deeper second-pass QA sweep for the network redesign across the backend runtime, persistence, transports, Slack bridge, CLI, and React operator UI. The primary objective is to prove that the network workspace is correct from the wire protocol and durable store through to operator-facing API and UI workflows, not only that isolated helpers pass unit tests.

The highest-risk areas are channel aggregation correctness, cursor-scoped timeline queries, Slack webhook security and deduplication, create-channel rollback behavior, local-session selection for sending, and UI state synchronization across URL search params, TanStack Query, and local persistence.

## Scope

### In Scope

- API Core handlers in `internal/api/core/network.go` and `internal/api/core/network_details.go`.
- Shared API contracts and UDS/HTTP parity for network status, peers, channels, messages, send, inbox, and create-channel flows.
- Global DB persistence for `network_channels`, `network_timeline_log`, and `network_audit_log`.
- Session layer peer capability projection and channel-bound session creation.
- Network runtime peer, channel, send, receive, audit, delivery, and status behavior.
- Slack bridge provider initialization, webhook request handling, signature verification, deduplication, batching, DM policy, and delivery acknowledgements.
- CLI commands `agh network status`, `agh network peers`, `agh network channels`, `agh network send`, and `agh network inbox`.
- Frontend network workspace shell, room filtering, URL search synchronization, create channel dialog, composer, details panel, loading/empty/error states, responsive behavior, and settings network page.

### Out Of Scope

- Live Slack workspace validation with real credentials.
- NATS interoperability with another daemon outside local integration fixtures.
- Backward compatibility with pre-alpha persisted schemas or old state.
- `.old_project/` reference implementation.
- Marketing/docs site visual validation except where docs are used as CLI reference evidence.

## Test Strategy

Testing uses a layered approach:

1. Static source analysis to derive risks and test cases from the implementation surface.
2. Baseline gate execution using the repository contract, with `make verify` as the canonical gate.
3. Focused backend unit and integration tests for API/store/session/runtime/Slack/CLI gaps.
4. Frontend Vitest and existing Playwright/e2e coverage review for the network workspace and settings page.
5. Browser validation where the dev server and local runtime can be started without external credentials.
6. Final full verification with `make verify`, followed by `go vet ./...`.

P0 and P1 test cases are execution candidates for Phase 2. P2 and P3 cases document extended regression coverage and release-hardening scenarios.

## Environment Requirements

- OS: Linux container workspace at `/tmp/agh-network-slack-redesign`.
- Go toolchain compatible with the repository `go.mod`.
- Bun workspace dependencies for `web/` tests and builds.
- SQLite available through Go SQLite driver.
- Local ports available for test HTTP servers and Vite dev server if browser validation runs.
- No real Slack credentials required; Slack API interactions use local test servers/fakes.
- Browser validation uses the repository-supported browser tooling when available.

## Entry Criteria

- Worktree has no unexpected user edits that conflict with QA changes.
- Round 1 fix commit `971c51cf` is reachable in git history.
- QA output directories exist under `.compozy/tasks/unified-capabilities/qa/`.
- Repository contract discovery completes.
- Baseline `make verify` is run before scenario execution.

## Exit Criteria

- All P0 test cases are executed or explicitly blocked with evidence.
- All P1 test cases are executed through automated tests, source-backed coverage inspection, or runtime/browser flow evidence.
- Every new bug found during planning or execution has a `BUG-XXX.md` report.
- Every filed bug is fixed or explicitly impossible to fix in the current environment with concrete blocker evidence.
- New or updated automated tests cover every fix and meaningful coverage gap.
- Final `make verify` passes with zero warnings and zero errors.
- Final `go vet ./...` passes.
- Verification report is written to `.compozy/tasks/unified-capabilities/qa/verification-report-round2.md`.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Channel summaries drift from detailed channel payloads because API aggregation pulls sessions, peers, metadata, and messages from different sources. | Medium | High | Execute TC-FUNC-004, TC-FUNC-005, TC-PERF-001, and add targeted API tests where coverage is missing. |
| Cursor queries leak messages across channel or peer boundaries. | Medium | High | Execute TC-FUNC-006, TC-FUNC-007, TC-SEC-003 against `global_db_network_messages.go` and handler mappings. |
| Slack webhook security accepts unsigned, stale, oversized, duplicate, or disallowed-DM payloads. | Medium | Critical | Execute TC-SEC-001, TC-SEC-002, TC-INT-103, and TC-INT-104 with local signed requests and invalid variants. |
| Create-channel partially creates sessions then fails to persist metadata or read details. | Medium | High | Execute TC-FUNC-008 and TC-INT-101 with rollback assertions. |
| Network composer chooses the wrong local session for a direct or channel send. | Medium | High | Execute TC-UI-004 and TC-FUNC-009 against UI hook state and API request payloads. |
| Peer capability catalog is exposed stale or duplicated between brief and rich detail payloads. | Low | Medium | Execute TC-FUNC-002, TC-FUNC-003, and TC-UI-005. |
| Frontend URL synchronization loops or overwrites user selection while data refetches. | Medium | Medium | Execute TC-UI-002 with deep links, search updates, and back/forward behavior. |
| Network settings allow invalid values to be saved or leave stale validation errors. | Medium | High | Execute TC-UI-006 and backend settings validation coverage. |
| Runtime delivery goroutines or heartbeats leak during shutdown. | Low | High | Execute TC-INT-102 under race tests and inspect shutdown paths. |
| CLI output omits critical network fields or diverges from HTTP/UDS contracts. | Medium | Medium | Execute TC-INT-006 and SMOKE-001. |

## Deliverables

- Test plan: `.compozy/tasks/unified-capabilities/qa/test-plans/network-redesign-round2-test-plan.md`
- Test cases: `.compozy/tasks/unified-capabilities/qa/test-cases/`
- Bug reports: `.compozy/tasks/unified-capabilities/qa/issues/`
- Screenshots/evidence: `.compozy/tasks/unified-capabilities/qa/screenshots/`
- Verification report: `.compozy/tasks/unified-capabilities/qa/verification-report-round2.md`

## Execution Order

1. Run smoke tests: SMOKE-001 through SMOKE-003.
2. Execute all P0 functional, integration, and security cases.
3. Execute P1 API/store/Slack/CLI/UI/performance cases.
4. Review P2/P3 cases for cheap automated coverage opportunities.
5. Fix discovered defects and add tests.
6. Run final `make verify`.
7. Run final `go vet ./...`.
