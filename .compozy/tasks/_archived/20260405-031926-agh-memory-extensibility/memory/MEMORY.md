# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 completed: `internal/memory` now provides the file-backed store core with validated frontmatter, dual-scope directory handling, index truncation, staleness helpers, and unit coverage above the task threshold.
- Task 02 completed: config/session/store now expose memory config defaults, `HomePaths.MemoryDir`, `session.SessionType`, `session.PromptAssembler`, and persisted `session_type` data in both `meta.json` and the global sessions index.
- Task 04 completed: daemon boot/runtime now initialize and expose a shared `memory.Store`, inject `memory.Assembler` into the session manager, and wire the dream service/ticker/session-stop trigger into the daemon lifecycle.

## Shared Decisions
- `internal/memory.Store` stays immutable per configured scope: workspace-scoped callers must bind a clone with `Store.ForWorkspace(workspaceRoot)` instead of mutating a shared store instance. This keeps daemon/API use race-safe across concurrent workspaces.
- `Store.EnsureDirs()` creates the global memory directory unconditionally and only creates the workspace directory when the store has already been bound to a workspace. This lets daemon boot initialize the global store before any workspace is known.
- ACP still has no dedicated system-prompt field in the protocol, so task 04 added a repo-local runtime convention: `session.Manager` passes the assembled startup prompt via `acp.StartOpts.SystemPrompt`, and the ACP driver prepends it exactly once to the first prompt turn as session instructions.

## Shared Learnings
- Validation failures in the memory package are wrapped with the exported sentinel `memory.ErrValidation`, which future API/CLI tasks can map to HTTP 400 or CLI usage errors without string matching.
- Task 03 dream session counting accepts both an explicit `stopped_at` timestamp and the current persisted `state=stopped` + `updated_at` metadata shape. Future daemon/API wiring should preserve at least one of those completion signals or dream gating will undercount completed sessions.
- Future memory API/CLI wiring should reuse `daemon.RuntimeDeps.MemoryStore` instead of constructing a second store instance; task 04 already bootstraps and directory-initializes the shared store for server factories.
- Dream consolidation checks now enter through two daemon paths: the periodic ticker (`memory.dream.check_interval`) and a non-blocking queue triggered by `notifierFanout.OnSessionStopped` for non-dream sessions.
- Any HTTP/UDS server or integration harness that enables the memory routes must inject both the shared `MemoryStore` and the dream trigger dependency. Registering `/api/memory` without those deps leaves the routes present but causes runtime 500s such as `memory store is not configured`.

## Open Risks
- `internal/memory` still has a package-local `memoryDirName = "memory"` constant. Future memory tasks should switch to the centralized `config.MemoryDirName` path source instead of keeping parallel directory-name definitions.

## Handoffs
- Future tasks should reuse `Store.ForWorkspace(workspaceRoot)` for workspace `Scan`, `LoadIndex`, `Read`, `Write`, and `Delete` calls rather than sharing one mutable store across requests or session assembly paths.
