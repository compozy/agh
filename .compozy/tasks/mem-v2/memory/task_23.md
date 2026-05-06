# Task Memory: task_23.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Hard-cut runtime narrative docs to the Slice 1 Memory v2 model: rewrote `core/memory/{index,system,scopes,dream,best-practices}.mdx`, `core/configuration/{config-toml,file-locations}.mdx`, `core/workspaces/resolver.mdx`, `core/sessions/index.mdx`, `core/hooks/index.mdx`, `core/extensions/index.mdx`, `core/skills/bundled.mdx`.
- Refreshed docs-truth/discovery guards in `packages/site/lib/runtime-docs-truth.test.ts` and `runtime-docs-discovery.test.ts`.
- Generated CLI/API reference pages were untouched (owned by task_24).

## Important Decisions

- Allowed forbidden-pattern matches per scope rule:
  - `memory.global_dir` mentions in `config-toml.mdx`, `file-locations.mdx`, `workspaces/config-overlays.mdx` — real config field at `internal/config/config.go:173` (still wired in defaults/validation/expandUserPath).
  - `memory.consolidated` mentions in `automation/index.mdx` and `automation/triggers.mdx` — real ActivationEnvelope kind emitted by `internal/automation/trigger.go:1043` (unaffected adjacent docs).
  - `memory_consolidations` table reference in `memory/system.mdx` — real catalog table created by `internal/memory/catalog.go:340`.
  - `memory.auto_consolidate` referenced in `agent-md.mdx` and `agents/definitions.mdx` only inside "Rejected" rows — explicit hard-cut documentation.
  - `consolidate`/`[memory.v2]`/`memory_read`/`memory_history`/`PUT /api/memory*` mentions in `memory/system.mdx`, `memory/dream.mdx`, `memory/scopes.mdx`, `configuration/config-toml.mdx` — explicit negative/hard-cut prose.
- Did not edit `sessions/lifecycle.mdx` "Background memory consolidation work" copy: the Go package `internal/memory/consolidation` still owns that runtime, so the prose remains runtime-truthful adjacent doc and is out of scope for this task.

## Learnings

- `bun run build` is what catches MDX/rolldown surprises that pure typecheck/test cannot. Run it on every doc-only branch even when typecheck and vitest pass.
- Site-side `bun run test` (and `typecheck`/`build`) implicitly run `bun run generate:openapi`, `source:generate`, and `content:generate` via the `pretest`/`pretypecheck`/`prebuild` scripts, so generated `runtime/api-reference/*.mdx` and Velite output are produced before vitest reads them. Don't commit those outputs.
- `runtime-docs-truth.test.ts` itself contains the very strings it forbids inside `expect(...).not.toMatch(...)` assertions; the forbidden-pattern scan picks them up under `packages/site/lib`. Treat lib matches as test infrastructure, not as doc regressions.

## Files / Surfaces

- `packages/site/content/runtime/core/memory/index.mdx`
- `packages/site/content/runtime/core/memory/system.mdx`
- `packages/site/content/runtime/core/memory/scopes.mdx`
- `packages/site/content/runtime/core/memory/dream.mdx`
- `packages/site/content/runtime/core/memory/best-practices.mdx`
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
- `packages/site/content/runtime/core/configuration/file-locations.mdx`
- `packages/site/content/runtime/core/workspaces/resolver.mdx`
- `packages/site/content/runtime/core/sessions/index.mdx`
- `packages/site/content/runtime/core/hooks/index.mdx`
- `packages/site/content/runtime/core/extensions/index.mdx`
- `packages/site/content/runtime/core/skills/bundled.mdx`
- `packages/site/lib/runtime-docs-truth.test.ts`
- `packages/site/lib/runtime-docs-discovery.test.ts`

## Errors / Corrections

- None this run. Two prior delegated docs runs hung between final test and tracking writes; this run finished both site-side gates and `make verify` cleanly.

## Validation Evidence

- `cd packages/site && bun run source:generate` — PASS (`[MDX] generated files in 13.4ms`).
- `cd packages/site && bun run typecheck` — PASS (`tsgo --noEmit` clean after pretypecheck OpenAPI/Velite generation).
- `cd packages/site && bun run test -- runtime-docs-truth runtime-docs-discovery` — PASS (`Test Files 2 passed (2)`, `Tests 13 passed (13)`).
- `cd packages/site && bun run build` — PASS (1137 static pages generated).
- `git diff --check` — clean.
- `make verify` — PASS (`DONE 8359 tests in 11.632s`, `OK: all package boundaries respected`).
- Forbidden-pattern scan over `packages/site/content/runtime/core` and `packages/site/lib` — every match falls under explicit hard-cut/negative context (memory docs + config-toml header) or unaffected truthful adjacent docs (automation events, `memory.global_dir`, `memory_consolidations` table, `auto_consolidate` rejected fields).

## Ready for Next Run

- Task 24 (CLI/API Reference and Discoverability Co-Ship) is the next iteration. Generated `cli-reference/`/`api-reference/` MDX is regenerated automatically by site `pre*` scripts; task_24 owns the cobra/openapi sources and any `make cli-docs` regeneration.
