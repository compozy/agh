# Autonomy MVP QA Verification Report

## Result

- **Status:** PASS after root-cause fixes.
- **Date:** 2026-04-26.
- **QA workflow:** `qa-execution` activated with `qa-output-path=.compozy/tasks/autonomous`.
- **Source matrix:** `.compozy/tasks/autonomous/qa/test-plans/` and `.compozy/tasks/autonomous/qa/test-cases/TC-AUTO-001.md` through `TC-AUTO-018.md`.
- **Primary evidence root:** `.compozy/tasks/autonomous/qa/`.

## Executive Summary

Task 18 executed the autonomy MVP QA pass across backend/store, generated contracts, daemon lifecycle, CLI/UDS, hooks, coordination channels, scheduler, spawn, coordinator bootstrap, Tasks UI, site docs, and post-MVP boundary scans.

Three regressions were found and fixed at root cause:

- `BUG-001`: Playwright daemon-served specs raced workspace onboarding and one handoff assertion skipped the real session-create dialog.
- `BUG-002`: The deterministic ACP mock exact-matched raw prompts after Task 04 situation-context augmentation, so fixture user turns no longer matched.
- `BUG-003`: The Tasks browser E2E expected only empty/fallback Agents states, while the current manual-first publish flow correctly renders an active task-bound run.

After the last fix, the full required gates passed:

- `make verify`
- `make codegen-check`
- `make test-e2e-web` with `AGH_E2E_QA_OUTPUT_DIR=.compozy/tasks/autonomous`
- `packages/site` source generation, typecheck, tests, and production build

Known non-blocking warnings:

- Node emitted repeated `NO_COLOR` ignored because `FORCE_COLOR` is set.
- Web build emitted the existing Vite chunk-size warning for a chunk over 500 kB.
- macOS linker emitted `-bind_at_load is deprecated` while building `golangci-lint`.

## Final Gates

| Gate | Result | Evidence |
| --- | --- | --- |
| Dependency health | PASS | `qa/logs/baseline/make-deps.log` |
| Baseline repository verification | PASS | `qa/logs/baseline/make-verify-baseline.log` |
| Baseline generated contract check | PASS | `qa/logs/baseline/make-codegen-check-baseline.log` |
| Final repository verification | PASS | `qa/logs/final-make-verify-postfix.log` |
| Final generated contract check | PASS | `qa/logs/final-codegen-check-postfix.log` |
| Final daemon-served web E2E | PASS, 19/19 | `qa/logs/final-test-e2e-web-postfix.log` |
| Final site source generation | PASS | `qa/logs/final-site-source-generate-postfix.log` |
| Final site typecheck | PASS | `qa/logs/final-site-typecheck-postfix.log` |
| Final site tests | PASS, 9 files / 43 tests | `qa/logs/final-site-test-postfix.log` |
| Final site production build | PASS, 242 static pages | `qa/logs/final-site-build-postfix.log` |

## Execution Matrix

