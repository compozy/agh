# TechSpec: Harness Runtime Architecture

## Executive Summary

This TechSpec defines the next-generation internal harness architecture for AGH.
The goal is to make prompt construction, turn-time augmentation, synthetic
reentry, and detached async execution explicit, testable, and observable
without introducing a parallel runtime model that conflicts with the daemon’s
current composition.

The revised design intentionally changes two earlier assumptions:

1. `HarnessProfile` is no longer the foundational primitive. Harness behavior
   is resolved from:
   - durable session context
   - turn origin
   - runtime signals such as network metadata and synthetic reentry metadata

2. Detached async harness work does not introduce a new persisted
   `BackgroundRun` subsystem. It reuses the existing task runtime and
   `task_runs` persistence surface as the durable substrate for async work and
   wake-up semantics.

This remains an internal-only runtime initiative. No user-facing configuration
surface is introduced in this phase.

## System Architecture

### Core Model

Harness behavior is resolved from three layers:

- **Durable Session Context**
  - session type
  - channel presence
  - workspace binding
  - agent/session ownership metadata

- **Turn Context**
  - `TurnOriginUser`
  - `TurnOriginNetwork`
  - `TurnOriginSynthetic`

- **Resolved Harness Policy**
  - startup prompt sections to include
  - turn augmenters to apply
  - observability annotations to emit
  - detached async behavior rules
  - reentry behavior rules

“Profile” may still appear as a derived internal label for logging or debugging,
but it is not the source-of-truth abstraction.

### Workstreams

This TechSpec is organized into six workstreams:

1. Harness Context Resolution
2. Startup Prompt Architecture
3. Turn Augmentation Pipeline
4. Synthetic Reentry Model
5. Detached Async Runtime on Task Infrastructure
6. Storage, Observability, and Verification

Each workstream is independently specifiable and later decomposable into tasks,
but the full document describes the final integrated architecture.

## Workstream 1: Harness Context Resolution

### Problem

The current draft over-centered runtime behavior around `HarnessProfile`, but
AGH already has separate durable and turn-time axes in code:

- session durability/type in [session.go](/Users/pedronauck/Dev/compozy/agh/internal/session/session.go:30)
- turn provenance in [interfaces.go](/Users/pedronauck/Dev/compozy/agh/internal/session/interfaces.go:18)

Collapsing those into one enum creates ambiguous cases such as:

- user session receiving a network turn
- system session receiving synthetic reentry
- network-bound system session with later user-driven follow-up

### Decision

Introduce an internal `HarnessContextResolver` that produces a
`ResolvedHarnessPolicy` from:

- session type
- channel presence
- prompt metadata
- turn origin
- synthetic reentry metadata
- detached-run/task metadata when applicable

### Core Types

```go
type TurnOrigin string

const (
    TurnOriginUser      TurnOrigin = "user"
    TurnOriginNetwork   TurnOrigin = "network"
    TurnOriginSynthetic TurnOrigin = "synthetic"
)

type ResolvedHarnessPolicy struct {
    SessionClass      SessionClass
    TurnOrigin        TurnOrigin
    IncludeSections   []string
    EnableAugmenters  []string
    ReentryMode       ReentryMode
    DetachedRunMode   DetachedRunMode
    ObservabilityTags map[string]string
}
```

`SessionClass` is internal and derived from durable session properties. It is
not a user-facing config concept.

### Required Behavior

- Resolution must be deterministic.
- Resolution must be daemon-owned.
- The TechSpec must include a matrix that enumerates valid combinations of:
  - session type
  - channel presence
  - turn origin
  - resulting startup sections
  - resulting turn augmenters

### Implementation Home

The resolver belongs in `internal/daemon`, not `internal/session`.

Suggested file:
- `internal/daemon/harness_context.go`

## Workstream 2: Startup Prompt Architecture

### Problem

Startup prompt assembly is already split between the session manager and the
daemon, but section inclusion is currently global/workspace-driven rather than
runtime-policy-driven:

- assembler composition in [composed_assembler.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/composed_assembler.go:65)
- provider boot wiring in [boot.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/boot.go:247)
- startup overlay of `agh-network` in [manager_start.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_start.go:225)

### Decision

Keep the existing assembler/provider chain, but insert a daemon-owned
`SectionSelector` before final assembly.

### New Model

Each prompt section must have metadata:

```go
type PromptSectionDescriptor struct {
    Name      string
    Position  string
    Budget    int
    Provider  session.PromptProvider
    Predicate SectionPredicate
}
```

`SectionPredicate` is evaluated against `ResolvedHarnessPolicy`.

### Required Behavior

- The selector must decide section inclusion before final prompt concatenation.
- Section ordering must be deterministic.
- Section budgets must be declared and testable.
- `agh-network` must be modeled explicitly as a startup section behavior, not
  left as an undocumented special case.

### Implementation Home

