# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Create `packages/ui` as `@agh/ui` workspace package with design tokens and 12 base components extracted from `web/`.

## Important Decisions

- **Exports map uses source .ts files directly** — no build step producing `dist/`. The package is consumed via workspace protocol with bundler moduleResolution. `tsgo --noEmit` is the "build" (type-checking only).
- **`"use client"` only on badge.tsx and progress.tsx** — badge uses `useRender` hook from base-ui; progress uses base-ui Progress primitives with internal state. Label was stripped of `"use client"` since it's a pure `<label>` with no hooks.
- **turbo.json unchanged** — the generic `build` task already picks up `@agh/ui`'s `build` script. Warning about no output files is expected for noEmit packages.

## Learnings

- `@base-ui/react` subpath imports (`/button`, `/separator`, `/progress`, `/input`, `/merge-props`, `/use-render`) are the component primitive dependencies.
- `shadcn/tailwind.css` import is in `web/src/styles.css` but NOT in `tokens.css` — it's web-app-specific (shadcn CLI compatibility).

## Files / Surfaces

- `packages/ui/package.json` — @agh/ui with exports map
- `packages/ui/tsconfig.json` — strict, noEmit, bundler resolution
- `packages/ui/src/tokens.css` — full DESIGN.md tokens extracted from web/src/styles.css
- `packages/ui/src/tailwind-preset.ts` — spacing, borderRadius, fontFamily scales
- `packages/ui/src/lib/utils.ts` — cn() utility
- `packages/ui/src/components/` — 12 components (button, badge, card, input, label, separator, skeleton, spinner, alert, progress, table, kbd)
- `packages/ui/src/index.ts` — barrel exports
- `package.json` (root) — added `packages/ui` to workspaces

## Errors / Corrections

None.

## Ready for Next Run

- task_02 should update `web/` to import tokens and components from `@agh/ui` instead of local copies.
- `shadcn/tailwind.css` import stays in `web/src/styles.css` — it's web-app-specific.
