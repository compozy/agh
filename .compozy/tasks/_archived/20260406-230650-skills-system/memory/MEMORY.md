# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 is complete: `internal/skills` now exists with the base types, SKILL.md loader, and scan constraints that later skills tasks will build on.

## Shared Decisions
- The loader is intentionally lenient: unknown top-level frontmatter fields and missing descriptions only warn via `slog`, while malformed YAML and missing `name` fail parsing.
- Missing scan roots are treated as empty results so later registry tasks can probe optional skill directories without turning absent folders into hard errors.
- The registry caches only workspace-local overlays (`.agents` + `.agh`) per workspace and merges them with the current global snapshot on each `ForWorkspace()` call, so `RefreshGlobal()` does not need to invalidate workspace cache entries.
- `GlobalVersion` only advances when the newly loaded global skill map is materially different from the current one; watcher-triggered refreshes with no filesystem change leave the version stable.

## Shared Learnings
- Daemon boot resolves the global `.agents/skills` root via `HOME`; daemon tests that exercise skills boot should override `Daemon.getenv` instead of calling `t.Setenv`, because many daemon tests run with `t.Parallel()`.
- The local skill CLI follows the same `HOME/.agents/skills` resolution rule as the daemon registry path; CLI tests should override `commandDeps.getenv` so they do not scan the real user home during `t.Parallel()` runs.

## Open Risks

## Handoffs
