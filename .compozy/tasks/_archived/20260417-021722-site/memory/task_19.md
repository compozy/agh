# Task Memory: task_19.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Write the AGH Network v0 protocol documentation set under `packages/site/content/protocol/`: overview, envelope, message kinds, interactions, peer discovery, recipes, delivery, examples, plus `meta.json`.
- Source of truth priority: Task 19 + TechSpec Appendix B + RFC 003 for protocol semantics; current `internal/network/` for implemented AGH behavior.

## Important Decisions
- Use `AGH Network` in overview/product positioning and `AGH Network Protocol` in spec/reference contexts.
- Treat archived and QMD material as terminology/history only; reconcile all protocol details with RFC 003 and current source before documenting.
- Replace the existing nested protocol placeholder pages with the required flat files, because `protocol/overview/index.mdx` and `protocol/overview.mdx` both route to `/protocol/overview/`.

## Learnings
- QMD has the required collections: `agh-site-archived`, `agh-site-ledger`, and `agh-site-plans`. `agh-site-archived` surfaces the old network implementation techspec, but it uses older `space` terminology in places; current RFC/source use `channel`.
- Existing prior docs ledgers report an unrelated full-repo `make verify` blocker in `web/src/styles.test.ts`; verify current state fresh before any completion or commit claim.
- Site routing uses the `protocol` Fumadocs collection with `baseUrl: "/protocol"`; flat `packages/site/content/protocol/<slug>.mdx` files route to `/protocol/<slug>/`.
- The custom `<Mermaid chart={...} />` component is registered globally for MDX and should be used instead of raw mermaid code fences.
- Protocol examples should prefer current implementation field names: `protocol`, `id`, `kind`, `channel`, `from`, `to`, `interaction_id`, `reply_to`, `trace_id`, `causation_id`, `ts`, `expires_at`, `body`, `proof`, and `ext`.
- Current implementation validates `greet`, `whois`, `say`, `direct`, `recipe`, `receipt`, and `trace`; `direct`, `receipt`, and `trace` require `interaction_id`, while `whois` responses require `reply_to`.
- Current `whois.query` is a string, not a structured object; docs were corrected to use string query examples.
- Current lifecycle allows directed `recipe` messages with `interaction_id` to participate in opening/updating an interaction.
- Task literal build selector `bunx turbo run build --filter=packages/site` still fails because no workspace package is named `packages/site`; correct selector is `bunx turbo run build --filter=@agh/site`.
- Correct site build passed after the protocol docs edit and exported the new `/protocol/*` routes.
- Browser QA with `make site-dev` + `agent-browser` opened all eight protocol routes, followed sidebar navigation, and reported no browser errors.
- Required full `make verify` still fails outside this task in `web/src/styles.test.ts` due stale neutral-token expectations (`#121212`, `#1C1C1E`, `#2C2C2E`) versus current stylesheet values (`#141312`, `#1e1c1b`, `#2e2c2b`).

## Files / Surfaces
- `docs/rfcs/003_agh-network-v0.md`
- `internal/network/`
- `packages/site/content/protocol/`
- `packages/site/lib/source.ts`
- `packages/site/mdx-components.tsx`
- `packages/site/content/protocol/{overview,envelope,message-kinds,interactions,peer-discovery,recipes,delivery,examples}.mdx`
- `packages/site/content/protocol/meta.json`

## Errors / Corrections
- Corrected draft `whois` examples from object query syntax to the implementation's string `query` field.
- Full verification failure is unrelated to protocol docs; do not mark task complete or commit until the branch-level gate is clean.

## Ready for Next Run
- Protocol docs authored and task-scoped checks passed; blocked from task completion/tracking/commit by unrelated full-repo verification failure.
