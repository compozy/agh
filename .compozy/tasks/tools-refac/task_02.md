---
status: completed
title: Tools Guidance Assets and Startup Prompt Section
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Tools Guidance Assets and Startup Prompt Section

## Overview

Teach agents that the registry exists, what the default discovery loop is, and when AGH-native tools are preferred over shelling out to `agh ...`. This task adds the bundled `agh-tools-guide`, wires a new startup prompt section, and removes the current CLI-first guidance bias from the shipped prompt/catalog assets.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 before editing prompt or skill guidance
- REFERENCE TECHSPEC sections "Architectural Boundaries", "Agent Manageability Plan", and "Docs And Generated Surfaces"
- FOCUS ON WHAT: teach discovery and invocation behavior; do not broaden the tool catalog here
- MINIMIZE CODE — reuse the existing prompt-section and bundled-skill pipelines already shipped on this branch
- TESTS REQUIRED — prompt assembly and skill catalog outputs must stay deterministic
</critical>

<requirements>
1. MUST add a dedicated startup prompt section for tools guidance.
2. MUST add a bundled `agh-tools-guide` skill that teaches `search -> info -> invoke`, tool-first behavior, and management-surface exceptions.
3. MUST update shipped catalog and setup guidance so AGH-internal operations are described as tools-first by convention.
4. MUST preserve current bundled-skill loading and startup prompt assembly patterns rather than introducing a second guidance pipeline.
</requirements>

## Subtasks
- [x] 2.1 Add `agh-tools-guide` under the bundled skills set
- [x] 2.2 Wire a new tools prompt section into startup prompt assembly
- [x] 2.3 Update skill catalog copy so tool discovery is explicit and current
- [x] 2.4 Update `agh-agent-setup` examples to reflect default discovery toolsets and tool-first behavior
- [x] 2.5 Add tests for prompt assembly, bundled-skill availability, and catalog output

## Implementation Details

See TechSpec sections "Architectural Boundaries", "Implementation Design", and "Agent Manageability Plan". The task should change shipped guidance only; later tasks will add more tool families on top of this teaching layer.

### Relevant Files
- `internal/daemon/prompt_sections.go` — current startup prompt sections and insertion points
- `internal/daemon/composed_assembler.go` — prompt assembly pipeline
- `internal/skills/catalog.go` — current catalog text still teaches CLI-first skill loading
- `internal/skills/bundled/skills/agh-agent-setup/SKILL.md` — current built-in setup guidance that must be updated
- `internal/skills/bundled/skills/` — home for the new `agh-tools-guide`

### Dependent Files
- `internal/skills/bundled/content.go` — bundled content index that must include the new guide
- `packages/site/content/runtime/core/configuration/agent-md.mdx` — docs must match the shipped prompt and guide behavior

### Related ADRs
- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md)

### Web/Docs Impact
- `web/`: none — checked surfaces `web/src/systems/session/*` and tool-call renderers; this task changes runtime guidance, not web contracts or UI flows.
- `packages/site`: `packages/site/content/runtime/core/configuration/agent-md.mdx`, `packages/site/content/runtime/core/agents/definitions.mdx`, and `packages/site/content/runtime/core/network/index.mdx` where current operator/agent guidance cross-references shipped startup behavior.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: adds a bundled guidance asset and a new prompt section without changing extension or protocol surfaces.
- Agent manageability: makes the discovery loop and management-surface exceptions explicit to every agent by default.
- Config lifecycle: no new config keys expected; default guidance must stay consistent with existing agent toolset grammar and policy behavior from task_01.

## Deliverables
- New bundled `agh-tools-guide`
- New startup prompt section for tools guidance
- Updated shipped catalog and setup guidance
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for startup prompt assembly and bundled-skill visibility **(REQUIRED)**

## Tests
- Unit tests:
  - [x] bundled skill registry exposes `agh-tools-guide` with deterministic content loading
  - [x] tools prompt section renders only when the runtime includes the corresponding section in startup assembly
  - [x] skill catalog output no longer biases agents toward CLI-first AGH-internal flows
- Integration tests:
  - [x] startup prompt assembled by the daemon includes the new tools section alongside existing sections
  - [x] bundled-skill and catalog outputs remain stable across fresh daemon boot and prompt rebuild paths
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Agents receive explicit tools guidance at startup and through bundled skill discovery
- Shipped catalog and setup guidance no longer contradict the canonical tool-first posture

## References
- `.compozy/tasks/tools-refac/analysis/competitor-tool-surface-notes.md`
- `.resources/claude-code/constants/prompts.ts`
- `.resources/hermes/agent/prompt_builder.py`
- `.resources/openclaw/src/agents/system-prompt.ts`
