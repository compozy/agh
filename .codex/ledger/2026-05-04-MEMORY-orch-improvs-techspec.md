Goal (incl. success criteria):

- Create and maintain the English TechSpec set for `.compozy/tasks/orch-improvs` based on `.compozy/tasks/orch-improvs/analysis`.
- Success requires: cy-spec-preflight context loaded, codebase exploration completed, technical clarifications answered, ADRs created, approved orchestration hardening design saved, review-gate design incorporated from Codex Loop goal research, and `_techspec.md` kept as the canonical aggregate for task generation.

Constraints/Assumptions:

- Conversation in Brazilian Portuguese; persistent artifacts in English.
- Must follow `cy-spec-preflight` and `cy-create-techspec`.
- TechSpec files are persistent artifacts; new review-gate child content should receive a peer-review pass before task generation.
- ADRs may be written before the TechSpec as required by the skill.
- Local date for ledger naming is `2026-05-04` from `date +%F`.

Key decisions:

- MVP scope: Core 1-7 from the synthesis plus task context bundle, cursor-seeded SSE, bundled orchestration skills, and notifier cursor. Bulk endpoints and frontend extension/plugin SDK are out of MVP unless pulled back in later.
- Primary architecture: preserve existing AGH authorities (`task.Service`, `task_runs`, `BaseHandlers`, scheduler observe/wake/recover only); adapt Hermes patterns into existing packages rather than introducing a new orchestration package or queue.
- Autonomy fit: AGH already has the substantial autonomy substrate: single `task_runs` queue, token-fenced/session-bound lease mutation, mechanical scheduler, coordinator runtime, safe spawn lineage, typed task hooks, `/agent/context`, task stream `after_sequence`/`Last-Event-ID`, and bundled generic skills. `orch-improvs` must be framed as orchestration hardening/enrichment, not a new autonomy subsystem.
- Data model: use explicit typed columns/side-tables, not JSON, for queryable orchestration state. Planned fields include `task_runs.summary`, `tasks.current_run_id`, `tasks.max_runtime_seconds`, `tasks.spawn_failure_count`, `tasks.last_spawn_error`, and a durable notifier cursor side-table keyed by bridge/thread subscription. `metadata_json` and `result_json` remain opaque payloads only.
- `tasks.current_run_id`: keep in MVP as a denormalized read projection only. `task_runs` remains the only authoritative execution queue and ownership source; only `task.Service`/store transitions may update `current_run_id`; scheduler/coordinator/web/API may not use it as claim/assignment/terminal-state authority.
- Config lifecycle: add minimal explicit `[task.orchestration]` config with documented defaults: `summary_max_bytes = 4096`, `context_body_max_bytes = 8192`, `context_prior_attempts = 5`, `context_recent_events = 50`, `spawn_failure_limit = 5`, `scheduler_bad_tick_threshold = 6`, `scheduler_bad_tick_cooldown = "5m"`, and `default_max_runtime = "0s"` where zero disables the default budget. Tasks may override `max_runtime_seconds`.
- Notification cursors: create a new shared `internal/notifications` primitive for durable event delivery cursors instead of anchoring the notifier cursor only inside `internal/bridges`.
- Bundled skills: create both `agh-task-worker` and `agh-orchestrator`. `agh-orchestrator` is a versioned instruction source injected deterministically by the coordinator runtime at coordinator bootstrap, not dependent on manual discovery. `agh-task-worker` guides normal spawned/manual workers. Both are instructional only; runtime authority remains in `task.Service`, `task_runs`, session-bound lease lookup, tool policy, coordinator runtime, scheduler boundaries, and spawn lineage.
- Prior autonomy/supervisor-orchestration artifacts are archived under `.compozy/tasks/_archived/`; cite them as archived prior art, not active task state.
- Spec structure: keep `_techspec.md` as the canonical aggregate, move the already reviewed orchestration hardening design to `_techspec_orchestration.md`, and add `_techspec_review_gate.md` as the review-on-stop / goal-continuation child spec.
- Review gate v1: post-terminal continuation loop, no `pending_review` run status. Review requests/verdicts are typed task-owned state persisted through `task.Service`; channels can route/coordinate reviewers but cannot define verdicts.
- Review gate state: add task review policy/rollup fields and `task_run_reviews`. `missing_work` and `next_round_guidance` feed the next worker through `TaskContextBundle.ReviewContinuation`.
- Review skill: add `agh-task-reviewer` as an instructional-only bundled skill loaded for reviewer sessions bound to persisted review requests.
- Task execution profile decision: add typed per-task profile fields for coordinator guidance, worker selection, review selection, participant/channel policy, and sandbox override. MVP keeps the coordinator as a workspace singleton with task-specific `guided` policy; `dedicated` coordinator per task is out of MVP.

