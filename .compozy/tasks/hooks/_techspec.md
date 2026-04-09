# TechSpec: Lifecycle Hooks Platform

## Executive Summary

This TechSpec introduces a first-class hooks platform for AGH. The platform defines a typed lifecycle taxonomy, a centralized dispatcher with per-event type-safe functions, and a multi-source declaration model so extensions can observe, block, enrich, and transform runtime operations without changing core packages for each new capability.

The implementation strategy is to create a dedicated `internal/hooks` package that exposes typed dispatch functions (not a generic event bus), uses Go generics internally for shared infrastructure, and wires into `internal/session`, `internal/daemon`, `internal/skills`, `internal/config`, and future tool and permission paths. The primary trade-off is additional runtime and configuration complexity in exchange for a stable extensibility contract that matches AGH's documented platform ambitions. Because AGH is greenfield alpha, the system should define the full contract now rather than evolve through incompatible one-off seams.

The dispatcher replaces the existing `notifierFanout` by implementing the `session.Notifier` interface, unifying hook dispatch and session notification into a single path.

## System Architecture

### Component Overview

- `internal/hooks`
  - `Hooks`: main struct owning the registry, async worker pool, and typed dispatch functions. Implements `session.Notifier`.
  - `Registry`: hot-reloadable registry using `sync.RWMutex` with build-then-swap semantics (same pattern as `skills.Registry`). Stores pre-sorted `map[HookEvent][]*ResolvedHook` snapshots.
  - `pipeline[P, R]`: generic internal type that executes sync hooks as a sequential pipeline and schedules async hooks to the worker pool. Each typed dispatch function instantiates a concrete pipeline.
  - `Executors`: native callback executor, subprocess executor, and a future Wasm-ready executor seam.
  - `Matchers`: event-specific filtering by tool name, agent name/type, workspace, session type, message kind, and provider.
  - `Telemetry`: emits structured hook lifecycle records into `internal/observe`, including patch audit trail for security-relevant families.
  - `Worker pool`: fixed-size goroutine pool (stdlib channel + WaitGroup) for async hook execution with bounded shutdown.
- `internal/skills`
  - Parses `metadata.agh.hooks` into typed hook declarations; no longer owns dispatch.
- `internal/config`
  - Parses hook declarations from policy, user, and workspace config and feeds them into the registry.
- `internal/session`
  - Receives the `Hooks` dispatcher as its `Notifier`. Calls typed dispatch functions at session, input/prompt, event recording, agent lifecycle, and future turn/message seams.
- `internal/daemon`
  - Composition root. Creates the `Hooks` dispatcher, wires native hooks, declaration providers, executor implementations, and reload triggers. Replaces `notifierFanout` and `skillsHookDispatcher`.
- `internal/tools` and permission flow
  - Integrate with `tool.*` and `permission.*` hook families through the same typed dispatch functions.

### Hook Taxonomy

Events are classified as **sync-eligible** or **async-only**. Sync-eligible events accept both sync and async hooks. Async-only events reject sync hook registration.

#### Sync-Eligible Events

- `session.pre_create`, `session.post_create`
- `session.pre_resume`, `session.post_resume`
- `session.pre_stop`, `session.post_stop`
- `input.pre_submit`
- `prompt.post_assemble`
- `agent.pre_start`, `agent.spawned`, `agent.crashed`, `agent.stopped`
- `turn.start`, `turn.end`
- `message.start`, `message.end`
- `tool.pre_call`, `tool.post_call`, `tool.post_error`
- `permission.request`
- `context.pre_compact`, `context.post_compact`

#### Async-Only Events

- `event.pre_record`, `event.post_record`
- `message.delta`
- `permission.resolved`, `permission.denied`

Rationale: async-only events are either high-frequency (`message.delta`, `event.*`) where subprocess fork/exec per invocation is a denial-of-service, or post-decision observations (`permission.resolved`, `permission.denied`) where mutation is semantically invalid. Note: `event.pre_record` and `event.post_record` use "pre/post" naming to indicate timing relative to the record operation, but as async-only events they are observation-only — hooks cannot mutate or block the record.

