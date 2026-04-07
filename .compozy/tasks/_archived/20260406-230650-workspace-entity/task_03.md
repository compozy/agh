---
status: completed
title: Resolver implementation with cache and tests
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Resolver implementation with cache and tests

## Overview

Implement `workspace.Resolver` with `Resolve`, `ResolveOrRegister`, registration helpers, mtime-based caching with TTL eviction, and the full resolution algorithm (config cascade, agent merge, skill path collection). Include unit tests with a mock `WorkspaceStore` and integration tests with real SQLite and temp directories per TechSpec.

<critical>
- ALWAYS READ `_techspec.md` and ADR-003 (asymmetric discovery)
- REFERENCE TECHSPEC "Resolution Algorithm", "Caching", "Logging"
- TESTS REQUIRED — unit + integration coverage in this task
- GREENFIELD: Não sacrificar qualidade por retrocompatibilidade — preferir desenho limpo e mudanças breaking alinhadas ao TechSpec; evitar migrações, shims ou código defensivo só para estado/API antiga.
</critical>

<requirements>
- MUST implement `Resolve` routing: `ws_` prefix → ID, absolute path → canonical path lookup, else name lookup
- MUST implement `ResolveOrRegister` with canonical path, auto-name dedup (`-2`, `-3`, …) per TechSpec
- MUST validate root exists; return `ErrWorkspaceRootMissing` when directory is gone
- MUST re-evaluate symlinks on resolve and update stored `root_dir` when canonical path changes
- MUST implement mtime+size snapshot cache with 10-minute TTL idle eviction per TechSpec
- MUST scan `.agh/agents` and `.agh/skills` across root + additional dirs + global home with merge order per ADR-003
- MUST load config cascade from root only for workspace config (additional dirs do not contribute config)
- MUST add structured `slog` logging for register/resolve/cache events per TechSpec "Monitoring and Observability"
- MUST include unit tests (mock store) and integration tests (`//go:build integration` if required by repo convention) with real filesystem + SQLite
</requirements>

## Subtasks
- [x] 3.1 Implement `NewResolver` with functional options (`config` loader injection, logger, home paths)
- [x] 3.2 Implement store-backed CRUD used by API/CLI: `Register`, `Unregister`, `Update`, `List`, `Get` as needed by TechSpec
- [x] 3.3 Implement `Resolve` and `ResolveOrRegister` per algorithm pseudocode
- [x] 3.4 Implement cache invalidation and `Invalidate` for programmatic bust
- [x] 3.5 Add `resolver_test.go` and `resolver_integration_test.go` per TechSpec "Testing Approach"
- [x] 3.6 Ensure no double-scan of skills remains once task_07 lands (coordinate API surface)

## Implementation Details

See TechSpec "Resolver", "Caching", "Resolution Algorithm", and "Testing Approach". Mock `WorkspaceStore` in unit tests; use `t.TempDir()` for integration.

### Relevant Files
- `internal/config/` — `Load` with `WithWorkspaceRoot`, `HomePaths`, agent def loading
- `internal/store/global_db.go` — Real store for integration tests
- `internal/skills/registry.go` — Reference for mtime cache pattern (skills registry)

### Dependent Files
- `internal/daemon/` — Wires Resolver in task_06
- `internal/session/manager.go` — Calls Resolver in task_04

### Related ADRs
- [ADR-001: Resolver with Persistent Backing](adrs/adr-001.md) — Resolver vs manager semantics
- [ADR-003: Config from Root Only, Agents/Skills from All Dirs](adrs/adr-003.md) — Merge precedence

## Deliverables
- `internal/workspace/resolver.go` and `options.go` (or equivalent) with full behavior
- `resolver_test.go` with mock store covering all TechSpec unit scenarios
- `resolver_integration_test.go` (or tagged file) for end-to-end resolve with real DB + dirs
- Structured logging hooks for observability

## Tests
Unit tests:
- [x] Resolve by ID, name, and absolute path routes correctly
- [x] `ResolveOrRegister` returns existing row when path already registered
- [x] Auto-register creates `ws_` ID and dedupes name with `-2` suffix
- [x] Cache hit when mtimes unchanged; miss when `config.toml` or `agents/`/`skills/` changes
- [x] Missing root returns `ErrWorkspaceRootMissing`
- [x] Symlink target change updates stored `root_dir` and re-resolves
- [x] Local agent definition overrides global by name
Integration tests:
- [x] Register → resolve → merged agents/skills match expected paths in `t.TempDir()`
- [x] Symlink workspace: change target, re-resolve updates state
- Test coverage target: >=80% for `internal/workspace`
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for `internal/workspace`
- `make verify` passes
