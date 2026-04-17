---
status: pending
title: MCP Servers scoped collection page
type: frontend
complexity: high
dependencies:
  - task_09
---

# Task 13: MCP Servers scoped collection page

## Overview

Implement the most complex collection page in the settings UI: `mcp-servers`, with explicit scope, target, and precedence semantics. This page must make source selection and fallback behavior legible, because the TechSpec intentionally exposes `target=auto|config|sidecar`, `effective_source`, `shadowed_sources`, and workspace scope in v1.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Data Models", "Collection mutation semantics", and "Known Risks"
- FOCUS ON "WHAT" — make scope and precedence understandable to the user, not merely technically available
- MINIMIZE CODE — reuse shared collection patterns from task_12 where they remain explicit enough
- TESTS REQUIRED — scope switching, target selection, and precedence/fallback behavior must be covered
- GREENFIELD: não esconder precedence atrás de auto-magic; a UI precisa expor `effective_source` e `shadowed_sources`
</critical>

<requirements>
- MUST implement the `mcp-servers` settings route with global and workspace scope support
- MUST expose `target=auto|config|sidecar` selection and explain its consequences
- MUST render `effective_source`, `shadowed_sources`, and `available_targets` from the backend response
- MUST support scoped list/detail/edit/delete flows with workspace-aware behavior
- MUST make delete semantics clear when removing the highest-precedence definition reveals a lower-precedence one
- SHOULD reuse shared collection-page patterns only if they do not hide MCP-specific semantics
</requirements>

## Design References

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| MCP Servers | `docs/design/paper/settings/AGH Settings — MCP Servers@2x.png` | `AGH Settings — MCP Servers` (`YRR-0`) |

## Subtasks

- [ ] 13.1 Implement the `mcp-servers` route with scope-aware list and detail state
- [ ] 13.2 Add explicit target selection controls for `auto`, `config`, and `sidecar`
- [ ] 13.3 Render precedence metadata such as `effective_source`, `shadowed_sources`, and `available_targets`
- [ ] 13.4 Support workspace-scoped editing and deletion with clear fallback behavior
- [ ] 13.5 Add tests for scope changes, target behavior, and precedence rendering

## Implementation Details

See TechSpec sections "Data Models", "Scope rules", "Collection mutation semantics", and "Known Risks". This page is the only v1 workspace-scoped settings editor, so its UI must be more explicit than the other collection pages and should not blur global versus workspace operations.

### Relevant Files

- `web/src/routes/_app/settings/mcp-servers.tsx` — new scoped collection route for MCP servers
- `web/src/systems/settings/hooks/` — collection and workspace-aware hooks consumed by the route
- `web/src/systems/settings/components/` — natural place for precedence badges, target selectors, and scoped collection UI
- `web/src/systems/workspace/` — existing workspace selection state that the MCP page may need to consume
- `web/src/components/design-system/` — existing primitives that can host explicit source/target UI affordances

### Dependent Files

- `web/src/routes/_app/-settings*.test.tsx` — should add route coverage for scope and precedence behavior
- `web/src/systems/settings/**/*.test.ts` — should add precedence and target-selection coverage
- `web/src/routes/_app/settings/index.tsx` — may need to reflect workspace availability or default navigation hints
- `web/src/routeTree.gen.ts` — may update if route files change in this task

### Related ADRs

- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — Defines MCP target selection and precedence semantics
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Defines restart-required behavior for MCP settings changes

## Deliverables

- `mcp-servers` settings page with scope-aware list/detail/edit flows
- UI for explicit target selection and visible precedence metadata
- Workspace-scoped editing support for the one v1 settings section that needs it **(REQUIRED)**
- Route and component tests with >=80% coverage for scope, target, and precedence behavior **(REQUIRED)**
- Clear delete and fallback UX when lower-precedence definitions reappear **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] Scope switching between global and workspace reloads the correct collection state
  - [ ] Target selector renders and submits `auto`, `config`, and `sidecar` correctly
  - [ ] `effective_source`, `shadowed_sources`, and `available_targets` render correctly from backend metadata
  - [ ] Delete flow explains when a lower-precedence definition becomes effective again
  - [ ] New-server flows default `target=auto` to the intended sidecar destination while existing-server edits preserve highest-precedence semantics
- Integration tests:
  - [ ] Workspace-scoped MCP edits use the active workspace context correctly
  - [ ] Saving or deleting a server invalidates and refreshes the correct scoped query set
  - [ ] Switching between global and workspace scopes preserves separate cache entries and editor state boundaries
  - [ ] Auto-target edits update the effective source shown after refetch when a lower-precedence definition remains underneath
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80% for the MCP page and related shared components
- Operators can understand and control MCP scope and target semantics from the UI
- Precedence and fallback behavior are explicit instead of implicit or surprising
