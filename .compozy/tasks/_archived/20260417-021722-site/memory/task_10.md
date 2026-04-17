# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create the runtime memory docs section for task_10:
  `system.mdx`, `scopes.mdx`, `dream.mdx`, `best-practices.mdx`, and `meta.json`.
- Acceptance requires source-grounded content, site build, browser QA on all touched routes, task tracking updates, and a local commit only after clean verification/self-review.

## Important Decisions
- Document current implementation first. RFC 001 is draft/future-facing and must be reconciled against source before using examples.
- Current runtime memory is dual-scope global/workspace only. Do not document RFC 001 agent-scoped memory fields or `.agents/<name>/memory/` as implemented behavior.
- Use existing docs MDX helpers (`OperatorNote`, `RouteList`, `RouteRow`, `Mermaid`) without adding new site components.
- Add `memory` to `packages/site/content/runtime/meta.json` after `agents`, working with the existing uncommitted agents/sessions docs changes.

## Learnings
- Shared workflow memory says the correct site build selector is `bunx turbo run build --filter=@agh/site`; the task file's literal `--filter=packages/site` selector is stale.
- Existing agents docs use explicit RFC drift notes and `<Mermaid />` for diagrams; memory docs should follow that local pattern.
- Pre-change baseline: no `packages/site/content/runtime/memory/` docs section exists yet.
- Current memory taxonomy is closed: `user`, `feedback`, `project`, and `reference`.
- Default write scope is type-driven: `user`/`feedback` -> global, `project`/`reference` -> workspace.
- Global memory defaults to `~/.agh/memory/` or `$AGH_HOME/memory/`; workspace memory lives at `<workspace>/.agh/memory/`.
- Prompt assembly injects only prompt-safe `MEMORY.md` indexes ahead of the agent prompt; full files are read on demand through `agh memory read`.
- Dream consolidation defaults are enabled, agent `general`, 24 minimum hours, 3 minimum completed sessions, and 30 minute check interval.
- qmd lexical search over the available `ai-memory` collection returned no results; qmd semantic `query` crashed locally with sqlite-vec `no such module: vec0`, so required prior-note mining is covered by local `rg`.
- Task-scoped build verification passed with `bunx turbo run build --filter=@agh/site` after the docs were written; the stale task selector `--filter=packages/site` fails because no package has that name.
- Browser QA with `agent-browser` loaded `/runtime/memory/system/`, `/runtime/memory/scopes/`, `/runtime/memory/dream/`, `/runtime/memory/best-practices/`, and followed the `Memory Write CLI` link to `/runtime/cli-reference/memory/write/`; each route returned visible content and no browser errors.
- Full `make verify` remains blocked by the pre-existing `web/src/styles.test.ts` design-token mismatch: tests expect `#121212`, `#1C1C1E`, and `#2C2C2E`, while the current stylesheet contains `#141312`, `#1e1c1b`, and `#2e2c2b`.

## Files / Surfaces
- Docs output: `packages/site/content/runtime/memory/{system,scopes,dream,best-practices}.mdx` and `packages/site/content/runtime/memory/meta.json`.
- Navigation edit: `packages/site/content/runtime/meta.json` adds `memory` after existing `agents` worktree changes.
- Supporting source/docs under review: `internal/memory/`, `internal/memory/consolidation/`, `internal/config/`, `internal/cli/memory.go`, `internal/api/*/memory*`, `internal/session/`, `internal/daemon/`, and `docs/rfcs/001_agent-md-with-skills-memory.md`.

## Errors / Corrections
- QMD `agh-compozy` / `agh-docs` collections are empty; used qmd status/search plus local markdown search over `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` to mine prior notes.
- Current worktree contains unrelated uncommitted task_08/task_09/site shell changes; do not revert or overwrite them.
- Task tracking and automatic commit are intentionally not completed while `make verify` exits non-zero on the unrelated token tests.

## Ready for Next Run
- Docs implementation and task-scoped verification are done, but final completion is blocked until the unrelated token mismatch in `web/src/styles.test.ts` / `packages/ui/src/tokens.css` is resolved or the full gate is otherwise clean.