### Dispatch Model

- **Sync hooks compose as a sequential pipeline.** Each sync hook receives the payload as modified by all previous hooks in the chain. Hook A receives the original, returns a patch. The patch is applied, producing a modified payload. Hook B receives the modified payload. This continues through all sync hooks in order.
- **Pipeline short-circuit**: an explicit deny from any hook stops the pipeline. A `required` hook that fails (error or timeout) stops the pipeline with an error. A non-required hook that fails is skipped.
- **Async hooks are observational/background hooks.** They may emit side effects and telemetry but may not block or mutate the completed primary operation. Async hooks are submitted to the worker pool after sync pipeline completion.
- `required` is valid only for sync hooks on sync-eligible events.
- **Dispatch depth guard**: a counter in `context.Context` tracks dispatch nesting. Max depth is 3. Exceeding it returns an error immediately — prevents circular dispatch (e.g., a hook on `event.pre_record` triggering `recordEvent` which fires `event.pre_record` again).
- **Permission invariant**: hooks in the `permission.*` family may observe, enrich context, or deny — but may **never** upgrade a deny to an allow. This is enforced in code by the dispatcher, not by documentation. Any patch attempting deny→allow is rejected and logged as `hook.dispatch.permission_escalation_blocked`.
- Dispatch order is deterministic:
  1. Go-native hooks
  2. settings/config hooks
  3. agent-definition hooks
  4. skill hooks
- Within each source class:
  1. higher priority runs first (descending)
  2. stable name order (ascending lexicographic)
- Default priorities by source: native=1000, config=500, agent-definition=100, skill=0.
- Skill hooks preserve existing skill-source precedence (Bundled → Marketplace → User → Additional → Workspace) as sub-ordering before name.

## Implementation Design

### Core Interfaces

```go
type RegisteredHook struct {
    Name     string
    Event    HookEvent
    Source   HookSource
    Mode     HookMode // sync | async
    Required bool
    Priority int
    Timeout  time.Duration
    Matcher  HookMatcher
    Executor Executor
}
```

```go
type Executor interface {
    Kind() HookExecutorKind
    Execute(ctx context.Context, hook RegisteredHook, payload []byte) ([]byte, error)
}
```

```go
// Generic internal pipeline — package-private
type pipeline[P any, R any] struct {
    hooks  *Hooks
    event  HookEvent
    apply  func(P, R) P       // typed patch applicator
    encode func(P) ([]byte, error)  // serialize payload for subprocess/Wasm executors
    decode func([]byte) (R, error)  // deserialize patch from subprocess/Wasm executors
}
```

Each typed dispatch function provides concrete `encode`/`decode` functions at initialization. Native Go executors bypass serialization entirely — they receive the typed payload directly via a type-safe callback. The `[]byte` boundary exists only for subprocess and future Wasm executors.

```go
// Typed public dispatch functions — one per event
func (h *Hooks) DispatchSessionPreCreate(ctx context.Context, payload SessionPreCreatePayload) (SessionPreCreatePayload, error)
func (h *Hooks) DispatchToolPreCall(ctx context.Context, payload ToolPreCallPayload) (ToolCallPatch, error)
func (h *Hooks) DispatchPromptPostAssemble(ctx context.Context, payload PromptPayload) (PromptPayload, error)
// ... one function per event in the taxonomy
```

```go
// Hooks implements session.Notifier
var _ session.Notifier = (*Hooks)(nil)

func (h *Hooks) OnSessionCreated(ctx context.Context, session *session.Session)
func (h *Hooks) OnSessionStopped(ctx context.Context, session *session.Session)
func (h *Hooks) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent)
```

### Data Models

- `HookDecl`
  - declarative hook source record from config, agent definitions, or skills
