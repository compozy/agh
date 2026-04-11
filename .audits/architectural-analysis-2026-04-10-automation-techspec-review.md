# Architectural Analysis Report

**Date**: 2026-04-10
**Scope**: `.compozy/tasks/automation/_techspec.md` and ADRs
**Review mode**: pre-task architecture audit

---

## Executive Summary

The automation spec is directionally strong: built-in daemon component, unified dispatch, gocron isolation, and extension seams are all reasonable. The main problem is not feature intent; it is that several critical execution boundaries are still underspecified or contradictory relative to the current AGH runtime.

The highest-risk gaps are:

1. Trigger ingestion is designed around subscription seams that do not currently exist in `session` and `memory/consolidation`.
2. The advertised global concurrency limit is only grounded in the scheduler path, not in the shared dispatcher path, so webhook/manual/event-triggered runs can bypass it.
3. Workspace identity is inconsistent across TOML, API, database schema, and current `session.CreateOpts`.
4. Webhook security is treated as a rate-limit problem, not an authentication problem.
5. TOML ownership versus runtime mutability is still ambiguous, which will create drift and confusing behavior immediately.

These should be resolved in the techspec before decomposing into implementation tasks. Otherwise the task list will either encode hidden redesign work later or incentivize workaround-shaped patches.

---

## Findings

### HIGH: Trigger sources rely on integration seams that do not exist yet, while the spec says `internal/session/` needs no changes

**Evidence**

- Techspec says the trigger engine subscribes directly via `session.Notifier`: `.compozy/tasks/automation/_techspec.md:415`
- Current `session.Notifier` is a single fan-out interface, not a subscription registry: `internal/session/interfaces.go:150`
- The daemon-owned notifier currently does nothing on `OnSessionCreated`/`OnSessionStopped`; lifecycle observation is bridged through hooks instead: `internal/daemon/hooks_bridge.go:96`
- Observer callbacks are wired from native hooks, not from additional notifier subscribers: `internal/daemon/hooks_bridge.go:326`
- The impact matrix claims `internal/session/` requires no changes: `.compozy/tasks/automation/_techspec.md:729`
- Memory consolidation callbacks are also assumed but no completion subscription surface exists on the current runtime: `.compozy/tasks/automation/_techspec.md:422`, `internal/memory/consolidation/runtime.go:38`

**Why this matters**

- The trigger engine currently has no clean place to attach for `session.*` and `memory.consolidated` events without either:
  - extending `session`/`memory` to expose a real event stream, or
  - consuming events from existing hook/observer infrastructure.
- If this is left unresolved, implementation will drift into ad hoc fan-out additions or duplicated event plumbing.

**Recommendation**

- Pick one canonical ingestion boundary for automation triggers:
  - Option A: observer/hook-driven ingestion, reusing the existing lifecycle bridge.
  - Option B: explicit typed event bus internal to daemon composition, with automation as a consumer.
- Update the impact analysis to reflect the real packages that must change.
- Add one ADR-level decision for “automation trigger ingress boundary”.

### HIGH: `max_concurrent_jobs` is not enforceable globally with the current design

**Evidence**

- Config declares a global limit: `.compozy/tasks/automation/_techspec.md:274`
- The design centralizes execution through `Dispatcher`, but the dispatcher interface has no concurrency guard or lease/token mechanism: `.compozy/tasks/automation/_techspec.md:99`
- Scheduler and trigger engine are separate activation paths into that dispatcher: `.compozy/tasks/automation/_techspec.md:27`, `.compozy/tasks/automation/_techspec.md:62`
- Build order also treats scheduler and trigger engine separately with no shared execution governor: `.compozy/tasks/automation/_techspec.md:759`

**Why this matters**

- gocron can cap scheduled job overlap, but webhook-triggered jobs, hook-triggered jobs, session-triggered jobs, and manual `jobs/:id/trigger` executions can still dispatch in parallel unless the shared dispatcher enforces the same limit.
- That breaks the “global” contract and makes the cost-control story unreliable.

**Recommendation**

