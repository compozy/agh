# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

## Shared Decisions

- `internal/kernel/memdir.Store` is self-contained and receives explicit global/workspace directory paths; it does not depend on kernel session types.
- `memdir.Store.Delete()` removes matching `(filename)` entries from the same-scope `MEMORY.md` if the index exists, so higher layers do not need separate delete-time index cleanup.
- `internal/kernel/dream.DreamService` holds the consolidation lock across `ShouldRun()` and `Run()`, but `Run()` can also acquire the lock directly so later manual/API paths can bypass the time/session gates without duplicating lock logic.
- The dream session gate counts completed sessions by scanning `~/.agh/sessions/*/meta.json` and using `stopped_at > lock.mtime`; it never imports kernel/session packages directly.
- `internal/prompt.MemoryContext` is the prompt-side contract for cross-session memory injection. Empty global/workspace/team strings omit the `MEMORY:` section entirely; when any field is present, prompt assembly inserts one memory block after `CONTEXT:` and before additional sections.
- Task_04 wires dream-mode sessions through the existing session manager via `sessionBootstrapPlan`: standard sessions still start supervisor+advisor, while dream sessions start exactly one `dream-worker` and skip recursive dream triggering on stop.
- Daemon-scoped memory HTTP endpoints are intentionally explicit about workspace scope: `GET/PUT/DELETE /api/memory...` require a `workspace` parameter for workspace-scope operations instead of inferring from daemon cwd.

## Shared Learnings

- `memdir.Store.Scan()` skips malformed or unreadable files and emits `slog.Warn` instead of failing the full scan, which keeps prompt assembly and listing resilient when one memory file is broken.
- `memdir.Store.LoadIndex()` enforces the 200-line / 25KB cap line-by-line and warns via `slog` when truncation occurs.
- `internal/kernel/dream.ConsolidationLock` reclaims a lock when the PID is dead, the lock age exceeds 1 hour, or the lock body is corrupt, while `Rollback()` restores the pre-acquire mtime so failed consolidation attempts do not advance the time gate.
- The prompt memory block carries only discovery-safe content: global `MEMORY.md`, workspace `MEMORY.md`, team memory snippets, and guidance on taxonomy/staleness/CLI usage. Full memory file bodies are still read on demand via `agh memory read`.
- The kernel dream ticker only evaluates `DreamService.ShouldRun()` when it can resolve a workspace. Ticker/manual triggers without an explicit workspace therefore depend on persisted stopped-session metadata to find the latest workspace.
- `Kernel.writeMemory()` must not call `memdir.Store.EnsureDirs()` for daemon-level writes: `Store.Write()` already creates the selected scope directory, and the redundant ensure call breaks global-only writes because the store intentionally has no workspace path in that case.
- The CLI keeps the daemon memory API explicit per-scope, but `agh memory list` aggregates global + workspace scopes when `--scope` is omitted, while `agh memory read` / `agh memory delete` auto-detect scope by filename and fail on ambiguity instead of guessing.
- `agh memory write` derives required frontmatter `name` values from the filename stem because the memdir store validates `name`, but the CLI contract intentionally exposes only `filename`, `type`, `description`, and body content.

## Open Risks

- `MEMORY.md` write/addition logic is still outside task_01 scope; later tasks that create or consolidate index entries must keep using the same `(filename)` link pattern for delete cleanup to stay accurate.
- The touched task_04 production surfaces reached 80.01% coverage (`2069/2586` statements across `internal/config/{config,home}.go` and `internal/kernel/{api,kernel,session_manager,types}.go`), but the monolithic `internal/kernel` package still sits below 80% because of pre-existing unrelated code. Future package-wide coverage gates would need broader kernel tests.

## Handoffs