- `RegisteredHook`
  - normalized hook ready for dispatch, with resolved source, mode, matcher, timeout, and executor binding
- `ResolvedHook`
  - pre-sorted hook in the registry snapshot, with executor and matcher pre-compiled
- `HookRunRecord`
  - persisted observability record: hook name, event, source, mode, duration, outcome, dispatch depth, and `PatchApplied json.RawMessage` (populated for security-relevant families: `permission.*`, `prompt.*`, `tool.*`, `input.*`)
- Event-specific payload and patch types (one pair per event):
  - `SessionPreCreatePayload` / `SessionCreatePatch`
  - `PromptPayload` / `PromptPatch`
  - `ToolPreCallPayload` / `ToolCallPatch`
  - `ToolPostCallPayload` / `ToolResultPatch`
  - `InputPreSubmitPayload` / `InputPreSubmitPatch`
  - `ContextCompactPayload` / `ContextCompactionPatch`
  - Other events use payload-only types (no patch) when they are async-only or observation-only.

### Declaration Model

- Go-native declarations are registered from the composition root.
- Settings/config declarations are loaded from existing AGH config layers and may be used for organizational policy.
- Agent-definition declarations are scoped to one agent type and execute only for matching sessions.
- Skill declarations remain the portable, user-facing mechanism for reusable procedures and automation.
- The declaration schema supports:
  - `name`
  - `event`
  - `mode` (sync | async — validated against event eligibility)
  - `required` (valid only for sync hooks)
  - `priority` (default: 0 for skills, 100 for agent-definitions, 500 for config)
  - `timeout`
  - `matcher`
  - `executor`
  - `metadata` for observability labels

### Matcher Model

- `session.*`: session type, workspace id/root, agent name
- `input/prompt.*`: agent name, workspace id/root, input class
- `event.*`: ACP event type, turn id, agent name
- `tool.*`: tool name, namespace, read-only flag
- `permission.*`: tool name, decision class
- `message.*`: message role and delta type
- `context.*`: compaction reason and strategy

### Hot Reload

The registry uses `sync.RWMutex` with build-then-swap semantics, following the same pattern as `skills.Registry`:

- **Read path**: `RLock`, copy slice reference for target event, `RUnlock`, dispatch against snapshot. Zero allocations in critical section.
- **Write path**: `Rebuild(ctx)` reads all 4 sources, validates declarations, builds pre-sorted snapshot map, then `Lock` + swap + `Unlock`. If validation fails, old snapshot stays.
- **Triggers**: `skills.Watcher` notifies on skill changes. Config and agent-definition changes use analogous watch or explicit refresh.
- **Version counter**: `atomic.Int64` bumped on each swap for staleness detection.
- **Consistency**: in-flight dispatches operate on the snapshot they read — concurrent reloads do not affect them.

### Async Worker Pool

Async hooks execute in a fixed-size worker pool using Go stdlib primitives:

- **Pool size**: configurable, default 4 workers
- **Queue**: buffered channel, configurable capacity, default 64
- **Backpressure**: non-blocking send with `select`/`default` — full buffer drops the hook with structured log `hook.dispatch.async_dropped` and metric
- **Workers**: `select { case task := <-ch: execute(task) case <-ctx.Done(): return }`
- **Shutdown**: close channel, workers drain with deadline (10s), `sync.WaitGroup.Wait()`
- **Ownership**: `Hooks` struct owns the pool, starts on init, joins on `Close()`
- **Panic recovery**: each worker wraps execution in `recover()` to prevent a panicking hook from killing the pool
- **Shutdown ordering**: `Hooks.Close()` runs **after** session manager shutdown (so `session.post_stop` hooks can fire during session teardown) and **before** database close (so async hooks that write telemetry can complete). In the daemon shutdown sequence: stop sessions → `Hooks.Close()` (drain async pool) → close HTTP/UDS servers → close database → release lock.

### API Endpoints