- Move concurrency ownership into the dispatcher/runtime manager, not the scheduler wrapper.
- Model it as a daemon-owned execution semaphore/lease with explicit accounting for all activation paths.
- Keep gocron singleton mode for per-job overlap prevention, but do not treat it as the global concurrency solution.

### HIGH: Workspace identity is inconsistent across TOML, API, schema, and current runtime APIs

**Evidence**

- Data model and extension examples use `workspace_id`: `.compozy/tasks/automation/_techspec.md:138`, `.compozy/tasks/automation/_techspec.md:463`
- TOML and CLI examples use `workspace` as a path-like value: `.compozy/tasks/automation/_techspec.md:281`, `.compozy/tasks/automation/_techspec.md:352`
- Several examples omit workspace entirely even though schema marks it required: `.compozy/tasks/automation/_techspec.md:233`, `.compozy/tasks/automation/_techspec.md:292`
- Current session creation accepts either a workspace ID or a workspace path as separate fields: `internal/session/manager.go:37`

**Why this matters**

- This is a boundary-definition bug, not just naming drift.
- If tasks are created now, different slices of the implementation will likely invent incompatible resolution rules for path vs alias vs stored workspace ID.

**Recommendation**

- Introduce a canonical `WorkspaceRef` concept in the spec.
- Define exactly what external surfaces accept:
  - TOML/CLI: `workspace_ref` or `workspace`
  - persisted model: resolved `workspace_id` plus original `workspace_ref` if needed for diagnostics
  - extension/Host API: choose either resolved `workspace_id` only, or the same `workspace_ref` contract as humans use
- Add explicit validation and resolution semantics to the config/API sections before task decomposition.

### HIGH: Webhook security is currently a workaround, not a design

**Evidence**

- Public webhook endpoint is exposed at `POST /api/webhooks/:trigger-name`: `.compozy/tasks/automation/_techspec.md:335`
- Route behavior only checks trigger existence/type/enabled state: `.compozy/tasks/automation/_techspec.md:438`
- Known risk says unauthenticated webhook abuse is mitigated by fire limits, with HMAC deferred to “future”: `.compozy/tasks/automation/_techspec.md:846`

**Why this matters**

- Fire limits reduce blast radius after abuse begins; they do not authenticate the caller or protect against malicious prompt injection and unwanted agent execution.
- This is exactly the kind of symptom-patch that the no-workarounds constraint is meant to reject.

**Recommendation**

- Define v1 webhook authentication now.
- Minimum acceptable options:
  - per-trigger secret in header or query parameter plus constant-time comparison
  - HMAC signature over body with timestamp
  - explicit statement that webhook routes are localhost-only in v1, if that is the real product decision
- Task creation should not proceed until the ownership and auth story is explicit.

### MEDIUM: TOML source-of-truth semantics still conflict with runtime mutability

**Evidence**

- Spec says TOML jobs are source-of-truth and resync on boot: `.compozy/tasks/automation/_techspec.md:60`, `.compozy/tasks/automation/_techspec.md:834`
- API still allows patch/delete-style operations, with config jobs “only disabled”: `.compozy/tasks/automation/_techspec.md:316`
- Known risk mitigation is just logging a warning when TOML overwrites runtime changes: `.compozy/tasks/automation/_techspec.md:844`

**Why this matters**

- “Warn and overwrite later” is not a behavior model.
- If disabling a config job via API is allowed, the spec must state whether that disabled state:
  - persists across restart as an overlay,
  - is immediately rejected,
  - or is ephemeral until next sync.

**Recommendation**

- Pick one of these models explicitly:
  - strict TOML ownership: config-sourced jobs are read-only through API except manual trigger
  - overlay model: runtime state stores an override layer separate from config materialization
  - promote-to-dynamic workflow: API mutation clones config job into dynamic ownership
- This needs to be tasked before CRUD handlers, not discovered during implementation.

### MEDIUM: Fire-limit safety disappears on daemon restart

**Evidence**

- Fire limits are positioned as the main runaway-execution safety mechanism: `.compozy/tasks/automation/adrs/adr-004.md:17`
- ADR explicitly says tracking is in-memory and resets on daemon restart: `.compozy/tasks/automation/adrs/adr-004.md:71`

**Why this matters**