State:

- TechSpec approved by the user and originally saved to `.compozy/tasks/orch-improvs/_techspec.md`.
- Peer review round 1 completed via `cy-spec-peer-review`.
- Review verdict: `NEEDS_REWORK` with 9 blockers and 10 nits.
- Peer-review summary saved to `.compozy/tasks/orch-improvs/qa/peer-review-summary-round1.md`.
- User selected incorporation option B: selected blockers/nits.
- Incorporated selected review findings into the TechSpec and tied ADRs.
- Incorporation record saved to `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round1.md`.
- Peer review round 2 completed via `cy-spec-peer-review`.
- Review round 2 verdict: `NEEDS_REWORK` with 7 blockers and 10 nits.
- Peer-review summary round 2 saved to `.compozy/tasks/orch-improvs/qa/peer-review-summary-round2.md`.
- User selected round 2 incorporation set: all blockers `B-001` through `B-007`; nits `N-001`, `N-002`, `N-003`, `N-004`, `N-006`, `N-007`, `N-008`, `N-009`, and `N-010`; `N-005` deferred.
- Incorporated the selected round 2 findings into the TechSpec and tied ADRs.
- Incorporation record saved to `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round2.md`.
- User requested incorporating a Codex Loop `goal`-inspired review mechanism and restructuring the spec into an aggregate plus child specs.
- Research completed against `/Users/pedronauck/dev/ai/codex-loop-plugin`, `.resources/codex`, `cy-codex-loop`, and current AGH task/channel code.
- `_techspec.md` is now the canonical aggregate spec.
- `_techspec_orchestration.md` contains the previously peer-reviewed orchestration hardening design.
- `_techspec_review_gate.md` contains the new full review-gate child spec.
- ADRs 007-009 were created for post-terminal review gate, channel routing without authority, and typed review verdict/continuation state.
- `analysis/analysis_codex-loop-goal-review.md` was created to preserve research evidence.
- Text validation passed: expected markers present and code fences balanced across TechSpecs/ADRs.
- User proposed adding task-selectable/default orchestration properties: custom coordinator agent, custom review agents/channels, involved agents/channels, and per-task sandbox choice.

Done:

- Loaded `cy-spec-preflight` and `cy-create-techspec` skill instructions.
- Confirmed target analysis files exist under `.compozy/tasks/orch-improvs/analysis`.
- Scanned `.codex/ledger` for related `orch`/`orchestr` ledgers; none found.
- Loaded AGH spec authoring playbook, standing directives, glossary, TechSpec phase lessons, root `CLAUDE.md`, `internal/CLAUDE.md`, `web/CLAUDE.md`, `packages/site/CLAUDE.md`, and `cy-web-docs-impact`.
- Read `.compozy/tasks/orch-improvs/analysis/*.md`; no `_prd.md`, `_techspec.md`, or `adrs/` currently exists.
- Explored task/autonomy code: `internal/task`, `internal/store/globaldb`, `internal/api/core`, `internal/api/contract`, `internal/api/udsapi`, `internal/tools/builtin`, `internal/scheduler`, and `web/src/systems/tasks`.
- Confirmed AGH already has session-bound agent lease mutations through `LookupActiveRunForSession`; worker-scoped mutation enforcement should be preserved/extended rather than treated as absent.
- Received read-only explorer findings and closed the explorer agent.
- User selected scope: Core 1-7 + context bundle + cursor-seeded SSE + skills + notifier.
- User selected explicit typed columns/side-tables for data model.
- User selected minimal explicit `[task.orchestration]` config defaults.
- User selected new shared `internal/notifications` for notifier cursors.
- User clarified the prior autonomy material lives under `.compozy/tasks/_archived`.
- Read-only autonomy-fit subagent completed and was closed. Main correction: prior artifact paths cited by subagent as missing are actually archived; use archived paths found locally instead of active `.compozy/tasks/autonomous`.
- User selected bundled skills decision B with nuance: `agh-task-worker` plus injected `agh-orchestrator`, with explicit ADR guardrail that skills are instructional artifacts only and cannot define runtime authority, permission boundaries, ownership, queue semantics, or terminal state.
- User selected `tasks.current_run_id` option A: keep it as a denormalized read projection with strict authority guardrails.
- Created six ADRs under `.compozy/tasks/orch-improvs/adrs/`:
  - `adr-001-orchestration-hardening-extends-existing-autonomy.md`
  - `adr-002-queryable-orchestration-state.md`
  - `adr-003-shared-durable-notification-cursors.md`
  - `adr-004-minimal-task-orchestration-config.md`
  - `adr-005-current-run-id-denormalized-projection.md`
  - `adr-006-bundled-orchestration-skills-are-instructional.md`
