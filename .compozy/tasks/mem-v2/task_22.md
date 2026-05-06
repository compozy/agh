---
status: complete
title: Web Session Inspector Memory Surface
type: frontend
complexity: medium
dependencies:
  - task_15
  - task_19
---

# Task 22: Web Session Inspector Memory Surface

## Overview

Update the session inspector so its memory tab reflects Memory v2’s final packaged snapshot, lineage, and forensic semantics. This task keeps the inspector truthful to the daemon’s real session/memory model without turning it into a speculative dashboard.

<critical>
- ALWAYS READ `_techspec.md`, `web/CLAUDE.md`, and the relevant ADRs before implementation.
- REFERENCE the TechSpec sections `Session ledger`, `Web/Docs Impact`, and `System Architecture`.
- ACTIVATE `react`, `tailwindcss`, `tanstack-query-best-practices`, and `vercel-react-best-practices` before editing web code.
- MINIMIZE CODE churn outside the session inspector and its route consumers.
- TESTS REQUIRED: memory-tab rendering, lineage/forensic metadata display, and empty/error states must ship here.
- NO WORKAROUNDS: do not invent memory-session controls or metrics that the runtime does not expose.
</critical>

<requirements>
- MUST update the session inspector memory surface to the final Memory v2 snapshot and lineage semantics.
- MUST expose only the approved forensic/session-facing memory details from the daemon contract.
- MUST keep route consumers and tests aligned with the regenerated session/memory payloads.
- MUST preserve truthful empty/error states when memory/ledger data is unavailable.
- MUST avoid introducing editor-style controls or speculative observability widgets into the inspector.
</requirements>

## Subtasks
- [x] 22.1 Update session inspector memory-tab rendering to the final Memory v2 payloads.
- [x] 22.2 Surface lineage/ledger/session-memory metadata that the daemon actually exposes.
- [x] 22.3 Refresh route and component tests for the inspector memory state.
- [x] 22.4 Confirm the inspector remains forensic/read-only where the TechSpec requires it.

## Implementation Details

See TechSpec `Session ledger`, `Web/Docs Impact`, and `System Architecture`. The inspector should explain what the running session saw or persisted, not become a separate memory-management UI.

### Relevant Files
- `web/src/systems/session/components/session-inspector.tsx` — session inspector implementation and memory tab.
- `web/src/routes/_app/session.$id.tsx` — operator session route that consumes the inspector.
- `web/src/routes/_app/agents.$name.sessions.$id.tsx` — agent-scoped session route using the same inspector behavior.
- `web/src/routes/_app/-agents.$name.sessions.$id.test.tsx` — route-level inspector coverage.
- `web/src/systems/session/index.ts` — public exports for session inspector components.

### Dependent Files
- `packages/site/content/runtime/core/sessions/**` — later docs task depends on the truthful inspector/session semantics.
- `.compozy/tasks/mem-v2/task_23.md` — runtime docs task depends on the final session-facing behavior.

### Related ADRs
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md) — forensic/session semantics for UI.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — packaged memory semantics that may surface in the inspector.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none directly — checked surfaces are provider/extension/runtime hooks, which are backend concerns.
- Agent manageability: the inspector must remain a truthful consumer of the same daemon state available through machine-readable surfaces.
- Config lifecycle: none — checked surfaces are memory/session settings; no config editing happens here.

### Web/Docs Impact

- `web/`: `web/src/systems/session/components/session-inspector.tsx` and its consuming routes/tests are expected to change here.
- `packages/site`: session/runtime docs may need updates later to match the final inspector behavior.

## Deliverables

- Updated session inspector memory tab for Memory v2 snapshot/lineage semantics.
- Refreshed route/component tests for the final session-memory payloads.
- Truthful read-only/forensic presentation of session memory information.

## Tests

- Unit tests:
  - [x] Session inspector renders Memory v2 session-memory rows, lineage, and empty states correctly.
  - [x] Read-only/forensic semantics are preserved in the inspector UI.
- Integration tests:
  - [x] Session routes render the updated inspector correctly for success and failure scenarios.
  - [x] Generated/session payload updates do not break existing inspector route composition.
- Test coverage target: web coverage for all changed session inspector memory surfaces.
- All tests must pass.

## References

- `.resources/hermes/hermes_state.py`
- `.resources/codex/codex-rs/rollout/src/recorder.rs`
- `.resources/claude-code/memdir/memdir.ts`

## Success Criteria

- All tests passing.
- The session inspector reflects Memory v2 snapshot and forensic semantics truthfully.
- No speculative session-memory controls or metrics are introduced.