- Restarting the daemon reopens the full firing budget, which weakens the exact control meant to cap expensive LLM-backed automation.
- Because runs are already persisted in SQLite, making the safety invariant ephemeral feels like implementation convenience, not a principled boundary.

**Recommendation**

- Compute fire-limit windows from persisted recent runs, or persist just the rolling-window counters/checkpoints in SQLite.
- If you intentionally keep this ephemeral for v1, the spec should explicitly downgrade fire limits from “safety net” to “best-effort local throttle”.

### MEDIUM: The schema makes names globally unique even though automation is modeled as workspace-scoped

**Evidence**

- Jobs and triggers both carry `workspace_id`: `.compozy/tasks/automation/_techspec.md:138`, `.compozy/tasks/automation/_techspec.md:160`
- API filtering is workspace-aware: `.compozy/tasks/automation/_techspec.md:519`, `.compozy/tasks/automation/_techspec.md:526`
- But schema uses global `UNIQUE(name)` for both tables: `.compozy/tasks/automation/_techspec.md:216`, `.compozy/tasks/automation/_techspec.md:231`

**Why this matters**

- Two workspaces should be able to have a `daily-report` automation independently.
- Global uniqueness will cause surprising collisions and awkward naming hacks across unrelated repos.

**Recommendation**

- Change uniqueness to `(workspace_id, name)` if automation is workspace-bound.
- If global jobs are also allowed, the spec needs an explicit scope column and uniqueness rules per scope.

### MEDIUM: Trigger prompt/filter semantics are too stringly typed and under-validated

**Evidence**

- Filter model is `map[string]string`: `.compozy/tasks/automation/_techspec.md:163`
- Prompt templating uses raw `text/template` examples with nested payload access: `.compozy/tasks/automation/_techspec.md:297`, `.compozy/tasks/automation/_techspec.md:700`
- Validation section mentions parsing config and rendering templates, but not strict missing-key or schema validation: `.compozy/tasks/automation/_techspec.md:743`

**Why this matters**

- Without a defined event payload contract and strict template behavior, the system will silently accept broken filters/templates and fail only at fire time.
- That creates operational drift and pushes bugs into runtime.

**Recommendation**

- Define event payload envelopes per built-in event.
- Require template compilation and execution with `missingkey=error`.
- Define filter semantics explicitly:
  - top-level envelope keys only, or
  - dotted-path selectors with exact-match comparison.

### LOW: The spec and ADRs are internally inconsistent about the “unified job” model

**Evidence**

- ADR-002 says schedules and triggers use the same `Job` definition: `.compozy/tasks/automation/adrs/adr-002.md:19`
- The techspec defines separate `Job` and `Trigger` models and separate tables: `.compozy/tasks/automation/_techspec.md:134`, `.compozy/tasks/automation/_techspec.md:156`, `.compozy/tasks/automation/_techspec.md:214`

**Why this matters**

- This is mostly a task-shaping problem: implementers will not know whether the abstraction goal is “shared dispatch only” or “single persisted automation model”.

**Recommendation**

- Resolve the language:
  - either revise ADR-002 to say “shared dispatch, separate persisted models”
  - or redesign the techspec around a single `AutomationDefinition` entity with two activation modes

---

## Recommended Task-Shaping Changes Before Decomposition

1. Add a short “Boundary Decisions” section to the techspec covering:
   - canonical event ingress boundary
   - canonical workspace reference model
   - concurrency governor ownership
   - config ownership/override semantics
   - webhook auth model

2. Split implementation tasks by dependency order only after the above is clarified:
   - boundary/event ingress
   - persistence/config ownership
   - dispatcher execution controls
   - scheduler and trigger adapters
   - transport surfaces

3. Add one explicit task for spec correction, not just implementation:
   - update contradictory examples
   - align ADR-002 wording with actual data model
   - harden the risk section to reflect real v1 guarantees

---

## Conclusion

This automation spec is close enough to proceed, but not yet stable enough to decompose safely into implementation tasks. The main blockers are boundary clarity and invariant ownership, not library choice or package naming.

If those are fixed first, the later tasks can stay clean and additive. If not, the task list will encode hidden redesign work and invite workaround-driven implementation.