| Case | Track | Result | Evidence |
| --- | --- | --- | --- |
| `TC-AUTO-001` | Coordinator config defaults/resolver precedence | PASS | `qa/logs/targeted/TC-AUTO-001/config-coordinator.log` |
| `TC-AUTO-002` | Agent contracts, OpenAPI, generated contracts, token redaction parity | PASS | `qa/logs/targeted/TC-AUTO-002/contract-api.log`, `qa/logs/targeted/TC-AUTO-002/openapi-spec-full.log`, `qa/logs/targeted/TC-AUTO-002/codegen-check.log` |
| `TC-AUTO-003` | Autonomy hooks and safety guards | PASS | `qa/logs/targeted/TC-AUTO-003/hooks-autonomy.log`, `qa/logs/targeted/TC-AUTO-003/hooks-integration.log` |
| `TC-AUTO-004` | Situation context and caller identity | PASS | `qa/logs/targeted/TC-AUTO-004/situation-identity.log` |
| `TC-AUTO-005` | Agent self and channel verbs | PASS | `qa/logs/targeted/TC-AUTO-005/channel-ux.log`, `qa/logs/targeted/TC-AUTO-005/channel-cli-integration.log` |
| `TC-AUTO-006` | Task-run lease schema, capability rows, restart reads, redacted reads | PASS | `qa/logs/TC-AUTO-006/schema-inspection.log`, `qa/logs/TC-AUTO-006/capability-rows.log`, `qa/logs/TC-AUTO-006/restart-read.log`, `qa/logs/TC-AUTO-006/read-model-redaction.log` |
| `TC-AUTO-007` | `ClaimNextRun` and token-fenced lease mutations | PASS | `qa/logs/smoke/TC-AUTO-007/claim-lease-unit.log` |
| `TC-AUTO-008` | Agent task lease API and CLI lifecycle | PASS | `qa/logs/smoke/TC-AUTO-008/agent-task-unit.log`, `qa/logs/smoke/TC-AUTO-008/agent-task-cli-integration.log` |
| `TC-AUTO-009` | Execution boundary and coordination channel binding | PASS | `qa/logs/smoke/TC-AUTO-009/task-boundary-unit.log`, `qa/logs/smoke/TC-AUTO-009/task-boundary-integration.log` |
| `TC-AUTO-010` | Scheduler wake, sweep, restart recovery | PASS | `qa/logs/smoke/TC-AUTO-010/scheduler-unit-full.log`, `qa/logs/smoke/TC-AUTO-010/scheduler-integration.log` |
| `TC-AUTO-011` | Session lineage and spawn metadata persistence | PASS | `qa/logs/targeted/TC-AUTO-011/lineage-readmodels.log`, `qa/logs/targeted/TC-AUTO-011/lineage-integration.log` |
| `TC-AUTO-012` | Safe spawn permission narrowing and lease release | PASS | `qa/logs/smoke/TC-AUTO-012/spawn-unit.log` |
| `TC-AUTO-013` | Coordinator bootstrap and restricted orchestration | PASS | `qa/logs/smoke/TC-AUTO-013/coordinator-unit-corrected.log`, `qa/logs/smoke/TC-AUTO-013/coordinator-integration.log` |
| `TC-AUTO-014` | Coordination channels are conversation only | PASS | `qa/logs/smoke/TC-AUTO-014/channel-redaction-unit.log` |
| `TC-AUTO-015` | Tasks UI manual-first labels and coordinator handoff | PASS after `BUG-001` and `BUG-003` | `qa/logs/targeted/TC-AUTO-015/web-lint.log`, `qa/logs/targeted/TC-AUTO-015/web-typecheck.log`, `qa/logs/targeted/TC-AUTO-015/web-test.log`, `qa/logs/final-test-e2e-web-postfix.log` |
| `TC-AUTO-016` | Runtime autonomy docs and CLI reference consistency | PASS | `qa/logs/targeted/TC-AUTO-016/site-source-generate.log`, `qa/logs/targeted/TC-AUTO-016/site-typecheck.log`, `qa/logs/targeted/TC-AUTO-016/site-test.log`, `qa/logs/targeted/TC-AUTO-016/site-build.log`, final `qa/logs/final-site-*-postfix.log` |
| `TC-AUTO-017` | End-to-end coordinated run from manual task to token-fenced completion | PASS | `qa/logs/smoke/TC-AUTO-017/full-autonomy-smoke.log`, corroborated by `qa/logs/final-test-e2e-web-postfix.log` |
| `TC-AUTO-018` | Post-MVP boundary and non-regression scope | PASS | `qa/logs/TC-AUTO-018/web-route-scope.log`, `qa/logs/TC-AUTO-018/post-mvp-scope-rg.log`, `qa/logs/TC-AUTO-018/network-message-kind-scope.log`, `qa/logs/TC-AUTO-018/memory-scope.log`, `qa/logs/TC-AUTO-018/follow-up-notes.md` |

## End-To-End Workflow Proof

The QA evidence covers the required autonomy MVP workflow:

