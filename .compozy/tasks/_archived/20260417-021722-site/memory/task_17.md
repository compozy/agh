# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create six Diataxis reference pages under `packages/site/content/runtime/reference/` for config.toml, AGENT.md, SKILL.md, mcp.json, env vars, and file locations, plus `meta.json`.
- Source of truth is current code/RFCs: `internal/config/`, env/path handling, daemon/CLI surfaces, and RFC 001/002 reconciled against implementation.

## Important Decisions
- Add a dedicated `/runtime/reference/` docs section for this task. Existing `core/configuration`, `agents`, and `skills` pages are adjacent context, not replacements.
- Treat QMD/archived/planning material as advisory only; current source wins when there is drift.

## Learnings
- Shared memory says the task's literal site build selector `--filter=packages/site` is stale; the current package selector is `--filter=@agh/site`.
- Shared memory records a known unrelated full-repo `make verify` blocker in `web/src/styles.test.ts`; this task still needs fresh verification before completion.
- QMD searches across `agh-site-archived`, `agh-site-ledger`, and `agh-site-plans` returned no useful prior-plan material for this reference set; current source/RFC docs remain the authority.
- Current `config.toml` source includes sections beyond the task examples: `[http]`, `[limits]`, `[session.limits]`, `[permissions]`, `[providers.<name>]`, `[observability]`, `[observability.transcripts]`, `[log]`, `[automation]`, `[[automation.jobs]]`, `[[automation.triggers]]`, `[[hooks.declarations]]`, and `[network]`. There is no implemented `[environments.*]` section; unknown TOML keys fail loading.
- Current `AGENT.md` parser accepts only `name`, `provider`, `command`, `model`, `tools`, `permissions`, `mcp_servers`, `hooks`, plus a required non-empty Markdown body. RFC 001 fields such as `description`, `skills.*`, and `memory.*` are rejected today.
- Current `SKILL.md` loader maps only `name`, `description`, `version`, and `metadata`; `metadata.agh.mcp_servers` and `metadata.agh.hooks` are implemented, while `metadata.agh.memory_tags` and AgentSkills-style fields such as `allowed-tools` are not runtime controls today.
- `mcp.json` accepts top-level `mcpServers` and `mcp_servers`; unknown JSON fields and trailing JSON fail. Sidecar collisions replace the whole server object, while TOML `[[mcp_servers]]` overlays merge fields by name.

## Files / Surfaces
- Planned docs output: `packages/site/content/runtime/reference/*`.
- Existing adjacent docs reviewed: `packages/site/content/runtime/core/configuration/index.mdx`, `packages/site/content/runtime/agents/definitions.mdx`, `packages/site/content/runtime/skills/skill-md.mdx`, workspace config overlay docs.
- Required navigation update: `packages/site/content/runtime/meta.json`.
- Created six reference MDX pages plus `packages/site/content/runtime/reference/meta.json`; updated `packages/site/content/runtime/meta.json` to add the `reference` section.

## Errors / Corrections
- Initial parallel QMD search hit a transient SQLite lock for `agh-site-plans`; sequential retry completed and returned no results.
- `bunx turbo run build --filter=@agh/site` passed after docs were added.
- Mandatory browser QA with `make site-dev` and `agent-browser` returned HTTP 200/rendered titles for all six new routes and verified sidebar/internal navigation to `/runtime/reference/config-toml/` and `/runtime/reference/mcp-json/`.
- Full `make verify` was run and failed in pre-existing `web/src/styles.test.ts` assertions for `--color-canvas`, `--color-surface`, and `--color-surface-elevated` design token values. Do not mark task complete or commit until the full gate is clean.
- Follow-up investigation found the mismatch predates this docs task: commit `5c5c27ae` changed `packages/ui/src/tokens.css` to the warmer `#141312` / `#1e1c1b` / `#2e2c2b` values, while `web/src/styles.test.ts` and `DESIGN.md` still lock the older neutral `#121212` / `#1C1C1E` / `#2C2C2E` values.

## Ready for Next Run
- If continuing this task, either resolve the unrelated design-token test mismatch or get explicit direction on completion policy; then rerun `make verify`, update task tracking, and commit if clean.
