---
status: completed
title: Site Docs, Generated References, and Example Alignment
type: docs
complexity: high
dependencies:
  - task_02
  - task_03
  - task_04
  - task_05
  - task_06
  - task_07
  - task_08
  - task_09
  - task_10
---

# Task 11: Site Docs, Generated References, and Example Alignment

## Overview

Bring the documentation, generated CLI references, and downstream generated artifacts into alignment with the branch-grounded canonical surface implemented by tasks 01-10. This task also removes stale guidance that still teaches the old CLI-first or raw-`claim_token` behavior after the hard cuts land.

<critical>
- ALWAYS READ `_techspec.md`, every ADR, and the completed implementation tasks before editing docs or generated references
- REFERENCE TECHSPEC sections "Docs And Generated Surfaces", "Old vs New Effective Behavior", and "Post-Implementation Residual Checks"
- FOCUS ON WHAT: update docs, examples, and generated references to match the shipped runtime truth
- MINIMIZE prose churn — delete obsolete guidance instead of layering compatibility notes over it
- TESTS REQUIRED — generated docs, codegen artifacts, and example references must build and match the runtime contracts
</critical>

<requirements>
1. MUST update runtime docs to teach the final tools-first posture, default discovery, and operator-only exceptions.
2. MUST update autonomy, hooks, automation, extensions, config, memory, workspace, network, and MCP auth docs to match the implemented tool surface.
3. MUST regenerate CLI reference pages and any generated contract artifacts touched by the implementation tasks.
4. MUST remove stale examples or prose that still reference raw `claim_token`, opt-in discovery defaults, or CLI-only management where the canonical surface changed.
</requirements>

## Subtasks
- [x] 11.1 Update core runtime docs for the new tool surface, discovery defaults, and management boundaries
- [x] 11.2 Regenerate CLI reference pages so command docs match the current branch behavior after the implementation cuts
- [x] 11.3 Refresh generated contract references consumed by `web/` when public API shapes changed
- [x] 11.4 Remove stale examples and delete-target prose from site pages and task artifacts
- [x] 11.5 Add build and regression coverage for docs/codegen alignment

## Implementation Details

See TechSpec sections "Docs And Generated Surfaces", "Post-Implementation Residual Checks", and "Implementation Steps". This task is the cleanup and alignment pass after the runtime surface is real; it should not speculate ahead of the implemented behavior.

### Relevant Files
- `packages/site/content/runtime/core/configuration/agent-md.mdx` — startup guidance and AGENT tooling posture
- `packages/site/content/runtime/core/configuration/config-toml.mdx` — default toolsets and config lifecycle documentation
- `packages/site/content/runtime/core/autonomy/task-runs-and-leases.mdx` — autonomy hard-cut documentation
- `packages/site/content/runtime/core/hooks/*.mdx` — hook ownership and mutation guidance
- `packages/site/content/runtime/core/extensions/*.mdx` — extension lifecycle guidance
- `packages/site/content/runtime/core/automation/*.mdx` — automation lifecycle guidance

### Dependent Files
- `packages/site/content/runtime/cli-reference/` — generated command reference tree that must be refreshed
- `web/src/generated/agh-openapi.d.ts` — generated web contract that must match the public API after codegen

### Related ADRs
- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md)
- [ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces](adrs/adr-004-mcp-auth-status-tool.md)
- [ADR-005: Autonomy Tool Surfaces Are Session-Bound And Never Expose Raw Claim Tokens](adrs/adr-005-session-bound-autonomy-surface.md)
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact
- `web/`: `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, and `web/src/systems/tasks/mocks/fixtures.ts` when contract or example payloads changed under task_09 or task_10.
- `packages/site`: runtime core pages for configuration, agents, autonomy, hooks, automation, extensions, memory, network, workspaces, sessions, bridges, and all affected CLI reference pages.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: docs must explain extension points, hooks, hosted MCP, and built-in tool families as they now exist.
- Agent manageability: docs and examples must show the real CLI/HTTP/UDS/tool surfaces agents and operators use after the redesign.
- Config lifecycle: docs must reflect the implemented config mutation boundaries and default discovery behavior without stale compatibility language.

## Deliverables
- Updated site docs for the canonical tool surface
- Regenerated CLI reference pages and any touched generated contract references
- Removed stale examples and delete-target prose
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for docs/build/codegen alignment **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] doc snippets and generated examples no longer mention removed raw-`claim_token` or stale CLI-first behavior
  - [ ] generated reference files stay consistent with the implemented public contracts
- Integration tests:
  - [ ] `make cli-docs` regenerates the expected CLI reference pages without drift
  - [ ] `make codegen` and `make codegen-check` pass with updated generated artifacts
  - [ ] `packages/site` build succeeds with the rewritten runtime docs
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Site docs and generated references match the runtime truth implemented by tasks 01-10
- No stale examples remain for discovery defaults, raw `claim_token`, or the old CLI-first guidance
