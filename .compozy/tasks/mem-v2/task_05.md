---
status: pending
title: Write Controller and Decisions WAL
type: backend
complexity: critical
dependencies:
  - task_01
  - task_03
  - task_04
---

# Task 05: Write Controller and Decisions WAL

## Overview

Implement the single mutation orchestrator for Memory v2 and the durable `memory_decisions` write-ahead log that protects curated state. This task converts the current file-first write path into the Slice 1 rule-first controller path, where every memory mutation is decided, recorded, and replayable before the durable file change lands.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Write controller`, `Data Models`, `Safety Invariants`, and `Greenfield Delete Targets`.
- ACTIVATE `agh-code-guidelines`, `agh-cleanup-failure-paths`, and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside controller/WAL/store seams; public transports stay thin adapters in later tasks.
- TESTS REQUIRED: happy-path decisions, ambiguous-target handling, NOOP/REJECT cases, WAL-before-write ordering, replay, and revert coverage must ship here.
- NO WORKAROUNDS: no write surface may bypass the controller once this task lands.
</critical>

<requirements>
- MUST implement the Slice 1 lexical/entity-only controller algorithm for ADD/UPDATE/DELETE/NOOP/REJECT decisions.
- MUST persist `memory_decisions` before file/catalog mutation and record enough data to replay or revert deterministically.
- MUST ensure controller-driven writes use the new atomic storage substrate from `task_03`.
- MUST provide a stable controller surface for CLI/API/native-tool/extractor/dream/provider callers to adopt later.
- MUST remove any remaining direct mutation path that would bypass the controller in runtime code.
</requirements>

## Subtasks
- [ ] 5.1 Create the controller package and rule-first decision flow.
- [ ] 5.2 Add durable `memory_decisions` WAL persistence with replay/revert material.
- [ ] 5.3 Integrate controller decisions with the new storage runtime and scan assets.
- [ ] 5.4 Replace direct runtime mutation paths with controller entry points where needed.
- [ ] 5.5 Add focused controller/WAL tests for happy, failure, replay, and revert flows.

## Implementation Details

See TechSpec `Write controller`, `Data Models`, and `Development Sequencing` step 11. This task owns mutation semantics only; public API/CLI/native-tool exposure happens later but must be able to call into this controller without adding logic forks.

### Relevant Files
- `internal/memory/store.go` — current direct write/delete path that the controller will supersede.
- `internal/memory/catalog.go` — candidate lookup and derived index updates used during decisioning.
- `internal/api/core/memory.go` — current handler write path that later tasks will reroute through the controller.
- `internal/extension/host_api.go` — current extension write surface that must eventually adopt the controller.
- `internal/memory/document.go` — header/frontmatter parsing helpers used by decisioning.
- `.compozy/tasks/mem-v2/analysis/analysis_write-controller.md` — design rationale and edge cases.

### Dependent Files
- `internal/memory/recall/*` — recall will consume controller-produced IDs and provenance semantics.
- `internal/memory/extractor/*` — extractor outputs must become controller proposals rather than direct writes.
- `internal/memory/dream.go` — dreaming promotions must route through the controller.
- `.compozy/tasks/mem-v2/task_14.md` — public contract work depends on final controller/WAL payloads.
- `.compozy/tasks/mem-v2/task_17.md` — CLI hard cut depends on final controller semantics.

### Related ADRs
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — normative controller behavior.
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — authority split between curated files, WAL, and audit events.
- [ADR-005: `_system/` Namespace Invariant](adrs/adr-005.md) — constrains non-injected failure/DLQ outputs related to mutation flows.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task defines the runtime mutation seam that provider hooks, extension host callers, and daemon-owned sub-systems must share.
- Agent manageability: public verbs are still unchanged here, but later CLI/HTTP/UDS/native-tool surfaces must expose only controller-backed operations.
- Config lifecycle: none — checked surfaces are controller mode keys, defaults, settings payloads, and docs; public config work is deferred to `task_13`.

### Web/Docs Impact

- `web/`: none — checked surfaces are generated types and knowledge/settings/session UI; public controller payloads land later via codegen.
- `packages/site`: none — checked surfaces are runtime memory docs and generated references; docs update after verbs and payloads stabilize.

## Deliverables

- A production controller package that owns all Memory v2 mutation decisions.
- Durable `memory_decisions` WAL writes with replay/revert-ready payloads.
- Direct runtime mutation call sites removed or rerouted behind the controller seam.
- Focused unit/integration tests for decisioning, WAL ordering, replay, and revert.

## Tests

- Unit tests:
  - [ ] Exact-match, update-target, delete-target, NOOP, and REJECT decision cases resolve deterministically.
  - [ ] Ambiguous candidates trigger the approved tiebreak path without introducing vector-only logic.
  - [ ] WAL rows are written before the curated file mutation and contain deterministic replay material.
- Integration tests:
  - [ ] Crash or failure between WAL write and final mutation replays safely on restart.
  - [ ] Revert restores `prior_content` and catalog/event state deterministically.
  - [ ] Runtime code paths that previously mutated memory directly now go through the controller seam.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/tools/memory_tool.py`
- `.resources/hermes/agent/curator.py`
- `.resources/codex/codex-rs/memories/write/src/control.rs`
- `.resources/codex/codex-rs/memories/write/src/phase1.rs`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Every Memory v2 mutation has a controller decision and durable WAL record before persistence.
- No runtime write path bypasses the controller.

