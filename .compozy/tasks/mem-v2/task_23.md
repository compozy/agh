---
status: completed
title: Runtime Memory and Configuration Docs
type: docs
complexity: high
dependencies:
  - task_13
  - task_17
  - task_18
  - task_20
  - task_21
  - task_22
---

# Task 23: Runtime Memory and Configuration Docs

## Overview

Rewrite the hand-authored runtime docs that explain how memory works in AGH after the Slice 1 hard cut. This task updates the narrative docs for memory, configuration, workspaces, sessions, hooks, and extensions so the site stops teaching the replaced two-scope/file-first model.

<critical>
- ALWAYS READ `_techspec.md`, `packages/site/CLAUDE.md`, `COPY.md`, and the relevant ADRs before editing docs.
- REFERENCE the TechSpec sections `Web/Docs Impact`, `Config Lifecycle`, and `Greenfield Delete Targets`.
- ACTIVATE `documentation-writer` before runtime docs edits; add `copywriting` when changing public-facing explanation or headings.
- MINIMIZE churn outside the runtime docs listed here; generated references are handled in `task_24`.
- TESTS REQUIRED: runtime docs truth/discovery checks and any affected MDX validation must ship here.
- NO WORKAROUNDS: do not document behaviors, scopes, provider flows, or UI controls the runtime no longer supports.
</critical>

<requirements>
- MUST rewrite runtime memory docs to the final Slice 1 model, including scopes, authorities, dreaming, `_system/`, session ledgers, and manageability surfaces.
- MUST update configuration docs for the new memory config lifecycle and stable workspace/session file layout.
- MUST update adjacent runtime docs where memory changes alter hooks, extensions, sessions, or workspace behavior.
- MUST keep vocabulary aligned to the glossary and public product language aligned to `COPY.md`.
- MUST update docs tests or examples that assert old memory behavior.
</requirements>

## Subtasks
- [x] 23.1 Rewrite the core runtime memory docs for the Slice 1 model.
- [x] 23.2 Update runtime configuration, workspace, and session docs affected by Memory v2.
- [x] 23.3 Update hooks/extensions docs where Memory v2 changes extensibility or lifecycle behavior.
- [x] 23.4 Refresh docs tests/examples that assert memory/runtime truth.

## Implementation Details

See TechSpec `Web/Docs Impact`, `Config Lifecycle`, and `Impact Analysis`. Keep generated references out of this task; this task owns narrative and conceptual runtime docs only.

### Relevant Files
- `packages/site/content/runtime/core/memory/index.mdx` — top-level runtime memory narrative.
- `packages/site/content/runtime/core/memory/system.mdx` — system architecture explanation.
- `packages/site/content/runtime/core/memory/scopes.mdx` — scope model that currently documents the old behavior.
- `packages/site/content/runtime/core/memory/dream.mdx` — dreaming/consolidation narrative.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` — runtime config key explanation.
- `packages/site/content/runtime/core/configuration/file-locations.mdx` — file/layout documentation affected by workspace/session/ledger changes.

### Dependent Files
- `packages/site/content/runtime/core/workspaces/resolver.mdx` — workspace identity story affected by Memory v2.
- `packages/site/content/runtime/core/sessions/index.mdx` — session and ledger explanation affected by Memory v2.
- `packages/site/content/runtime/core/hooks/index.mdx` — hook lifecycle docs affected by persisted-message extraction.
- `packages/site/content/runtime/core/extensions/index.mdx` — provider/extensibility story affected by the new MemoryProvider.
- `packages/site/lib/runtime-docs-truth.test.ts` — docs-truth guardrails to update.

### Related ADRs
- [ADR-001: Hybrid Escopado as Memory Source-of-Truth Model](adrs/adr-001.md) — core memory narrative.
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md) — session/docs narrative.
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — extension/provider narrative.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: document the final MemoryProvider, hook, and extension-host story truthfully.
- Agent manageability: runtime docs must explain the supported CLI/HTTP/UDS/native-tool/operator paths without relying on the web UI.
- Config lifecycle: this task documents the final memory config keys, defaults, layout, and operational assumptions from `task_13`.

### Web/Docs Impact

- `web/`: none — this task updates site docs only.
- `packages/site`: runtime memory, configuration, workspace, session, hook, and extension docs are expected to change here.

## Deliverables

- Rewritten runtime memory docs for the Slice 1 model.
- Updated configuration, workspace, session, hooks, and extension narrative docs affected by Memory v2.
- Updated runtime docs truth/discovery checks where needed.

## Tests

- Unit tests:
  - [x] Changed MDX pages pass source generation, content validation, and local doc checks.
- Integration tests:
  - [x] `packages/site/lib/runtime-docs-truth.test.ts` passes for the updated memory/runtime pages.
  - [x] `packages/site/lib/runtime-docs-discovery.test.ts` passes after doc rewrites.
- Test coverage target: affected docs-truth/discovery guards for all changed runtime pages.
- All tests must pass.

## References

- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/hermes/website/docs/user-guide/features/memory-providers.md`
- `.resources/hermes/website/docs/developer-guide/memory-provider-plugin.md`
- `.resources/claude-code/memdir/memdir.ts`

## Success Criteria

- All tests passing.
- Runtime docs explain the final Slice 1 memory model truthfully and in current vocabulary.
- No site page continues to teach the replaced two-scope or legacy write-path model.

