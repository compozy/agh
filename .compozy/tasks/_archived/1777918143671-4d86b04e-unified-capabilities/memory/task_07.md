# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Align `packages/site` protocol reference with the unified capability model from RFC 003 and ADRs 001/003. Remove `recipes.mdx` from steady-state nav, rewrite `message-kinds.mdx` and `examples.mdx`, and reframe `capability-discovery.mdx` around the three protocol roles (brief / rich / transfer).

## Important Decisions

- Deleted `packages/site/content/protocol/recipes.mdx` outright instead of keeping a supersession page, per GREENFIELD guidance and task requirement to avoid a first-class recipe page in steady-state nav.
- Extended `capability-discovery.mdx` with an explicit transfer section and a cross-link to `message-kinds.mdx#capability` so readers see one concept surfacing in three roles instead of two disjoint flows.
- Expanded the `examples.mdx` discovery/transfer scenario into six steps (greet → whois request → whois response → capability → direct → trace) to mirror the RFC 003 worked example while preserving the existing migration/check example.
- Updated Peer Card `artifacts_supported` examples across peer-discovery, message-kinds, capability-discovery, and examples to `["capability"]` to reflect the shared-memory invariant that local peers advertise capability transfer even with an empty capability catalog.
- Renamed the runtime `## Recipes` heading in `runtime/core/agents/capabilities.mdx` to `## Authoring patterns` to avoid reintroducing "recipe" as any live concept on the site.

## Learnings

- The landing page has static protocol-kind registries (`kind-chip.tsx`, `network-section.tsx`, `network-protocol-visual.tsx`, and the matching landing test). Shared memory already called out that these must stay in sync; kept them in scope for this task since they otherwise would advertise a `recipe` wire kind that no longer exists.
- Fumadocs/MDX build is sensitive to nav metadata consistency. Removing `recipes` from `meta.json` plus the index link fully cleared the nav; build succeeded with 191 static pages generated.
- The site test `landing.test.tsx` duplicates the `NetworkKind` union as a static list, so the kind rename had to land in both the type and the test fixture.

## Files / Surfaces

- `packages/site/content/protocol/meta.json` (removed `recipes`)
- `packages/site/content/protocol/recipes.mdx` (deleted)
- `packages/site/content/protocol/message-kinds.mdx` (recipe → capability section + summary row)
- `packages/site/content/protocol/capability-discovery.mdx` (three-role framing + transfer section)
- `packages/site/content/protocol/examples.mdx` (Example 3 rewritten as capability flow)
- `packages/site/content/protocol/index.mdx` (removed Recipes link, updated meta copy)
- `packages/site/content/protocol/overview.mdx` (updated kind list + adoption step)
- `packages/site/content/protocol/envelope.mdx` (kind list + schema enum)
- `packages/site/content/protocol/interactions.mdx` (recipe → capability in lifecycle rules)
- `packages/site/content/protocol/nats.mdx` (subject mapping notes)
- `packages/site/content/protocol/peer-discovery.mdx` (Peer Card rule + examples)
- `packages/site/content/runtime/core/agents/capabilities.mdx` (renamed "Recipes" heading)
- `packages/site/content/runtime/core/skills/bundled.mdx` (recipe → capability skill body copy)
- `packages/site/components/landing/primitives/kind-chip.tsx`
- `packages/site/components/landing/__tests__/landing.test.tsx`
- `packages/site/components/landing/network-section.tsx`
- `packages/site/components/landing/network-protocol-visual.tsx`
- `.compozy/tasks/unified-capabilities/task_07.md`
- `.compozy/tasks/unified-capabilities/_tasks.md`

## Errors / Corrections

- None required; site typecheck/test/build all passed on first run after rewrites, and repo `make verify` remained green.

## Ready for Next Run

- `bun run typecheck` in `packages/site` is clean.
- `bun run test` in `packages/site` is green (6 files / 30 tests).
- `bun run build` in `packages/site` produces 191 static pages without MDX errors.
- `make verify` at repo root is green (5439 tests, zero lint findings).