- **Task creation is saved intent:** `TC-AUTO-009`, `TC-AUTO-015`, and `tasks-coordinator-handoff.spec.ts` verify creation alone does not enqueue work.
- **Start/publish/approval creates executable work:** `TC-AUTO-009`, `TC-AUTO-013`, `TC-AUTO-015`, and `TC-AUTO-017` verify explicit operator boundaries enqueue runs idempotently.
- **Coordinator bootstrap is workspace-scoped:** `TC-AUTO-013` and final web E2E verify one coordinator/handoff path for executable workspace work.
- **Claims use `ClaimNextRun`:** `TC-AUTO-007`, `TC-AUTO-008`, `TC-AUTO-017`, and `TC-AUTO-006` verify claim criteria, capability matching, and token boundaries.
- **Leases heartbeat and fence mutations:** `TC-AUTO-007`, `TC-AUTO-008`, and `TC-AUTO-017` verify heartbeat, release, complete, fail, stale-token rejection, and expired-lease recovery.
- **Coordination channel is conversation only:** `TC-AUTO-005`, `TC-AUTO-014`, and `TC-AUTO-017` verify channel metadata/redaction and that channel messages do not mutate task-run status.
- **Safe spawn remains bounded:** `TC-AUTO-011`, `TC-AUTO-012`, and `TC-AUTO-013` verify lineage, permission narrowing, and coordinator-owned orchestration limits.
- **Scheduler recovers without claiming:** `TC-AUTO-010` verifies wake/sweep/recovery behavior without scheduler-side claim authority.
- **Web and docs describe manual-first autonomy:** `TC-AUTO-015`, `TC-AUTO-016`, final web E2E, and final site gates verify copy, labels, docs, and generated references.

## Issues Found And Fixed

| Issue | Severity | Root Cause | Fix | Verification |
| --- | --- | --- | --- | --- |
| `BUG-001-web-e2e-workspace-onboarding-race.md` | P1 | Daemon-served Playwright specs checked onboarding visibility before either onboarding or app shell was settled; one handoff test skipped the real session-create dialog. | Added `web/e2e/fixtures/workspace.ts`, updated daemon-served specs to wait for onboarding or shell, and updated the handoff spec to submit the real dialog. | `qa/logs/targeted/TC-AUTO-015/playwright-tasks-coordinator-handoff-rerun-2.log`, `qa/logs/final-test-e2e-web-postfix.log` |
| `BUG-002-acpmock-situation-context-matching.md` | P0 | Task 04 situation context prepended `<agh-situation-context>` to prompts, while `acpmock` exact-matched raw prompt text. | Canonicalized `acpmock` user-text matching to strip the daemon-owned situation context prefix while preserving exact fixture `user_text` matching. | `go test ./internal/testutil/acpmock -count=1`, `qa/logs/final-test-e2e-web-rerun-failing.log`, `qa/logs/final-test-e2e-web-postfix.log` |
| `BUG-003-tasks-e2e-active-agents-state.md` | P1 | `tasks.spec.ts` accepted only empty/fallback Agents panel states, but publish now correctly renders an active task-bound run. | Added stable selectors for the multi-agent panel and asserted the active run link for the published task. | `bun run --cwd web typecheck`, `qa/logs/final-test-e2e-web-rerun-failing.log`, `qa/logs/final-test-e2e-web-postfix.log` |

## Screenshots

Screenshots were captured under `.compozy/tasks/autonomous/qa/screenshots/`, including:

- `automation-operator-history.png`
- `automation-linked-session.png`
- `session-onboarding-hydrated.png`
- `tasks-list-seeded.png`
- `tasks-draft-created.png`
- `tasks-draft-published.png`
- `tasks-detail-route.png`
- `tasks-live-agents.png`
- `tasks-dashboard.png`
- `tasks-run-detail.png`
- `tasks-linked-session.png`
- `tasks-inbox-approval-pending.png`
- `tasks-inbox-approval-approved.png`
- `tasks-approval-handoff-enqueued.png`

## Boundary Review

`TC-AUTO-018` found no accidental implementation of post-MVP scope:

- No new web routes or systems for broad coordinator/scheduler/spawn/eval/replay dashboards.
- Base network message kinds remain the existing core protocol plus the task coordination message kinds from the MVP.
- Memory matches are explicit post-MVP boundaries or existing memory documentation, not automatic broad memory/self-correction implementation.
- Follow-up notes are captured in `qa/logs/TC-AUTO-018/follow-up-notes.md`.

## Residual Risk

- No blocking residual risk remains for the autonomy MVP QA gate.
- The web chunk-size and Node color warnings are pre-existing/non-blocking verification noise; neither indicates a functional failure in this task.