- `GET /api/hooks/catalog?workspace=:id&agent=:name`
  - Returns resolved active hooks after precedence, matching defaults, and source attribution. Shows the sequential pipeline order.
- `GET /api/hooks/runs?session=:id&event=:event`
  - Returns recent hook execution records including patch diffs for security-relevant families.
- `GET /api/hooks/events`
  - Returns the supported hook taxonomy with sync eligibility classification and event-specific payload/patch schema names.

## Integration Points

- Local subprocess execution for skill/config/agent shell hooks
- Existing `internal/observe` pipeline for run records and metrics
- Existing `internal/session` lifecycle — `Hooks` replaces `notifierFanout` as the `session.Notifier`
- Existing `internal/skills` watcher for reload triggers
- Future Wasm executor seam — same `Executor` contract, no taxonomy change

## Migration from Current Hooks Implementation

AGH is greenfield alpha with zero legacy tolerance. The migration is a hard cut-over — delete old code, replace with new. No compatibility shims, no mapping layers, no deprecation period.

### Code to Delete

| File / Symbol | Current Role | Replacement |
|--------------|-------------|-------------|
| `internal/skills/hooks.go` — `HookRunner`, `RunHooks`, `runHook`, `orderSkillsForHooks`, `skillHasHookEvent`, `hookCapture*` | Subprocess hook execution, ordering, output capture | `internal/hooks` subprocess executor + `pipeline[P,R]` ordering |
| `internal/skills/types.go` — `HookDecl`, `HookEvent`, `HookPayload`, `HookResult` (hook-related types only) | Hook declaration and execution types | `internal/hooks` typed declarations and per-event payload/patch types |
| `internal/skills/registry.go` — `cloneHookDecls` | Deep-copy helper for old HookDecl | Not needed — new declarations are handled by the hooks registry |
| `internal/daemon/notifier.go` — `notifierFanout`, `skillsHookDispatcher`, `sessionHookPhase` | Session notification fanout and hook dispatch bridge | `internal/hooks.Hooks` implementing `session.Notifier` |
| `internal/skills/hooks_test.go` | Tests for old HookRunner | New tests in `internal/hooks` |
| Related test helpers in `registry_test.go` | `newSkillWithHook`, hook-related assertions | Updated to use new declaration types |

### Event Name Migration

Old event names are deleted, not mapped:

| Old Name | New Name |
|----------|----------|
| `on_session_created` | `session.post_create` |
| `on_session_stopped` | `session.post_stop` |

Existing skill YAML frontmatter using old names will fail validation at parse time with a clear error message pointing to the new name. The `validHookEvent` function in the skill loader is rewritten to accept only the new dotted taxonomy.

### Skill Declaration Schema Migration

Old schema (5 fields):
```yaml
hooks:
  - event: on_session_created
    command: ./setup.sh
    args: ["--init"]
    timeout: 5s
    env:
      KEY: value
```

New schema (up to 9 fields, backward-compatible subset):
```yaml
hooks:
  - event: session.post_create
    command: ./setup.sh
    args: ["--init"]
    timeout: 5s
    env:
      KEY: value
    # New optional fields:
    mode: async          # default: async for skill hooks
    priority: 0          # default: 0 for skills
    matcher:             # optional — narrows when hook fires
      agent_name: claude
```

The minimal migration for existing skills: change `on_session_created` → `session.post_create` and `on_session_stopped` → `session.post_stop`. All other fields remain compatible. New fields are optional with sensible defaults.

### Subprocess Payload Migration

Old payload (JSON via stdin):
```json
{"session_id": "...", "agent_name": "...", "workspace": "...", "event": "on_session_created"}
```

New payload (JSON via stdin, event-specific):
```json
{"session_id": "...", "agent_name": "...", "workspace": "...", "event": "session.post_create", "session_type": "...", "workspace_id": "..."}
```

