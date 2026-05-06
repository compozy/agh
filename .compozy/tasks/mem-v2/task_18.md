---
status: completed
title: Native Tools and Extension Host Memory Surfaces
type: backend
complexity: high
dependencies:
  - task_07
  - task_14
  - task_16
---

# Task 18: Native Tools and Extension Host Memory Surfaces

## Overview

Hard-cut the native-tool and extension-host surfaces to the approved Memory v2 model. This task aligns agent-callable builtin memory tools and extension-facing host operations with the same controller/recall/provider semantics already established for the daemon and public API.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Native tools`, `Extensibility Integration Plan`, `Agent Manageability Plan`, and `Development Sequencing` steps 14 and 24.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside builtin tool, host API, and daemon tool registration seams.
- TESTS REQUIRED: tool descriptor hard cut, policy gating, host API parity, and deterministic write/read semantics must ship here.
- NO WORKAROUNDS: do not leave old read-only builtin descriptors or direct extension-write bypasses alive beside the new surface.
</critical>

<requirements>
- MUST replace the current builtin memory tool set with the approved `agh__memory_*` surface.
- MUST expose extension-host memory operations through controller/recall/provider-backed flows rather than direct store access.
- MUST keep tool policy gating and root/sub-agent permission behavior aligned with the TechSpec.
- MUST update daemon tool registration and extension host tests to the new naming and semantics.
- MUST remove any redundant or bypassing memory host/tool write path.
</requirements>

## Subtasks
- [x] 18.1 Replace builtin memory tool descriptors and IDs with the approved Slice 1 set.
- [x] 18.2 Update daemon native-tool registration and policy gating for the new memory tool surface.
- [x] 18.3 Refactor extension host memory operations onto controller/recall/provider-backed flows.
- [x] 18.4 Add focused tests for descriptor hard cuts, policy behavior, and host API parity.

## Implementation Details

See TechSpec `Native tools`, `Extensibility Integration Plan`, `Agent Manageability Plan`, and `Development Sequencing` step 24. This task owns agent-callable builtin tool and extension-host surfaces; CLI and web/docs surfaces are separate tasks.

### Relevant Files
- `internal/tools/builtin/memory.go` — builtin memory descriptors to hard-cut.
- `internal/tools/builtin_ids.go` — builtin memory tool IDs that must align with the new set.
- `internal/daemon/native_tools.go` — daemon native-tool registration and policy wiring.
- `internal/extension/host_api.go` — extension-facing memory operations that must adopt controller/recall/provider flows.
- `internal/extension/host_api_test.go` — host API coverage to extend with Memory v2 behavior.
- `internal/tools/native_test.go` — native-tool behavior tests that should cover descriptor/policy changes.

### Dependent Files
- `internal/cli/tool.go` and `internal/cli/tool_test.go` — tool inspection surfaces must reflect the new builtin memory set.
- `packages/site/content/runtime/api-reference/extensions.mdx` — later docs/reference task depends on the final extension host memory behavior.
- `.compozy/tasks/mem-v2/task_24.md` — reference/discoverability task depends on final tool/host API names.
- `.compozy/tasks/mem-v2/task_23.md` — runtime docs task depends on final builtin/host API semantics.

### Related ADRs
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — provider/host API implications.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — write-tool semantics must route through the controller.
- [ADR-010: Fact Extraction Location — Hybrid Per-Turn Hook + Optional Compaction Flush](adrs/adr-010.md) — extension-host and tool proposals must not bypass extractor/controller ownership.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task is the primary extension-host and builtin-tool hard cut for Memory v2.
- Agent manageability: this task defines the final native-tool surface agents will invoke for memory reads and writes.
- Config lifecycle: tool/host behavior must respect the memory config/provider defaults established in `task_13`.

### Web/Docs Impact

- `web/`: none — checked surfaces are web memory/settings/session pages; they do not call builtin tools directly.
- `packages/site`: extension and tool docs plus API references must be updated later from these finalized surfaces.

## Deliverables

- New builtin memory tool descriptors and IDs for the approved `agh__memory_*` surface.
- Daemon native-tool registration and policy behavior updated for Memory v2.
- Extension host memory flows rerouted through controller/recall/provider seams.
- Focused tests for tool IDs, policy, host API parity, and bypass removal.

## Tests

- Unit tests:
  - [x] Builtin memory tool descriptors expose only the approved IDs, names, and schemas.
  - [x] Tool policy correctly distinguishes read-only vs proposal/note write access for root and sub-agents.
  - [x] Extension host memory operations call controller/recall/provider seams instead of direct store mutation.
- Integration tests:
  - [x] Daemon tool registration exposes the new builtin memory surface and hides removed tools.
  - [x] Host API tests prove memory read/write behavior matches the final daemon semantics and deterministic errors.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/tools/memory_tool.py`
- `.resources/hermes/website/docs/developer-guide/memory-provider-plugin.md`
- `.resources/codex/tools`
- `.resources/claude-code/tools/AgentTool/agentMemory.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Agents and extensions see one truthful Memory v2 builtin-tool and host API surface.
- No legacy or bypassing memory tool/host path remains alive.