- Verified ADR files are present and contain expected status/decision/reference sections via `find` and `rg`.
- Presented complete TechSpec draft in chat and received user approval with option A.
- Saved approved TechSpec to `.compozy/tasks/orch-improvs/_techspec.md`.
- Verified `_techspec.md` exists and contains the expected canonical headings plus the final Architecture Decision Records section.
- Loaded `cy-spec-peer-review` instructions plus `references/quality-markers.md` and `references/peer-review-prompt.md`.
- Peer-review preflight findings: missing exact final-shape sections `## Architectural Boundaries`, `## Implementation Steps`, and `## Test Strategy`; missing explicit MVP boundary statement; missing first-class architectural boundaries/import rules section; missing numbered lease/safety invariants. Go interfaces, data-model rationale, and side-table-vs-JSON decisions were present enough to reuse but needed tightening.
- User approved amending the TechSpec for peer-review quality markers before running Opus.
- Amended `.compozy/tasks/orch-improvs/_techspec.md` with an explicit MVP boundary, `## Architectural Boundaries`, `## Test Strategy`, `## Implementation Steps`, import/package boundary rules, and numbered lease/ownership/delivery invariants.
- Created `.compozy/tasks/orch-improvs/qa/peer-review-prompt-round1.md`.
- Ran `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json --prompt-file .compozy/tasks/orch-improvs/qa/peer-review-prompt-round1.md`; command exited 0.
- Captured raw event-stream stdout to `.compozy/tasks/orch-improvs/qa/peer-review-result-round1.json` and stderr to `.compozy/tasks/orch-improvs/qa/peer-review-result-round1.err`.
- Extracted strict review JSON to `.compozy/tasks/orch-improvs/qa/peer-review-result-round1-extracted.json`.
- Review round 1 result: `NEEDS_REWORK`, 9 blockers, 10 nits.
- Wrote `.compozy/tasks/orch-improvs/qa/peer-review-summary-round1.md`.
- Stderr contains a non-fatal extension discovery warning for `.compozy/extensions/cy-qa-workflow/extension.toml` with unknown hook event `plan.pre_resolve_task_runtime`.
- User chose option B and accepted the recommended incorporation set: all blockers `B-001` through `B-009`, plus nits `N-002` through `N-010`; `N-001` deferred.
- Incorporated peer-review findings into `.compozy/tasks/orch-improvs/_techspec.md`, ADR-001, ADR-002, ADR-003, ADR-004, ADR-005, and ADR-006.
- Added concrete Go signatures, cursor invariants, `current_run_id` transition matrix, synthetic terminal run scope, spawn-failure breaker policy, max-runtime actor sequence, deterministic coordinator skill injection contract, `/agent/context` lease binding, `latest_event_seq` SSE seed contract, bundled skill frontmatter, codegen artifact list, hook/observe dispatch mapping, and expanded Web/Docs impact.
- Created `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round1.md`.
- Validation performed: TechSpec/ADR code fences balanced; expected markers present for Go signatures, cursor invariants, `latest_event_seq`, context binding, contract artifacts, skill load metadata, notification cursor index, implementation/test sections, and deferred `N-001`.
- User requested peer-review round 2.
- Revalidated updated TechSpec quality markers for round 2.
- Created `.compozy/tasks/orch-improvs/qa/peer-review-prompt-round2.md`.
- Ran `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json --prompt-file .compozy/tasks/orch-improvs/qa/peer-review-prompt-round2.md`; command exited 0.
- Captured raw round 2 event-stream stdout to `.compozy/tasks/orch-improvs/qa/peer-review-result-round2.json` and stderr to `.compozy/tasks/orch-improvs/qa/peer-review-result-round2.err`.
- Extracted strict round 2 review JSON to `.compozy/tasks/orch-improvs/qa/peer-review-result-round2-extracted.json`.
- Round 2 result: `NEEDS_REWORK`, 7 blockers, 10 nits. Main blocker themes: task-level terminal mutation against active token-fenced runs, synthetic run concurrency, `ClaimNextRun` spawn-circuit filter/index design, notification cursor primitive with no MVP consumer, spawn failure observer/reset call sites, public `MaintainCurrentRunID` misuse seam, and task-level endpoint authorization.
- Wrote `.compozy/tasks/orch-improvs/qa/peer-review-summary-round2.md`.
- Round 2 stderr is empty.
- User directed incorporation of all round 2 blockers plus nits `N-001`, `N-002`, `N-003`, `N-004`, `N-006`, `N-007`, `N-008`, `N-009`, and `N-010`; `N-005` remains optional/deferred.
- Updated `.compozy/tasks/orch-improvs/_techspec.md` with active-run task-level terminal rejection, synthetic terminal concurrency, `ClaimNextRun` spawn-circuit filtering, concrete bridge terminal notification consumer, spawn-failure call sites/reset semantics, no public `MaintainCurrentRunID`, operator-only endpoint authorization, HTTP/UDS parity matrix, scheduler observe-only event wording, `requires_active_task_claim` loader support, `24h` max watchdog validation, `Last-Event-ID` precedence, and expanded delete/replace targets.
- Kept `internal/notifications` in MVP as a durable cursor primitive only and specified `internal/bridges` as the first concrete consumer through bridge-delivered terminal task notifications using `bridge_task_subscriptions`, `notification_cursors`, durable `task_events.event_seq` replay, direct `bridges/deliver`, deterministic `delivery_id`, and post-success cursor advancement.
- Updated ADR-001, ADR-002, ADR-003, ADR-004, ADR-005, and ADR-006 to match the round 2 decisions.
- Created `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round2.md`.
- Validation performed: TechSpec/ADR code fences balanced; expected markers present for all incorporated round 2 blockers/nits; `N-005` documented as deferred.
- Researched Codex Loop goal mechanism:
  - `goal_confirm.go` structured verdict includes completion/confidence/reason/missing_work/next_round_guidance.
  - `hooks.go` runs review on Stop and continues unless completed.
  - `store.go` persists loop state and goal outcomes.
  - rapid-stop guardrails exist and should map to review circuit behavior.
