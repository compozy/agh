# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Write the task_08 Sessions docs at the task-specified top-level runtime routes (`/runtime/sessions/*`) and make the runtime navigation point there.
- Replace the existing drifted session prose with code-verified lifecycle, resume/replay, events, and permissions documentation.

## Important Decisions
- Honor the task contract and move Sessions out of `runtime/core/sessions` into `runtime/sessions`, updating adjacent runtime navigation files as part of this task.
- Document resume behavior as ACP `session/load` first, with fresh-start fallback when stored ACP resources are missing; keep transcript reconstruction documented as a separate persisted replay surface exposed by `/api/sessions/:id/transcript`.
- Add explicit Mermaid support to `@agh/site` with a small client-side Mermaid component instead of relying on plain Mermaid code fences, because the current docs setup does not render Mermaid diagrams natively.

## Learnings
- Relevant QMD collections (`agh-compozy`, `agh-docs`) are empty, so markdown mining for this task falls back to repo-local `rg` after confirming QMD availability.
- Session SSE uses `GET /api/sessions/:id/stream`, not `/events/stream`, and reconnects use numeric persisted event sequences through `Last-Event-ID`.
- HTTP session creation requires `workspace` or absolute `workspace_path`; CLI `agh session new` defaults to the caller CWD when neither `--workspace` nor `--cwd` is provided.
- Built-in config defaults resolve agent permissions to `approve-all`; dream sessions also force `approve-all`.
- Per-agent permission overrides are resolved from `AGENT.md` frontmatter (`permissions` on `internal/config.AgentDef`), not from a `[[agents]]` table in `config.toml`.
- HTTP interactive approval is implemented at `POST /api/sessions/:id/approve`; the UDS transport registers the same route but currently responds `501 Not Implemented`.
- The current static export still serves session pages at `/runtime/core/sessions/*`, so this task needs a real IA move rather than a prose-only refresh.

## Files / Surfaces
- `packages/site/content/runtime/meta.json`
- `packages/site/content/runtime/index.mdx`
- `packages/site/content/runtime/core/meta.json`
- `packages/site/content/runtime/core/index.mdx`
- `packages/site/content/runtime/core/getting-started/web-ui.mdx`
- `packages/site/content/runtime/sessions/`
- `packages/site/components/docs/`
- `packages/site/mdx-components.tsx`
- `packages/site/package.json`

## Errors / Corrections
- Corrected the draft assumption that resume fallback automatically reconstructs and injects transcript history into the restarted agent process. The current code exposes transcript replay separately and falls back to a fresh ACP start when `session/load` resources are missing.
- Corrected the draft assumption that the default permission mode is `approve-reads`; current config resolution defaults new sessions to `approve-all` unless overridden.
- Corrected the draft SSE endpoint from `/api/sessions/:id/events/stream` to `/api/sessions/:id/stream`.
- Corrected the practical permissions example from a nonexistent `[[agents]]` TOML table to a real `AGENT.md` frontmatter override, matching `internal/config.AgentDef` and `Config.ResolveAgent`.
- Corrected the earlier routing assumption that `runtime/core/*` already served `/runtime/*`. The static export confirms `packages/site/content/runtime/core/*` builds to `/runtime/core/*`, so sessions must move to `packages/site/content/runtime/sessions/*` for the task's URLs to exist.
- Fresh task verification passed at the docs level (`turbo run build --filter=@agh/site`, browser QA on `/runtime/sessions/permissions/`), but the required repo-wide `make verify` is currently blocked by pre-existing `web/src/styles.test.ts` expectations that still assert the old DESIGN.md palette (`#121212`, `#1C1C1E`, `#2C2C2E`) while `packages/ui/src/tokens.css` already contains `#141312`, `#1e1c1b`, `#2e2c2b`.

## Ready for Next Run
- Decide whether to expand scope to reconcile the design-token/test mismatch or leave task tracking and commit blocked until that unrelated branch-level failure is resolved.
