# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Build 8 landing page section components in `packages/site/components/landing/`, compose them in `app/page.tsx`, add snapshot tests.

## Important Decisions

- Used `buttonVariants()` from `@agh/ui` for CTA links rather than raw class strings — stays consistent with the design system.
- SVG architecture diagram uses hardcoded hex values (not CSS vars) because SVG `fill`/`stroke` attributes have limited CSS var support in static SVG — values match DESIGN.md tokens exactly.
- Comparison section has both a desktop table view and mobile card view for responsive behavior.
- Added vitest + @testing-library/react as devDeps to @agh/site for component snapshot tests.
- Added `packages/site/vitest.config.ts` to root vitest workspace projects array.

## Learnings

- `@base-ui/react` is hoisted by bun but only in `packages/ui/node_modules` — vitest alias for `@agh/ui` resolves transitively through the ui package.
- Next.js 16 + Turbopack builds the site with 122 static pages in ~3s total.

## Files / Surfaces

- `packages/site/components/landing/hero.tsx` — Hero section
- `packages/site/components/landing/two-pillars.tsx` — Runtime vs Protocol split
- `packages/site/components/landing/how-it-works.tsx` — 3-step install flow
- `packages/site/components/landing/runtime-features.tsx` — 8 feature cards
- `packages/site/components/landing/protocol-section.tsx` — 7 message kinds + lifecycle
- `packages/site/components/landing/architecture.tsx` — SVG architecture diagram
- `packages/site/components/landing/comparison.tsx` — AGH vs typical harness table
- `packages/site/components/landing/final-cta.tsx` — Closing CTA
- `packages/site/components/landing/index.ts` — Barrel export
- `packages/site/app/page.tsx` — Composes all sections
- `packages/site/components/landing/__tests__/landing.test.tsx` — 16 tests (8 content + 8 snapshot)
- `packages/site/vitest.config.ts` — Test configuration
- `packages/site/package.json` — Added test deps + script
- `vitest.config.ts` (root) — Added site project

## Errors / Corrections

None.

## Ready for Next Run