- Researched `.resources/codex` goal/review types:
  - `ThreadGoal`, `ThreadGoalStatus`, `ReviewStartParams`, `ReviewTarget`, `ReviewDelivery`, `ThreadSourceKind`, `NonSteerableTurnKind`, and `ApprovalsReviewer`.
- Confirmed AGH already has `review_request` as a coordination message kind in `internal/task/lease.go`, making channel-based review routing compatible as non-authoritative coordination.
- Copied the approved orchestration design into `.compozy/tasks/orch-improvs/_techspec_orchestration.md` and marked review gate out of scope for that child spec.
- Rewrote `.compozy/tasks/orch-improvs/_techspec.md` as a 263-line aggregate master with normative precedence, shared authority model, lifecycle, data ownership matrix, surface matrix, config lifecycle, implementation sequence, tests, peer-review plan, and ADR index.
- Created `.compozy/tasks/orch-improvs/_techspec_review_gate.md` as a 648-line child spec with research basis, goals/non-goals, MVP boundary, authority model, lifecycle, review policy, routing, bundled skill, data model, interfaces, native tool, context bundle, API/UDS/CLI, hooks, failure policy, security, web/docs impact, implementation steps, tests, and risks.
- Created ADRs:
  - `adr-007-review-gate-post-terminal-continuation-loop.md`
  - `adr-008-review-routing-uses-channels-without-channel-authority.md`
  - `adr-009-review-verdicts-and-continuation-guidance-are-typed-task-state.md`
