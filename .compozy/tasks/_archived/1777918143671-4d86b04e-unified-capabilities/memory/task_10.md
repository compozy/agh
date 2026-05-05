# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Completed the task_09 QA matrix for unified capabilities using the fixed artifact root `.compozy/tasks/unified-capabilities/qa/`.
- Baseline gates and final reruns now pass with fresh backend/API/web/docs evidence and a published `qa/verification-report.md`.
- Confirmed regressions were fixed at the source, matched with narrow durable regression coverage, rerun through impacted lanes, and then carried into task tracking and commit prep.

## Important Decisions

- Use the existing task_09 artifacts (`test-plans/`, `test-cases/`) as the authoritative execution matrix rather than redefining scope in task_10.
- Treat baseline failures as evidence that must be classified before remediation; do not assume they are unified-capability regressions.
- Keep screenshot mirroring as best-effort QA evidence only. If the mirrored rerun is blocked by unrelated worktree typing errors, preserve the earlier passing browser flow evidence and record the blocker in `qa/verification-report.md` instead of widening scope.

## Learnings

- The QA artifact set already exists under `.compozy/tasks/unified-capabilities/qa/` with P0/P1 cases and stable `issues/` / `screenshots/` paths.
- The repo-level QA contract discovery script identifies `make verify` as the canonical umbrella gate and detects a live web surface.
- The current worktree includes unrelated dirty web/session changes, so any baseline failure has to be traced carefully before editing.
- The first `make web-typecheck` failure was caused by my own parallel pre-flight execution and did not reproduce when rerun sequentially.
- The first `make verify` failure in `internal/acp` did not reproduce under targeted `go test -race ./internal/acp`, the direct gotestsum lane, or the clean sequential `make verify` rerun.
- Filtered rich `whois` discovery regressed because the response kept the original `peer_card.capabilities` while returning a filtered `capability_catalog`; brief and rich discovery must be projected from the same filtered selection.
- Same-daemon local-directed lifecycle regressed because sender-side `receipt`/`trace` syncing ran before the local receive path; local terminal lifecycle messages must not be pre-synced on send.
- The shipped browser operator flow depended on stable route keys and non-scaling dialog animation; the final UI fix used `router.latestLocation.pathname`, memoized popup/overlay nodes, and fade-only dialog transitions.
- Final repo gates also required non-feature gate repairs in `internal/api/httpapi/prompt.go`, `magefile.go`, `internal/api/httpapi/handlers_test.go`, and `internal/api/udsapi/handlers_test.go`.

## Files / Surfaces

- `.compozy/tasks/unified-capabilities/qa/test-plans/unified-capabilities-test-plan.md`
- `.compozy/tasks/unified-capabilities/qa/test-plans/unified-capabilities-regression.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-INT-001.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-INT-002.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-INT-003.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-INT-004.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-UI-001.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-REG-001.md`
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-REG-002.md`
- Root `AGENTS.md` / `CLAUDE.md`
- `web/AGENTS.md`
- `.compozy/tasks/unified-capabilities/qa/issues/BUG-001.md`
- `.compozy/tasks/unified-capabilities/qa/issues/BUG-002.md`
- `.compozy/tasks/unified-capabilities/qa/issues/BUG-003.md`
- `.compozy/tasks/unified-capabilities/qa/verification-report.md`
- `internal/network/capability_catalog.go`
- `internal/network/router.go`
- `internal/network/router_test.go`
- `internal/network/manager_test.go`
- `internal/daemon/daemon_network_collaboration_integration_test.go`
- `internal/daemon/daemon_test.go`
- `web/src/routes/_app.tsx`
- `web/src/routes/-_app.test.tsx`
- `packages/ui/src/components/dialog.tsx`
- `packages/ui/src/components/dialog.test.tsx`
- `internal/api/httpapi/prompt.go`
- `magefile.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/udsapi/handlers_test.go`

## Errors / Corrections

- Corrected the baseline classification: the initial pre-flight failures were not stable repo regressions, so task_10 can proceed to the scenario matrix without code changes.
- Corrected three confirmed QA regressions:
  - filtered rich discovery brief/rich mismatch in `internal/network`
  - same-daemon local trace lifecycle drops in `internal/network`
  - browser operator flow instability across app layout/dialog seams in `web/` and `packages/ui`
- The screenshot-mirroring rerun using `AGH_E2E_QA_OUTPUT_DIR=.compozy/tasks/unified-capabilities` was blocked by unrelated current-worktree `tsgo --noEmit` failures in `web/src/lib/api-client.ts` and `web/src/lib/daemon-api-contract.test.ts`; the report records the blocker instead of patching unrelated work.

## Ready for Next Run

- Task_10 is complete once the selective commit is written. If a follow-up run needs mirrored browser screenshots under the QA root, clear the unrelated web typing blockers first and rerun the Playwright network lane with `AGH_E2E_QA_OUTPUT_DIR=.compozy/tasks/unified-capabilities`.