Suggested files:
- `internal/daemon/section_selector.go`
- `internal/daemon/prompt_sections.go`

### Explicit Handling of Network Startup Behavior

The current bundled network skill in [manager_network_skill.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_network_skill.go:12)
must be rehomed intentionally into one of these forms:

- assembler-managed startup section with `TurnOriginNetwork`/channel
  eligibility
- daemon-selected startup overlay provider

It must not remain an undocumented inline exception.

## Workstream 3: Turn Augmentation Pipeline

### Problem

Turn-time augmentation currently exists as a single callback seam:

- augmenter type in [interfaces.go](/Users/pedronauck/Dev/compozy/agh/internal/session/interfaces.go:43)
- prompt dispatch in [manager_prompt.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_prompt.go:95)
- current memory augmenter in [recall.go](/Users/pedronauck/Dev/compozy/agh/internal/memory/recall.go:17)

The architecture needs ordered augmentation, but the implementation should not
require an immediate breaking rewrite of `session.Manager`.

### Decision

Stage the augmentation pipeline in two steps:

1. Introduce daemon-side composition over the current single augmenter seam.
2. Later, if necessary, promote that composition into a richer typed
   session-layer interface.

### Phase-1 Runtime Shape

```go
type CompositePromptInputAugmenter struct {
    augmenters []PromptInputAugmenter
}
```

This is injected from `internal/daemon`, preserving the current manager seam in
[daemon.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/daemon.go:495).

### Phase-2 Target Interface

```go
type TurnAugmenter interface {
    Name() string
    Priority() int
    Critical() bool
    Augment(ctx context.Context, session *Session, input PromptInput) (PromptInput, error)
}
```

### Required Behavior

- Augmenters execute in deterministic order.
- Budget must be aggregated across all enabled augmenters.
- Failure policy must be explicit:
  - `Critical()` augmenters fail-fast
  - non-critical augmenters warn and continue
- The original stored user input must remain unchanged for real user/network
  turns.

### Important Constraint

Synthetic turns are exempt from the “original user input preserved” invariant.
They require their own persistence model in Workstream 4.

## Workstream 4: Synthetic Reentry Model

### Problem

Synthetic reentry cannot be treated as a normal user message because too many
subsystems currently rely on `EventTypeUserMessage` semantics:

- prompt input persistence in [manager_prompt.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_prompt.go:197)
- transcript rendering in [transcript.go](/Users/pedronauck/Dev/compozy/agh/internal/transcript/transcript.go:150)
- hooks input classes in [manager_hooks.go](/Users/pedronauck/Dev/compozy/agh/internal/session/manager_hooks.go:20)
- extension prompt submission in [host_api.go](/Users/pedronauck/Dev/compozy/agh/internal/extension/host_api.go:1621)

### Decision

Synthetic reentry gets:

- a dedicated turn origin
- a dedicated persisted event type
- explicit trust/audit semantics
- explicit queueing/order semantics

### New Concepts

```go
const (
    EventTypeSyntheticReentry = "synthetic_reentry"
)

type SyntheticReentryPayload struct {
    TaskRunID   string
    TaskID      string
    Reason      string
    Summary     string
    ResultJSON  json.RawMessage
    CreatedAt   time.Time
}
```

### Required Behavior

- Synthetic reentry must never be persisted as `EventTypeUserMessage`.
- Synthetic reentry must always reference the originating task run id.
- Only daemon-owned runtime code may emit synthetic reentry events.
- Queue semantics must be explicit:
  - if a turn is active, reentry is queued
  - stopped session => drop with observable summary
  - multiple reentries => FIFO by completion time

### Hooks and Transcript

- hooks must receive a dedicated input class such as `synthetic_reentry`
- transcript must render synthetic reentry as a daemon/runtime-originated
  message, not a user message
- extension host code must stop assuming “first `user_message` after prompt
  submission” as the only source of turn identification for synthetic paths

## Workstream 5: Detached Async Runtime on Task Infrastructure

### Problem

Detached async runtime behavior remains necessary, but the earlier
`BackgroundRun` idea would duplicate an already capable runtime substrate:

- durable run lifecycle in [types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:38)
- durable run model in [types.go](/Users/pedronauck/Dev/compozy/agh/internal/task/types.go:212)
- persistence in [global_db_task.go](/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db_task.go:231)
- daemon bridge in [task_runtime.go](/Users/pedronauck/Dev/compozy/agh/internal/daemon/task_runtime.go:25)

### Decision

Detached async harness work reuses the task runtime.

There is no separate persisted `BackgroundRun` entity in this TechSpec.

### Harness Mapping onto Task Runtime

Harness-owned detached work will be represented as:

- task record owned by daemon/session runtime semantics
- task run record carrying execution attempt, result, error, timestamps, and
  session linkage
- synthetic reentry triggered from task-run completion when harness policy
  requires it

### Required Behavior