- Updated ADR-001, ADR-002, ADR-004, and ADR-006 to reflect review gate, review config, task review state, and `agh-task-reviewer`.
- Created `.compozy/tasks/orch-improvs/analysis/analysis_codex-loop-goal-review.md`.
- Validation performed: code fences balanced for `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, and all ADRs; expected review markers present; no `_techspec_1`/`_techspec_2` references found.
- Ran `make verify`; it passed.
- Reconfirmed runtime context for task execution profiles: global/workspace coordinator config exists in `internal/config/autonomy.go`; workspace has `DefaultAgent` and `SandboxRef`; tasks/runs already have `network_channel`, `coordination_channel_id`, `required_capabilities`, and `preferred_capabilities`; task session bridge currently starts system sessions without `AgentName`/`Provider`; sandbox is resolved from workspace during session start; `ClaimCriteria.AgentName` exists but claim SQL does not yet filter by it.
- User selected option A for task execution profiles: `TaskExecutionProfile v1` with typed `CoordinatorProfile`, `WorkerProfile`, `ReviewProfile`, `ParticipantPolicy`, and `SandboxPolicy`; coordinator modes are `inherit` and `guided` in MVP.
- Created ADR-010 and analysis for task execution profiles.
- Updated aggregate, orchestration, review-gate specs, and ADRs 001/002/004/006/008/009 with the approved profile design.
- Validation passed: all three TechSpecs carry the six required TechSpec markers, code fences are balanced, expected profile markers are present, and `make verify` passed.
- Peer review round 3 completed via `cy-spec-peer-review`.
- Review round 3 verdict: `NEEDS_REWORK` with 6 blockers and 10 nits.
- Round 3 artifacts saved under `.compozy/tasks/orch-improvs/qa/peer-review-{prompt,result,result-extracted,summary}-round3.*`; stderr is empty.
- Round 3 blocker themes: typed continuation-run schema, reviewer-session binding, coordinator review wake mechanism, `RecordRunReview` continuation atomicity, `ParticipantPolicy` enforcement surface, and review-request transaction boundary.
- Post-round-3 validation passed after retry: all three TechSpecs carry required markers, extracted review JSON parses, code fences are balanced, and `make verify` completed successfully.
- User selected round 3 incorporation set: all blockers `B-001` through `B-006`; nits `N-001`, `N-003`, `N-004`, `N-005`, `N-007`, `N-008`, `N-009`, and `N-010`; `N-002` and `N-006` deferred.
- Incorporated selected round 3 findings into the aggregate TechSpec, orchestration child spec, review-gate child spec, and ADRs 002/003/007/008/009/010.
- Round 3 incorporation record saved to `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round3.md`.
- Validation after round 3 incorporation passed: all three TechSpecs carry required markers, code fences are balanced, targeted round-3 markers are present, and `make verify` completed successfully.
- Peer review round 4 completed via `cy-spec-peer-review`.
- Review round 4 verdict: `NEEDS_REWORK` with 2 blockers and 6 nits.
- Round 4 artifacts saved under `.compozy/tasks/orch-improvs/qa/peer-review-{prompt,result,result-extracted,summary}-round4.*`; stderr is empty.
- Round 4 blocker themes: `task_runs` continuation-column migration ownership/FK ordering conflict, and bridge terminal notifier fail-closed behavior conflicting with review/continuation lifecycle.
- Round 4 validation passed after retry: extracted review JSON parses, prompt/summary fences are balanced, all three TechSpecs carry required markers, initial `make verify` timed out once in `@agh/extension-sdk` integration test, and rerun `make verify` completed successfully.
- User selected option A for round 4 incorporation: all blockers and all nits.
- Incorporated every round 4 finding into the aggregate TechSpec, orchestration child spec, review-gate child spec, and ADRs 002/003/007/008/009/010.
- Round 4 incorporation record saved to `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round4.md`.
- Validation after round 4 incorporation passed: extracted review JSON parses, Markdown code fences are balanced, required TechSpec quality markers are present, stale round-4 ambiguity strings are absent, target round-4 markers are present, and `make verify` exited 0. Gate evidence: Bun lint reported 0 warnings/errors, Vitest passed 329 files / 2088 tests, Go race test gate completed 8066 tests, and package boundaries were respected. Non-fatal warnings observed: Vite chunk-size warning and macOS linker `-bind_at_load` deprecation warning.

Now:

- Report round 4 incorporation result and ask whether to run peer-review round 5 or stop.

Next:

- User choice: run another `cy-spec-peer-review` round, stop peer review and proceed toward task generation, or make manual spec edits first.

Open questions (UNCONFIRMED if needed):

- Whether to run peer-review round 5 or stop with the current saved spec.

Working set (files/ids/commands):

- `.compozy/tasks/orch-improvs/analysis/*.md`
- `.compozy/tasks/orch-improvs/adrs/`
- `.compozy/tasks/orch-improvs/adrs/adr-001-orchestration-hardening-extends-existing-autonomy.md`
- `.compozy/tasks/orch-improvs/adrs/adr-002-queryable-orchestration-state.md`
- `.compozy/tasks/orch-improvs/adrs/adr-003-shared-durable-notification-cursors.md`
- `.compozy/tasks/orch-improvs/adrs/adr-004-minimal-task-orchestration-config.md`
- `.compozy/tasks/orch-improvs/adrs/adr-005-current-run-id-denormalized-projection.md`
- `.compozy/tasks/orch-improvs/adrs/adr-006-bundled-orchestration-skills-are-instructional.md`
- `.compozy/tasks/orch-improvs/_techspec.md`
- `.compozy/tasks/orch-improvs/_techspec_orchestration.md`
- `.compozy/tasks/orch-improvs/_techspec_review_gate.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_codex-loop-goal-review.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-prompt-round1.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round1.json`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round1-extracted.json`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round1.err`
- `.compozy/tasks/orch-improvs/qa/peer-review-summary-round1.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round1.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-prompt-round2.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round2.json`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round2-extracted.json`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round2.err`
- `.compozy/tasks/orch-improvs/qa/peer-review-summary-round2.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round2.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-prompt-round4.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round4.json`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round4-extracted.json`
- `.compozy/tasks/orch-improvs/qa/peer-review-result-round4.err`
- `.compozy/tasks/orch-improvs/qa/peer-review-summary-round4.md`
- `.compozy/tasks/orch-improvs/qa/peer-review-incorporation-round4.md`
- `.compozy/tasks/orch-improvs/adrs/adr-007-review-gate-post-terminal-continuation-loop.md`
- `.compozy/tasks/orch-improvs/adrs/adr-008-review-routing-uses-channels-without-channel-authority.md`
- `.compozy/tasks/orch-improvs/adrs/adr-009-review-verdicts-and-continuation-guidance-are-typed-task-state.md`
- `.compozy/tasks/orch-improvs/adrs/adr-010-task-execution-profiles-are-typed-overlays.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_task-execution-profile.md`
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/{activation.go,hooks.go,goal_confirm.go,store.go,config.go}`
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/{ThreadGoal.ts,ThreadGoalStatus.ts,ReviewStartParams.ts,ReviewTarget.ts,ReviewDelivery.ts,ReviewStartResponse.ts,ThreadSourceKind.ts,NonSteerableTurnKind.ts,ApprovalsReviewer.ts}`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/**`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/**`
- `internal/situation/**`
- `internal/coordinator/**`
- `internal/skills/bundled/**`
- `docs/_memory/spec-authoring-playbook.md`
- `docs/_memory/standing_directives.md`
- `docs/_memory/glossary.md`
- `internal/task/{interfaces.go,lease.go,lease_manager.go,manager.go,types.go}`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_task_claim.go`
- `internal/config/autonomy.go`
- `internal/workspace/workspace.go`
- `internal/session/sandbox.go`
- `internal/daemon/task_runtime.go`
- `internal/api/{contract,core,udsapi,httpapi}`
- `internal/tools/builtin/{tasks.go,autonomy.go}`
- `internal/scheduler/{scheduler.go,doc.go}`
- `web/src/systems/tasks/**`
