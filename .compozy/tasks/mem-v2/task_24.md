---
status: completed
title: CLI/API Reference and Discoverability Co-Ship
type: docs
complexity: high
dependencies:
  - task_15
  - task_16
  - task_17
  - task_18
  - task_23
---

# Task 24: CLI/API Reference and Discoverability Co-Ship

## Overview

Update the generated and semi-generated discoverability surfaces that operators and agents rely on: CLI reference pages, API reference pages, and site-level truth checks. This task closes the loop between the final runtime/transport/CLI/tool behavior and the discoverable docs that must ship in the same change.

<critical>
- ALWAYS READ `_techspec.md`, `packages/site/CLAUDE.md`, and the affected runtime/API/CLI surfaces before editing docs/reference files.
- REFERENCE the TechSpec sections `Web/Docs Impact`, `Agent Manageability Plan`, and `Impact Analysis`.
- ACTIVATE `documentation-writer` before editing reference-adjacent docs; use `copywriting` only for operator-facing explanatory text around the references.
- MINIMIZE churn outside generated references and site truth/discovery tests.
- TESTS REQUIRED: CLI reference generation, API reference truth, docs-truth tests, and manual-example checks must ship here.
- NO WORKAROUNDS: if a generated reference is wrong, fix the source surface; do not patch the generated page by hand.
</critical>

<requirements>
- MUST regenerate and update CLI reference pages affected by the Memory v2 hard cut.
- MUST update API reference pages and supporting truth/discovery tests for the final memory transport surface.
- MUST ensure site-level discoverability and example tests reflect the final Slice 1 verbs, payloads, and tool surfaces.
- MUST keep generated and source-backed reference pages in sync with the actual runtime.
- MUST remove any stale discoverability text or examples that mention deleted memory commands or payloads.
</requirements>

## Subtasks
- [x] 24.1 Regenerate CLI reference pages for the final memory command tree.
- [x] 24.2 Update API reference pages and truth guards for the final memory routes/payloads.
- [x] 24.3 Refresh docs discovery and manual-example tests that mention memory/config/session surfaces.
- [x] 24.4 Confirm all discoverability surfaces reflect the same final runtime truth.

## Implementation Details

See TechSpec `Web/Docs Impact`, `Agent Manageability Plan`, and `Impact Analysis`. This task depends on the final contract, route, CLI, and tool names being settled; fix sources rather than patching generated output by hand.

### Relevant Files
- `packages/site/content/runtime/cli-reference/memory/**` — generated CLI memory reference pages.
- `packages/site/content/runtime/api-reference/memory.mdx` — API reference entry for memory.
- `packages/site/content/runtime/api-reference/index.mdx` — discoverability entry that may need memory surface updates.
- `packages/site/lib/runtime-docs-truth.test.ts` — truth guard for runtime docs/reference alignment.
- `packages/site/lib/runtime-manual-api-routes.test.ts` — manual API route checks affected by memory surface changes.
- `packages/site/lib/runtime-manual-cli-examples.test.ts` — CLI example checks affected by renamed/removed memory commands.

### Dependent Files
- `packages/site/lib/runtime-docs-discovery.test.ts` — discoverability assertions that may need updates.
- `openapi/agh.json` — generated API truth source for the memory reference.
- `internal/cli/memory.go` — CLI reference generation source-of-truth for memory commands.

### Related ADRs
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — command/API reference implications.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — search/recall reference implications.
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — tool/extension discoverability implications.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: reference docs must describe the final extension-host and provider-facing memory surfaces truthfully.
- Agent manageability: this task documents the final CLI, HTTP, UDS, and builtin-tool memory paths agents can operate.
- Config lifecycle: examples and discoverability text must reflect the final memory config/settings keys and semantics.

### Web/Docs Impact

- `web/`: none — this task updates docs/reference artifacts only.
- `packages/site`: CLI reference, API reference, and docs-truth/discovery/test files are expected to change here.

## Deliverables

- Regenerated CLI memory reference pages.
- Updated memory API reference/discoverability pages and truth guards.
- Refreshed CLI/API/manual-example/docs-discovery tests for Memory v2.

## Tests

- Unit tests:
  - [x] CLI and API reference pages render the final Memory v2 verbs/routes and omit removed ones.
- Integration tests:
  - [x] `make cli-docs` regenerates the CLI memory reference successfully.
  - [x] `packages/site/lib/runtime-manual-api-routes.test.ts` passes for the final memory routes.
  - [x] `packages/site/lib/runtime-manual-cli-examples.test.ts` passes for the final memory CLI examples.
  - [x] `packages/site/lib/runtime-docs-truth.test.ts` and `packages/site/lib/runtime-docs-discovery.test.ts` pass after the updates.
- Test coverage target: all affected site truth/discoverability/reference guards.
- All tests must pass.

## References

- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/hermes/website/docs/developer-guide/memory-provider-plugin.md`
- `.resources/codex/docs`
- `.resources/claude-code/cli`

## Success Criteria

- All tests passing.
- CLI/API references and discoverability surfaces reflect the final Slice 1 memory truth.
- No stale command, route, or example remains in the site reference layers.