- harness-created detached runs must use explicit origin metadata
- idempotency must reuse task-run idempotency semantics
- boot recovery must reuse task-runtime reconciliation patterns
- session ownership and wake-up targeting must be explicit in task metadata and
  result payload

### Integration Contract

This TechSpec requires a harness-to-task bridge that can:

- enqueue detached work
- observe task-run completion
- map qualifying completions into synthetic reentry
- decide silent completion vs wake-up based on resolved harness policy

### Why This Decision Is Final

This is the final architectural choice for the TechSpec because:

- the existing task runtime already models durable async execution correctly
- it already has recovery semantics
- it already has session bridging
- introducing a second persisted async runtime would increase operational and
  conceptual complexity without enough technical justification

## Workstream 6: Storage, Observability, and Verification

### Schema Evolution

The TechSpec must explicitly declare zero-legacy schema evolution.

Required statement:

- schema changes for this initiative may use destructive zero-legacy
  migration/rebuild strategy where appropriate
- the strategy must be documented in the implementation plan and tests

### Observability

Harness lifecycle events must land in `EventSummary`:

- registry write path in [global_db_observe.go](/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db_observe.go:12)
- observer path in [observer.go](/Users/pedronauck/Dev/compozy/agh/internal/observe/observer.go:495)

Recommended event types:

- `harness.context_resolved`
- `harness.section_selected`
- `harness.augmenter_applied`
- `harness.augmenter_failed`
- `harness.synthetic_reentry_emitted`
- `harness.synthetic_reentry_dropped`
- `harness.detached_run_enqueued`
- `harness.detached_run_completed`

### Read-Side

Until public APIs are added, operational inspection is via:

- SQL over globaldb tables
- `event_summaries`
- existing observe surfaces

### Verification

The TechSpec must explicitly require:

- `make verify`
- `make test-integration`

### Test Coverage Areas

Unit and integration coverage must include:

- harness context resolution matrix
- startup section selection
- network startup prompt behavior
- composite augmenter ordering
- aggregate augmenter budget behavior
- fail-fast vs warn-and-continue augmenter policy
- synthetic reentry persistence semantics
- transcript rendering for synthetic events
- extension prompt flow compatibility
- detached task-run to synthetic reentry conversion
- boot recovery and orphan recovery
- shutdown behavior with in-flight detached work

## Impact Analysis

| Component | Impact | Notes |
|---|---|---|
| `internal/daemon` | high | gains harness context resolution, section selection, augmenter composition, task-runtime harness bridge |
| `internal/session` | high | gains synthetic turn/event semantics and updated prompt persistence behavior |
| `internal/task` | medium/high | becomes the durable substrate for detached harness work |
| `internal/store/globaldb` | medium | schema/read-model changes, event summary expansion |
| `internal/observe` | medium | explicit harness event visibility |
| `internal/transcript` | medium | must distinguish synthetic runtime-originated input from user input |
| `internal/extension` | medium/high | prompt submission flow assumptions need adjustment for synthetic turn paths |

## Development Sequencing

1. Workstream 1: context resolution and policy matrix
2. Workstream 2: startup section model and network startup integration
3. Workstream 3: composite augmenter rollout
4. Workstream 4: synthetic turn/event model
5. Workstream 5: harness bridge onto task runtime
6. Workstream 6: observability, migration strategy, and full verification

This sequencing is implementation order, not scope reduction. The full TechSpec
describes the final integrated architecture.

## Technical Considerations

### Key Decisions

- Harness behavior is resolved from session context plus turn origin, not from a
  foundational profile enum.
- Startup prompt architecture stays on the existing assembler/provider path.
- Turn augmentation evolves by composition first, not by immediate manager API
  break.
- Synthetic reentry gets its own turn/event semantics.
- Detached async runtime reuses the task runtime.
- Coordinator-grade orchestration remains follow-up work after the harness
  foundation lands.

### Main Risks

- policy matrix drift if resolution rules are not centralized
- over-complex synthetic event integration if transcript/hooks/extensions are
  patched inconsistently
- hidden coupling between task runtime and harness wake-up semantics
- under-testing of cross-product state combinations

### Mitigations

- centralize resolution in daemon-owned code
- encode matrix combinations in tests
- reuse task runtime rather than duplicating async persistence
- make synthetic event semantics explicit at every boundary

## Revised ADR Direction

The revised TechSpec implies these ADR updates:

- **ADR-001**
  - from: internal harness profiles with hybrid resolution
  - to: harness policy resolved from durable session context and turn origin

- **ADR-002**
  - stays conceptually similar
  - but should explicitly document staged augmenter composition rather than
    immediate seam replacement

- **ADR-003**
  - must change materially
  - from: global `BackgroundRun`
  - to: detached harness work reuses task runtime and task-run persistence

- **ADR-004**
  - remains valid
  - specialized coordinator/planner/reviewer orchestration stays out of scope
