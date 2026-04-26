# Autonomy MVP QA Test Plan

**qa-output-path:** `.compozy/tasks/autonomous`
**Artifact root:** `.compozy/tasks/autonomous/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Executive Summary

The autonomy MVP connects AGH's existing runtime substrate into a local-first autonomous execution
kernel: coordinator config, agent-facing contracts, typed hooks, bounded situation context,
identity-aware CLI/UDS verbs, task-run claim leases, coordination channels, mechanical scheduler
wake/sweep behavior, session lineage, safe spawn, coordinator bootstrap, Tasks UI honesty labels,
and runtime documentation.

This plan turns tasks 01-16 into a task_18 execution matrix. The emphasis is not broad smoke
coverage. The emphasis is proving the accepted autonomy invariants through real seams: SQLite
global store state, daemon lifecycle, UDS/CLI behavior, API contracts, hook payloads, network
channel metadata, web Tasks UI rendering, and `packages/site` documentation.

Highest-risk invariants:

- `ClaimNextRun` is the only authoritative next-work primitive; scheduler, coordinator, and
  channels must not bypass it.
- Every workspace-scoped coordinated run is bound to a stable `coordination_channel_id` at
  publish/start/approval enqueue, not at task creation.
- Coordination channels carry operational conversation only; `status` and `result` messages cannot
  mutate run ownership or terminal state.
- Raw `claim_token` appears only in the synchronous claim response and never in read models, SSE,
  logs, channel messages, web payloads, docs examples, or memory summaries.
- The scheduler is a context-owned wake/sweep/recovery component, not a second run claimant.
- Safe spawn is enforced by daemon code: TTL, lineage, depth/child caps, workspace bounds,
  permission subset checks, coordinator-role denial, and active lease release on reap.
- Manual task creation, manual session start, and direct prompting remain first-class paths.

## Objectives

- Prove coordinator config and resolver precedence are validated and do not start runtime behavior
  by themselves.
- Prove agent-facing DTOs, OpenAPI, generated web types, and read models preserve the claim-token
  exposure boundary.
- Prove autonomy hooks expose typed extension points without weakening claim/lease/spawn safety.
- Prove `/agent/context` and `agh me context` return bounded, ordered, provenance-bearing
  situation sections.
- Prove agent identity is validated against active daemon session state before `me`, `ch`, `task`,
  or `spawn` operations run.
- Prove channel list/receive/send/reply support MVP message kinds, typed correlation metadata, and
  raw-token rejection.
- Prove task-run lease schema, capability rows, coordination channel IDs, and redacted read models
  persist across SQLite reopen/restart.
- Prove `ClaimNextRun`, heartbeat, complete, fail, release, and expired-lease recovery are
  transactionally fenced by the current claim token and active owning session.
- Prove publish/start/approval is the execution boundary and binds exactly one stable channel for
  coordinated workspace runs.
- Prove scheduler boot rebuild, wake notifications, expired lease sweep, and shutdown are
  context-owned and do not claim work directly.
- Prove lineage and safe spawn metadata are durable, queryable, bounded, and contract-visible where
  public.
- Prove coordinator bootstrap is one-per-workspace, config-resolved, restricted, recoverable, and
  triggered only by executable channel-bound work.
- Prove the web Tasks UI and runtime docs explain the manual-first execution boundary and
  channel-versus-task authority boundary.

## Scope

In scope:

- Backend packages touched by tasks 01-14: `internal/config`, `internal/api/contract`,
  `internal/api/spec`, `internal/api/core`, `internal/api/udsapi`, `internal/api/httpapi`,
  `internal/hooks`, `internal/situation`, `internal/agentidentity`, `internal/network`,
  `internal/task`, `internal/store/globaldb`, `internal/scheduler`, `internal/session`,
  `internal/daemon`, and `internal/cli`.
- CLI and UDS surfaces: `agh me`, `agh me context`, `agh ch list|recv|send|reply`,
  `agh task next|heartbeat|complete|fail|release`, `agh task create|publish|start|approve`,
  and `agh spawn`.
- SQLite state: `task_runs`, claim-token hash fields, lease timestamps, capability side tables,
  coordination channel indexes, session lineage fields, and global session catalog filters.
- Daemon lifecycle: config loading, boot recovery ordering, scheduler loop lifecycle,
  coordinator bootstrap/recovery, spawn reaper, parent-stop cleanup, and graceful shutdown.
- Hook surfaces: `coordinator.*`, `task.run.*`, and `spawn.*` taxonomy, payloads, patches,
  introspection, bridge dispatch, and safety guards.
- Web surfaces: generated OpenAPI types, task adapters/formatters, task detail/list/run panels,
  action labels/tooltips, mocks, and `web/e2e/tasks-coordinator-handoff.spec.ts`.
- Site docs: `packages/site/content/runtime/core/autonomy/`, CLI reference pages for `me`, `ch`,
  `task` lease verbs, `spawn`, config docs, hooks event catalog, and runtime navigation/source tests.

Out of scope for task_17 and task_18 unless a confirmed regression touches them:

- Cross-daemon swarm coordination, leader election, contract-net negotiation, vote/react/escalate
  semantics, and broad network protocol evolution.
- Broad peer/channel memory extraction, automatic per-turn memory promotion, vector memory, and
  post-MVP session summary expansions.
- Built-in MCP tools that mirror agent CLI commands.
- New autonomy dashboards, scheduler dashboards, coordinator config GUI, spawn lineage tree UI, or
  eval/replay UI.
- Marketing redesign or non-runtime site pages.

## Environment Matrix

| Environment | Purpose | Required Evidence In task_18 |
|-------------|---------|-------------------------------|
| macOS local dev with isolated `AGH_HOME` and temp workspace | Primary daemon, CLI, UDS, SQLite, and web verification | Command logs under `qa/logs/<TC-ID>/`, state paths in `qa/verification-report.md` |
| SQLite temp global DB reopened across daemon/runtime restart | Claim lease, capability rows, channel binding, session lineage, scheduler recovery | DB inspection logs and focused Go/integration test output |
| Mock ACP/session fixtures | Agent identity, `agh me`, `agh ch`, `agh task`, `agh spawn`, coordinator bootstrap without live providers | Mock fixture names, session IDs, UDS/CLI JSON samples |
| Local HTTP/Web dev server with daemon proxy | Tasks UI manual-first and coordinator handoff flows | Playwright logs and screenshots under `qa/screenshots/<TC-ID>/` |
| `packages/site` build/test environment | Runtime autonomy docs and generated CLI references | `source:generate`, typecheck, test, and build logs under `qa/logs/TC-AUTO-016/` |
| Full repository verification environment | Final gate after any task_18 fixes | Fresh `make verify` output summarized in `qa/verification-report.md` |

## Artifact Layout

Task_18 must use the same `qa-output-path=.compozy/tasks/autonomous` and write under this root:

| Path | Owner | Purpose |
|------|-------|---------|
| `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-test-plan.md` | task_17 | Feature QA plan |
| `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-regression.md` | task_17 | Smoke, targeted, and full task_18 lanes |
| `.compozy/tasks/autonomous/qa/test-cases/TC-AUTO-*.md` | task_17 | Manual execution cases seeded by this plan |
| `.compozy/tasks/autonomous/qa/issues/BUG-*.md` | task_18 if needed | Structured bug reports tied to a TC ID and source traceability |
| `.compozy/tasks/autonomous/qa/screenshots/<TC-ID>/...` | task_18 | Browser and docs screenshots |
| `.compozy/tasks/autonomous/qa/logs/<TC-ID>/...` | task_18 | Command, daemon, DB, Go, web, site, and runtime logs |
| `.compozy/tasks/autonomous/qa/verification-report.md` | task_18 | Final execution report from `qa-execution` |

## Test Strategy

1. Run smoke first. Execute the P0 cases that prove execution-boundary channel binding, claim-token
   fencing, channel non-authority, scheduler non-claiming, safe spawn cleanup, and coordinator
   singleton bootstrap.
2. If any P0 smoke case fails, stop the lane, file `BUG-*.md`, fix root cause, rerun the failing
   case, then restart smoke from the first P0 case.
3. Run targeted lanes by surface after smoke passes or after a fix touches the surface.
4. Exercise public seams. Prefer CLI/UDS/API/browser/docs behavior plus backing DB evidence over
   parser-only checks.
5. Keep redaction checks mandatory wherever raw claim tokens, channel metadata, spawn policies,
   permission atoms, `.env` values, or provider/model config can surface.
6. Finish with the full lane: all P0, then all P1, then repository `make verify`, plus web/site
   commands if task_18 changes those surfaces.

## Entry Criteria

- Tasks 01-16 are tracked as completed and their local implementation commits are present.
- The working tree state is captured before task_18 execution, including any pre-existing dirty
  task tracking or memory files.
- `qa-output-path=.compozy/tasks/autonomous` is passed unchanged to `qa-execution`.
- Test homes/workspaces use isolated temp directories and do not depend on private credentials.
- The canonical repository verification gate is identified as `make verify`.
- Web UI presence is confirmed and `web/AGENTS.md` requirements are known for any browser/web fix.
- Site docs commands are available from `packages/site/package.json`.

## Exit Criteria

- All P0 cases pass.
- At least 90% of P1 cases pass; any P1 exception has a structured `BUG-*.md` with severity,
  impact, workaround, owner, and source traceability.
- No P0 issue remains open for task ownership, token leakage, duplicate active claim,
  stale-token mutation, channel-as-status-authority, scheduler direct claim, coordinator duplicate,
  unsafe spawn, data loss, or manual-control regression.
- `make verify` passes after the last task_18 change.
- Required web/site gates pass for touched surfaces.
- `.compozy/tasks/autonomous/qa/verification-report.md` cites executed commands, evidence paths,
  pass/fail status, issues, and residual risk.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Coordinated run starts without a stable channel | Medium | Critical | P0 execution-boundary and coordinator cases inspect run DTO/DB/channel metadata before claim |
| Channel `status` or `result` is treated as terminal task state | Medium | Critical | P0 channel-authority case sends messages and verifies run state remains unchanged |
| Raw claim token leaks through read models, channel metadata, logs, web, or docs | Medium | Critical | P0/P1 redaction checks across claim, channel, contract, UI, and docs cases |
| Scheduler claims or mutates run ownership directly | Low | Critical | P0 scheduler case verifies wake/sweep only and claim happens through agent task API |
| Stale holder completes or heartbeats after lease recovery | Medium | Critical | P0 lease case forces recovery and verifies stale token failures |
| Coordinator spawns twice or on task creation | Medium | High | P0 coordinator case covers creation-no-spawn, singleton, crash recovery, and global-scope skip |
| Safe spawn silently narrows or widens permissions | Medium | High | P0 spawn case verifies subset comparator, unknown atom rejection, and post-hook enforcement |
| Manual user workflows become second-class | Medium | High | P1 UI and P0/P1 runtime cases prove manual creation/start/session flows still work |
| Web generated types drift from API DTOs | Medium | High | P1 contract and UI lanes include codegen/typecheck/tests |
| Docs describe post-MVP features as available | Medium | Medium | P1 docs case verifies MVP-only language and CLI flag parity |

## Resource Reference Lessons Applied

| Reference | QA Lesson Applied |
|-----------|-------------------|
| Paperclip issue-run orchestration plan | Enforce one active owner per durable work item; defer/suppress overlapping wakeups centrally, not through prompt convention. |
| Paperclip heartbeat CLI | Runtime QA needs terminal-status polling, event/log evidence, timeout handling, and clear skipped/no-work outcomes. |
| Paperclip agent management plans | Agent creation requires explicit policy boundaries and durable identity/lineage evidence. |
| Hermes scheduler | Scheduler QA must prove locking/recovery/advance behavior and that delivery failures do not corrupt scheduling authority. |
| Hermes runner and trajectory utilities | Agent-loop QA must capture durable action/result evidence and failure trajectories without relying on stdout alone. |
| Hermes auxiliary client | Provider/model fallback should be config-driven and observable; coordinator provider/model tests should prove precedence and errors. |
| Multica issue E2E | Browser QA should use stable test-owned fixtures, visible UI assertions, and cleanup. |
| Multica autopilot/inbox references | Mutations should invalidate or update dependent read models; inbox/channel updates are notifications, not ownership changes. |

## Traceability Matrix

| Case | Priority | Surface | Proves | Source |
|------|----------|---------|--------|--------|
| TC-AUTO-001 | P1 | Config/daemon resolver | Coordinator config defaults, validation, workspace > global > bundled precedence, no runtime behavior from config alone | task_01, TechSpec Coordinator Config, ADR-001, ADR-005 |
| TC-AUTO-002 | P1 | Contracts/OpenAPI/web types | Agent DTOs, claim-token redaction, channel metadata, generated OpenAPI/TypeScript parity | task_02, ADR-002, ADR-003, ADR-011, ADR-012 |
| TC-AUTO-003 | P1 | Hooks | `coordinator.*`, `task.run.*`, `spawn.*` taxonomy, bridge dispatch, mutation safety | task_03, ADR-004, ADR-009, ADR-012 |
| TC-AUTO-004 | P1 | Situation/identity | Bounded `/agent/context`, stable section order, provenance, validated caller identity | task_04, task_05, ADR-002, ADR-010 |
| TC-AUTO-005 | P1 | CLI/UDS/network | `agh me`, `agh ch`, MVP message kinds, correlation metadata, raw-token rejection | task_06, ADR-007, ADR-010, ADR-012 |
| TC-AUTO-006 | P0 | SQLite task store | Lease fields, capability side tables, channel index, restart reads, public redaction | task_07, ADR-003, ADR-011, ADR-012 |
| TC-AUTO-007 | P0 | Task service | `ClaimNextRun`, one active lease per session, token-fenced mutations, expired recovery, race safety | task_08, ADR-003, ADR-004, ADR-010 |
| TC-AUTO-008 | P0 | Agent task CLI/UDS | Claim/heartbeat/complete/fail/release over public agent APIs with channel metadata and redaction | task_09, ADR-002, ADR-003, ADR-012 |
| TC-AUTO-009 | P0 | Execution boundary | Creation has no run/channel/coordinator; publish/start/approval enqueues one channel-bound run idempotently | task_10, ADR-005, ADR-010, ADR-012 |
| TC-AUTO-010 | P0 | Scheduler/daemon lifecycle | Scheduler boot rebuild, wake, sweep, shutdown, no direct claim authority | task_11, ADR-003, ADR-004, ADR-009 |
| TC-AUTO-011 | P1 | Session lineage | Durable root/coordinator/spawned metadata, TTL/budget/policy DTOs, manual root sessions | task_12, ADR-006, ADR-010, ADR-011 |
| TC-AUTO-012 | P0 | Safe spawn/reaper | Permission subset, TTL/depth/child caps, coordinator-role denial, reaper lease release | task_13, ADR-006, ADR-009, ADR-010 |
| TC-AUTO-013 | P0 | Coordinator | Config-resolved singleton bootstrap, restricted tools, recovery, channel-bound orchestration | task_14, ADR-004, ADR-005, ADR-006, ADR-012 |
| TC-AUTO-014 | P0 | Channels/task authority | Channel messages cannot mutate run ownership/status and cannot carry raw claim tokens | task_06, task_08, task_10, task_14, ADR-012 |
| TC-AUTO-015 | P1 | Web Tasks UI | Manual-first labels, coordinator handoff copy, channel chip semantics, Playwright evidence | task_15, ADR-010, ADR-011, ADR-012 |
| TC-AUTO-016 | P1 | `packages/site` docs | Runtime autonomy docs, CLI reference parity, hook/config/task/channel docs, MVP-only scope | task_16, ADR-002, ADR-003, ADR-005, ADR-006, ADR-009, ADR-010, ADR-011, ADR-012 |
| TC-AUTO-017 | P0 | End-to-end runtime | User-created task -> explicit start -> channel-bound run -> coordinator/worker claim -> heartbeat -> result message -> token-fenced completion | tasks 01-14, TechSpec Data Flow, ADR-010, ADR-012 |
| TC-AUTO-018 | P1 | Boundary regression | Post-MVP network/memory/eval/dashboard features stay absent or documented out of scope | TechSpec MVP boundary, ADR-001, ADR-007, ADR-008, ADR-011 |

## Web And Site Verification Requirements

| Task | Required web verification | Required site verification |
|------|---------------------------|----------------------------|
| 01 | None unless config DTOs surface; verify no Tasks UI config assumptions. | `config-toml.mdx` documents `[autonomy.coordinator]` defaults and precedence. |
| 02 | Generated `web/src/generated/agh-openapi.d.ts` and task/session type derivations compile. | Public docs never show raw claim tokens outside claim command examples. |
| 03 | Hook event catalog consumers remain typed if exposed. | Hook event catalog lists autonomy families and mutation boundaries. |
| 04 | No broad UI required; generated context DTO stays type-safe. | Autonomy docs describe situation/context discovery through `agh me context`. |
| 05 | No operator UI identity inference regressions. | CLI docs explain agent-facing commands rely on managed session identity. |
| 06 | Channel fields in Tasks UI remain non-authoritative when shown. | Coordination channel docs list message kinds, metadata, and token rejection. |
| 07 | Run read models expose safe lease state only. | Lease docs explain read/list surfaces use hashes, not raw tokens. |
| 08 | No UI treats channel metadata as task ownership. | Lease docs explain stale token and recovery behavior. |
| 09 | CLI examples and generated contracts match task lease verbs. | CLI reference pages for `next`, `heartbeat`, `complete`, `fail`, `release`. |
| 10 | Tasks UI shows saved intent until explicit start/publish/approval. | Autonomy docs explain execution boundary and channel binding. |
| 11 | No scheduler dashboard required; existing health/observability should not regress. | Docs avoid promising scheduler hooks or UI dashboards. |
| 12 | Session DTO changes compile in web session consumers. | Session/autonomy docs mention root/coordinator/spawned lineage where public. |
| 13 | No broad spawn UI required. | `safe-spawn.mdx` and `spawn.mdx` match implemented flags and constraints. |
| 14 | Tasks UI labels coordinator handoff accurately. | Coordinator docs explain singleton, config, global-scope skip, and recovery. |
| 15 | `web/e2e/tasks-coordinator-handoff.spec.ts` or equivalent passes. | No site change unless docs reference UI labels. |
| 16 | No web change required. | `packages/site` source generation, typecheck, tests, and build pass. |

## Deliverables

- This feature QA plan.
- `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-regression.md`
- Manual test cases under `.compozy/tasks/autonomous/qa/test-cases/`
- Reserved `issues/`, `screenshots/`, and `logs/` evidence paths for task_18.
- A task_18-ready traceability matrix and P0/P1 execution order.
