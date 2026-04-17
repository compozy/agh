# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Write the four runtime skills documentation pages plus sidebar metadata for task_11: overview, SKILL.md reference, marketplace/how-to, and bundled catalog.
- Required evidence includes site build, browser QA of all touched routes, content checklist, full verification attempt, self-review, tracking updates, and a local commit only after clean verification.

## Important Decisions
- Document current implementation first; mark RFC-only or future marketplace/version behavior explicitly if source code does not implement it.
- Use `bunx turbo run build --filter=@agh/site` as the effective site build gate because shared workflow memory records the task selector `--filter=packages/site` as stale.

## Learnings
- Shared memory reports known unrelated full-repo `make verify` blockers in `web/src/styles.test.ts` token assertions and `@agh/extension-sdk` build drift; still run required verification and report actual evidence.
- QMD AGH collections were empty, so task-local QMD collections were added for `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/`; archived specs showed older skills-v1/v2 plans, which must be reconciled against current code.
- Current source implements marketplace commands (`search`, `install`, `remove`, `update`) and skill MCP/hook metadata. Docs should avoid saying those are future-only.
- Current source keeps skill bodies out of list/detail payloads and loads full content explicitly through `agh skill view` or `GET /api/skills/:name/content`.
- Bundled skills are exactly `agh-agent-setup`, `agh-memory-guide`, `agh-network`, and `agh-session-guide`; embedded bundled resources currently include only `SKILL.md` files, not assets or sidecar files.
- Pre-change baseline: `packages/site/content/runtime/skills/` was absent; runtime sidebar already had uncommitted sessions/agents/memory additions but no skills entry.
- AGH tolerates but warns on unknown AgentSkills/Claude-style top-level fields such as `allowed-tools`, `user-invocable`, and `argument-hint`; current AGH behavior is controlled by `metadata.agh.*` and config, not those fields.
- CLI validation: `go run ./cmd/agh skill ... --help`, `go run ./cmd/agh skill list --source bundled`, and `go run ./cmd/agh skill info agh-session-guide` run successfully. Runtime warnings came from unrelated user-level skills with unknown top-level fields and an unregistered current workspace lookup.
- Build validation: `bunx turbo run build --filter=packages/site` fails because the selector is stale; `bunx turbo run build --filter=@agh/site` passes and exports 150 static pages including the skills routes.
- Browser QA: `make site-dev` served all four touched routes (`/runtime/skills/overview/`, `/runtime/skills/skill-md/`, `/runtime/skills/marketplace/`, `/runtime/skills/bundled/`) with expected headings/content, no page errors, and a rendered Mermaid diagram on overview. Followed skills cross-links and a bundled-to-memory link. Dev server stopped after QA; port 3000 no longer listening.
- Full verification: `make verify` failed in pre-existing `web/src/styles.test.ts` token assertions (`#121212/#1C1C1E/#2C2C2E` expected, current `web/src/styles.css` has `#141312/#1e1c1b/#2e2c2b`). Do not mark task complete, update tracking to done, or commit until the full gate is clean.

## Files / Surfaces
- Created docs output: `packages/site/content/runtime/skills/{overview,skill-md,marketplace,bundled}.mdx` and `packages/site/content/runtime/skills/meta.json`.
- Navigation surface: `packages/site/content/runtime/meta.json`; preserve existing dirty sessions/agents/memory entries and add skills after memory.

## Errors / Corrections
- Corrected source hierarchy wording in `overview.mdx`: later rows override earlier rows.
- Browser QA correction: `agent-browser wait --url '**/runtime/skills/skill-md/'` hung after a successful click, so route validation used direct URL/content assertions instead.

## Ready for Next Run
- Docs files are authored and task-scoped checks passed; next blocker is whether to fix or wait on the unrelated web token test failure before tracking/commit.