The new payload is a superset — it includes all old fields plus event-specific fields. Existing subprocess scripts that read `session_id`, `agent_name`, and `workspace` will continue to work without changes. The `event` field changes from `on_session_created` to `session.post_create`.

### Migration Sequencing

The migration happens atomically in build order step 10 (daemon wiring):
1. Steps 1-6: build the new `internal/hooks` package (no old code touched yet)
2. Step 7: rewrite `internal/skills` hook parsing to emit new `HookDecl` types, delete old types and `HookRunner`
3. Step 10: delete `notifierFanout` and `skillsHookDispatcher`, wire `Hooks` as `session.Notifier`

There is no transitional state where both old and new dispatch paths coexist.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/hooks` | new | Central platform package; medium design risk | Implement registry, pipeline, typed dispatch functions, worker pool, executors |
| `internal/skills` | modified | Moves from owning dispatch to supplying declarations; low risk | Parse richer hook schema and export declarations |
| `internal/config` | modified | Adds hook config parsing and validation; medium risk | Extend config schema and precedence handling |
| `internal/session` | modified | Receives `Hooks` as `Notifier`, calls typed dispatch at lifecycle points; medium risk | Wire typed dispatch invocations at session/input/event/agent paths |
| `internal/daemon` | modified | Replaces `notifierFanout` and `skillsHookDispatcher` with `Hooks`; medium risk | Compose Hooks, wire reload triggers, update shutdown sequence |
| `internal/observe` | modified | Stores hook execution telemetry with patch audit; low risk | Add HookRunRecord with PatchApplied field |
| `internal/tools` | future-modified | Consumes `tool.*` hooks when tool registry lands; medium risk | Reuse typed dispatch functions |
| web/API clients | modified | Optional hook introspection UI; low risk | Consume catalog/run endpoints if needed |

## Testing Approach

### Unit Tests

- Declaration parsing from all four sources
- Matcher evaluation for each hook family
- Ordering: source → priority → name (no specificity)
- Sequential pipeline patch composition: verify hook N sees output of hook N-1
- Pipeline short-circuit on deny and required-hook failure
- Async worker pool: submission, backpressure drop, graceful shutdown with drain
- Permission invariant: deny→allow escalation is rejected for every source type
- Event eligibility: sync registration rejected for async-only events
- Dispatch depth guard: depth > 3 returns error
- Patch audit trail: security-relevant families persist patches, others do not (unless debug mode)
- Registry hot reload: concurrent read during swap returns consistent snapshot
- Invalid contract cases:
  - `required` on async hooks
  - sync mode on async-only events
  - patch type mismatch
  - unknown event
  - illegal matcher fields for event family

### Integration Tests

- Session create/resume/stop hook flow with real SQLite and subprocess executors
- Sequential pipeline with multiple hooks patching the same field — verify composition order
- Prompt submit + event record hook chains
- Agent crash classification and crash-hook dispatch
- Async hook execution with bounded shutdown and observability emission
- Config + agent + skill source coexistence in one workspace
- Hot reload: add/remove skill mid-session, verify next dispatch uses updated registry
- Permission escalation attempt: verify deny→allow is blocked end-to-end
- Dispatch depth: hook triggering event that fires same hook — verify depth guard
- Hook introspection HTTP endpoints with patch audit data
- Tool and permission family integration tests once corresponding runtime path exists in the same implementation stream

## Development Sequencing

### Build Order

1. Create `internal/hooks` core types: `HookEvent` enum with sync eligibility, `HookSource`, `HookMode`, `RegisteredHook`, event-specific payload/patch types — no dependencies
2. Implement declaration normalization, matcher evaluation, and ordering (source → priority → name) — depends on step 1
3. Implement executor contracts and native/subprocess executors — depends on step 1
4. Implement generic `pipeline[P, R]` with sequential sync composition, deny short-circuit, depth guard, and permission invariant — depends on steps 2 and 3
5. Implement async worker pool (channel + goroutines + WaitGroup) — depends on step 1
6. Implement `Hooks` struct with typed dispatch functions, registry with RWMutex snapshot swap, and `session.Notifier` implementation — depends on steps 4 and 5
7. Extend `internal/skills` hook parsing to emit rich `HookDecl` with new schema fields — depends on step 1
8. Extend `internal/config` with hook declarations and validation — depends on step 1
9. Extend agent-definition loading to emit hook declarations — depends on step 1
10. Wire `Hooks` in `internal/daemon`: hard cut-over — delete `notifierFanout` and `skillsHookDispatcher`, wire `Hooks` as `session.Notifier`, connect reload triggers from skills watcher, update shutdown sequence (stop sessions → `Hooks.Close()` → close servers → close DB) — depends on steps 6, 7, 8, and 9
11. Integrate `session.*`, `input.*`, `prompt.*`, `event.*`, and `agent.*` dispatch points in session manager — depends on steps 6 and 10
12. Add `turn.*` and `message.*` dispatch from normalized ACP event flow — depends on step 11
13. Add `context.*` dispatch around compaction paths — depends on step 11
14. Add `tool.*` and `permission.*` integrations through the typed dispatch functions — depends on step 6 and the corresponding tool/permission runtime paths
15. Add hook observability storage (HookRunRecord with patch audit) and HTTP introspection endpoints — depends on steps 10 through 14
16. Complete full-package verification and cross-source integration tests — depends on all previous steps

### Technical Dependencies

- Existing config precedence rules in `internal/config`
- Existing session lifecycle and `session.Notifier` interface
- Existing `skills.Registry` pattern (RWMutex + snapshot swap) and `skills.Watcher`
- Existing observe/event store primitives
- Future tool registry and permission pipeline for `tool.*` and `permission.*` execution points

## Monitoring and Observability

- Metrics
  - hook dispatch count by event, source, mode, outcome
  - sync hook wall-clock latency (per-hook and full pipeline)
  - async queue depth and drain time
  - async hook drop count (`hook.dispatch.async_dropped`)
  - hook block/deny count
  - hook timeout count
  - permission escalation block count
  - dispatch depth violations count
  - registry reload count and duration
- Structured logs
  - `hook.dispatch.started` — includes dispatch depth
  - `hook.dispatch.completed` — includes pipeline trace
  - `hook.dispatch.blocked` — includes deny source
  - `hook.dispatch.failed` — includes error and required status
  - `hook.dispatch.async_dropped` — includes queue depth at time of drop
  - `hook.dispatch.permission_escalation_blocked` — security event
  - `hook.dispatch.depth_exceeded` — includes event chain
  - `hook.registry.reloaded` — includes version, hook count delta
- Alerting thresholds
  - repeated required-hook failure
  - async queue backlog above threshold
  - p95 sync dispatch latency above configured budget
  - permission escalation attempts (any occurrence)
  - dispatch depth violations (any occurrence)

## Technical Considerations

### Key Decisions

- The platform uses a dedicated `internal/hooks` package instead of embedding platform behavior in `internal/skills`. (ADR-001)
- The taxonomy is broad because AGH's docs position hooks as foundational extensibility infrastructure. (ADR-002)
- Sync hooks are the only mutation/blocking hooks; async hooks are side-effect/observer hooks. Mutation is event-specific and typed. (ADR-003)
- Declarations come from Go-native callbacks, settings/config, agent definitions, and skills. (ADR-004)
- The public API uses typed per-event dispatch functions, not a generic `Dispatch(ctx, any)`. Internally, Go generics share infrastructure without `any`. (ADR-005, ADR-007)
- Sync hooks compose as a sequential pipeline — each hook sees the output of the previous. (ADR-006)
- Async hooks execute in a stdlib worker pool with bounded channel and graceful shutdown. (ADR-008)
- Permission hooks are deny-only — the dispatcher rejects any deny→allow escalation. (ADR-009)
- Patch audit trail is persisted for security-relevant families (`permission.*`, `prompt.*`, `tool.*`, `input.*`). (ADR-010)
- Ordering uses source → priority → name. Specificity is removed. (ADR-011)
- Events are classified as sync-eligible or async-only. A dispatch depth guard (max 3) prevents circular dispatch. (ADR-012)
- The registry is hot-reloadable using RWMutex + snapshot swap. The dispatcher replaces `notifierFanout` by implementing `session.Notifier`. (ADR-013)

### Known Risks

- Overly broad first implementation could stall delivery
  - Mitigation: strict build order and package-local milestones
- Tool and permission hook families depend on adjacent runtime work
  - Mitigation: implement shared dispatcher now and integrate those families as part of the same program of work
- Async hooks can become silent failure sinks
  - Mitigation: explicit telemetry, queue limits, drop metrics, and shutdown draining rules
- Sequential pipeline makes ordering load-bearing — later hooks can overwrite earlier patches
  - Mitigation: introspection API exposes full pipeline trace with before/after state per hook, patch audit trail persists forensic records
- Hot reload introduces concurrency between dispatch and registry swap
  - Mitigation: RWMutex + snapshot isolation ensures in-flight dispatches are never affected by concurrent reloads
- Generic internals add moderate code complexity
  - Mitigation: generics are limited to the `pipeline[P, R]` type and executor interface — the public API is fully concrete

## Architecture Decision Records

- [ADR-001: Centralize Hooks in internal/hooks](adrs/adr-001.md) — Creates one typed registry and dispatcher for every hook family and source.
- [ADR-002: Use a Dotted Hook Taxonomy with Rich Families](adrs/adr-002.md) — Defines the full extensibility-facing lifecycle taxonomy.
- [ADR-003: Use Typed Patch Protocols and Hybrid Failure Policy](adrs/adr-003.md) — Limits mutation to typed patch surfaces and keeps fail-closed behavior explicit via `required`.
- [ADR-004: Support Four Declaration Sources with Ordered Dispatch](adrs/adr-004.md) — Combines native, config, agent, and skill hooks into one deterministic dispatch model.
- [ADR-005: Use Typed Per-Event Dispatch Functions Instead of Generic Dispatcher](adrs/adr-005.md) — Replaces the generic `Dispatch(ctx, HookInvocation)` with typed functions. Resolves the "event bus" contradiction.
- [ADR-006: Sequential Pipeline for Sync Hook Patch Composition](adrs/adr-006.md) — Each sync hook sees the output of the previous, Kubernetes-style.
- [ADR-007: Use Go Generics for Internal Dispatcher Type Safety](adrs/adr-007.md) — Shares infrastructure without `any` via `pipeline[P, R]`.
- [ADR-008: Stdlib Worker Pool for Async Hook Execution](adrs/adr-008.md) — Channel + goroutines + WaitGroup, following `consolidation.Runtime` pattern.
- [ADR-009: Permission Hooks Are Deny-Only](adrs/adr-009.md) — Dispatcher rejects deny→allow escalation. Architecturally impossible, not just discouraged.
- [ADR-010: Persist Patch Audit Trail for Security-Relevant Families](adrs/adr-010.md) — HookRunRecord stores patches for `permission.*`, `prompt.*`, `tool.*`, `input.*`.
- [ADR-011: Simplify Ordering to Source, Priority, Name](adrs/adr-011.md) — Removes undefined specificity sort key. Supersedes ordering details in ADR-004.
- [ADR-012: Classify Events into Sync-Eligible and Async-Only with Dispatch Depth Guard](adrs/adr-012.md) — Prevents subprocess fork-bomb on `message.delta` and circular dispatch stack overflow.
- [ADR-013: Hot-Reloadable Registry with RWMutex Snapshot Swap](adrs/adr-013.md) — Same pattern as `skills.Registry`. Dispatcher replaces `notifierFanout`.
