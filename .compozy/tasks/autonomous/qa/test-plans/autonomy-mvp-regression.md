# Autonomy MVP Regression Suite

**qa-output-path:** `.compozy/tasks/autonomous`
**Artifact root:** `.compozy/tasks/autonomous/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Execution Rules

- Task_18 must activate `qa-execution` with `qa-output-path=.compozy/tasks/autonomous`.
- Execute smoke first. If any smoke P0 fails, stop, file `BUG-*.md`, fix the root cause, rerun the
  failing case, then restart smoke.
- Execute all P0 before P1 in every lane.
- Do not weaken tests or assertions to pass. A failed invariant requires a production/config/docs
  fix plus narrow regression coverage.
- Capture command output in `.compozy/tasks/autonomous/qa/logs/<TC-ID>/`.
- Capture browser or docs screenshots in `.compozy/tasks/autonomous/qa/screenshots/<TC-ID>/`.
- Record final evidence and residual risk in `.compozy/tasks/autonomous/qa/verification-report.md`.
- Channels are never task authority during execution. Any case that uses `agh ch` must prove task
  state separately through task APIs or backing store reads.

## Smoke Lane

Estimated duration: 20-35 minutes.

| Order | Case | Priority | Stop Condition | Minimum Evidence |
|-------|------|----------|----------------|------------------|
| 1 | TC-AUTO-009 | P0 | Task creation creates a run/channel/coordinator, or start/publish/approval creates duplicate runs/channels | API/CLI response logs, run list JSON, channel ID evidence |
| 2 | TC-AUTO-007 | P0 | Concurrent claims produce duplicate owners, stale token mutates state, or one-session lease cap fails | Focused Go/integration log and run state samples |
| 3 | TC-AUTO-008 | P0 | Agent task CLI/UDS omits channel metadata, leaks token after claim, or lease mutation bypasses token | UDS/CLI JSON samples and redaction grep log |
| 4 | TC-AUTO-014 | P0 | `agh ch send --kind status|result` changes run ownership/status or accepts raw `claim_token` | Channel message logs plus before/after run state |
| 5 | TC-AUTO-010 | P0 | Scheduler directly claims a run, fails boot recovery, or leaks goroutines on shutdown | Scheduler test log, daemon restart evidence |
| 6 | TC-AUTO-012 | P0 | Spawn widens permissions, omits TTL, exceeds caps, or reaper leaves active child lease owned | Spawn/reaper test log, lease release evidence |
| 7 | TC-AUTO-013 | P0 | Coordinator starts on creation, duplicates per workspace, lacks channel, or bypasses public task APIs | Coordinator session/run/channel evidence |
| 8 | TC-AUTO-017 | P0 | End-to-end coordinated run cannot complete through public lease APIs | Full runtime transcript, task/run/channel/session logs |

## Targeted Lanes

Run targeted lanes after smoke passes or whenever a fix touches the relevant surface.

### Config, Contracts, And Hooks

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUTO-001 | P1 | `[autonomy.coordinator]` defaults, validation, and resolver precedence |
| 2 | TC-AUTO-002 | P1 | Contract DTOs, OpenAPI, generated web types, raw token redaction |
| 3 | TC-AUTO-003 | P1 | Autonomy hook taxonomy, bridge dispatch, patch safety |

Recommended commands/evidence for task_18:

- `go test ./internal/config ./internal/daemon -run 'Autonomy|Coordinator'`
- `go test ./internal/api/contract ./internal/api/spec`
- `make codegen-check` or the repository's current generated-contract check if available
- `go test ./internal/hooks ./internal/task`
- `make web-typecheck` when generated web contracts change

### Situation, Identity, And Agent Channels

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUTO-004 | P1 | `/agent/context`, prompt augmentation, caller identity validation |
| 2 | TC-AUTO-005 | P1 | `agh me`, `agh ch`, message kinds, JSON/JSONL, operator command regression |
| 3 | TC-AUTO-014 | P0 | Channel non-authority and token rejection |

Recommended commands/evidence:

- `go test ./internal/situation ./internal/agentidentity ./internal/api/udsapi ./internal/cli`
- Local UDS session fixture with `AGH_SESSION_ID` and `AGH_AGENT`
- Channel send/recv/reply JSON and before/after run-state reads

### Task Store, Claim Lease, And Execution Boundary

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUTO-006 | P0 | SQLite lease schema, capability rows, channel ID, read redaction |
| 2 | TC-AUTO-007 | P0 | `ClaimNextRun`, heartbeat/complete/fail/release fencing, expiry recovery |
| 3 | TC-AUTO-008 | P0 | Agent task CLI/UDS lease lifecycle |
| 4 | TC-AUTO-009 | P0 | Create versus start/publish/approval enqueue boundary |

Recommended commands/evidence:

- `go test ./internal/task ./internal/store/globaldb`
- Integration restart/reopen tests for claim lease rows
- CLI/UDS flow with one successful claim and one stale-token failure
- Run list/detail API samples proving raw token is absent after claim

### Scheduler, Spawn, And Coordinator

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUTO-010 | P0 | Scheduler wake/sweep/rebuild/shutdown and no direct claim |
| 2 | TC-AUTO-011 | P1 | Session lineage and spawn metadata persistence/read models |
| 3 | TC-AUTO-012 | P0 | Safe spawn permission/TTL/reaper/lease release |
| 4 | TC-AUTO-013 | P0 | Coordinator bootstrap, singleton, restricted orchestration, recovery |

Recommended commands/evidence:

- `go test ./internal/scheduler ./internal/session ./internal/daemon ./internal/coordinator`
- `go test -tags integration ./internal/scheduler ./internal/daemon` if integration tags are available for these packages
- Spawn CLI/UDS response samples and reaper release logs
- Coordinator boot/restart evidence with workspace-scoped and global-scope runs

### Web Tasks UI

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUTO-015 | P1 | Manual-first labels, coordinator handoff, channel chip, existing manual session UI |
| 2 | TC-AUTO-002 | P1 | Generated TypeScript contract compatibility if DTOs changed |
| 3 | TC-AUTO-014 | P0 | UI must not imply channel messages own task status |

Recommended commands/evidence:

- `make web-lint`
- `make web-typecheck`
- `make web-test`
- `cd web && bunx playwright test e2e/tasks-coordinator-handoff.spec.ts` or the repository-equivalent Playwright invocation
- Screenshots for saved intent, coordinator handoff, and channel chip states

### Site Docs And CLI References

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-AUTO-016 | P1 | Runtime autonomy docs and CLI reference parity |
| 2 | TC-AUTO-018 | P1 | Post-MVP scope stays absent or explicitly out of scope |

Recommended commands/evidence:

- `cd packages/site && bun run source:generate`
- `cd packages/site && bun run typecheck`
- `cd packages/site && bun run test`
- `cd packages/site && bun run build`
- CLI docs generation check if task_18 changes command metadata

## Full Regression Lane

Estimated duration: 2-4 hours.

1. Run all P0 cases in smoke order: TC-AUTO-009, TC-AUTO-007, TC-AUTO-008,
   TC-AUTO-014, TC-AUTO-010, TC-AUTO-012, TC-AUTO-013, TC-AUTO-017.
2. Run P1 cases in this order: TC-AUTO-001, TC-AUTO-002, TC-AUTO-003,
   TC-AUTO-004, TC-AUTO-005, TC-AUTO-011, TC-AUTO-015, TC-AUTO-016,
   TC-AUTO-018.
3. Run repository gate: `make verify`.
4. Run any additional web/site commands required by files changed during task_18.
5. Populate `.compozy/tasks/autonomous/qa/verification-report.md` with:
   - exact commands executed
   - exit codes
   - evidence paths
   - pass/fail result per TC
   - bug IDs and fix commits if any
   - residual risk and skipped/blocked scenarios

## Pass, Fail, And Conditional Criteria

PASS:

- All P0 cases pass.
- At least 90% of P1 cases pass.
- `make verify` passes after the last change.
- No critical bug, data loss, duplicate active claim, token leak, channel-authority regression,
  scheduler direct-claim path, unsafe spawn, coordinator duplicate, or manual-control regression
  remains open.

FAIL:

- Any P0 case fails.
- Raw `claim_token` appears outside the synchronous claim response.
- A channel message changes task-run ownership or terminal state.
- Scheduler or coordinator claims/mutates ownership outside the task service claim path.
- Spawn widens permissions, omits TTL, ignores caps, or reaper leaves active owned leases behind.
- `make verify` fails after the final fix set.

CONDITIONAL:

- A P1 docs/UI issue remains with a documented workaround, `BUG-*.md`, explicit owner, and no P0
  impact, while all P0 and final repository gates pass.

## Regression Maintenance

After task_18:

- Promote each confirmed bug reproduction into the narrowest durable Go, web, or site regression
  test available in the repository.
- Add a new `TC-AUTO-*` case only when the discovered gap represents a reusable autonomy invariant
  not covered by this matrix.
- Keep evidence paths stable under `.compozy/tasks/autonomous/qa/` so future autonomy QA runs can
  compare reports across releases.
