---
status: completed
title: Config workspace-scoped loading and agent paths
type: backend
complexity: medium
dependencies:
  - task_04
---

# Task 05: Config workspace-scoped loading and agent paths

## Overview

Adjust `internal/config` so workspace root handling and agent discovery align with the Resolver: config loads from the workspace root per ADR-003, and multi-root agent discovery follows TechSpec merge order. Remove obsolete `resolveWorkspaceRoot` `Getwd` fallbacks that conflict with explicit Resolver inputs.

<critical>
- READ `_techspec.md` and ADR-003
- DO NOT add N-layer config merge across additional dirs
- TESTS REQUIRED
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST ensure `Load` / `WithWorkspaceRoot` uses only `root_dir` for workspace config file (not additional dirs)
- MUST provide or reuse helpers for iterating workspace roots (root + additional + global) for agent `AGENT.md` discovery as consumed by Resolver (coordinate with task_03 if shared)
- MUST remove `os.Getwd()` fallback in `resolveWorkspaceRoot()` (or equivalent) in favor of explicit roots from Resolver/session layer
- MUST update config package tests for new resolution assumptions
</requirements>

## Subtasks
- [x] 5.1 Audit `internal/config` for `Getwd` workspace fallbacks and remove per greenfield rules
- [x] 5.2 Align workspace root resolution helpers with Resolver outputs
- [x] 5.3 Add or update tests for workspace + global merge behavior for agents (not config) if logic lives in config
- [x] 5.4 Document any public API changes in code comments (no new markdown files unless required)

## Implementation Details

See TechSpec "Resolution Algorithm" step 6 and ADR-003. Prefer small, testable functions callable from `workspace.Resolver`.

### Relevant Files
- `internal/config/config.go` (and related loaders) — Workspace root and merge
- `internal/config/agent.go` — Agent def loading patterns

### Dependent Files
- `internal/workspace/resolver.go` — Calls into config loaders

### Related ADRs
- [ADR-003: Config from Root Only, Agents/Skills from All Dirs](adrs/adr-003.md)

## Deliverables
- Config loading behavior matches TechSpec asymmetry rules
- Updated unit tests in `internal/config/` **(REQUIRED)**
- No session/daemon `Getwd` dependency for workspace identity

## Tests
- Unit tests:
  - [x] Workspace config loads only from `root_dir/.agh/config.toml` when workspace root is set
  - [x] Additional dirs do not add extra config layers
  - [x] Agent merge order matches documented precedence when exercised via exported helpers
- Integration tests:
  - [x] Optional if covered by Resolver integration tests in task_03
- Test coverage target: >=80% for `internal/config`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/config`
- `make verify` passes
