# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Scaffold packages/site with Fumadocs — two-collection docs site with static export.

## Important Decisions

- Used `@/.source/server` import instead of `@/.source` — the generated `.source` directory lacks an index file, Turbopack can't resolve bare directory imports
- Used `toFumadocsSource()` method (not `toRuntime()` from older docs) — current fumadocs-mdx v14.2 API
- Used `fumadocs-ui/provider/next` import (not `fumadocs-ui/provider`) — provider is split by framework in current fumadocs-ui v16.7
- Added `trailingSlash: true` in next.config.mjs so static export produces `/runtime/index.html` instead of `/runtime.html`
- Custom search dialog with tag filtering (Runtime/Protocol) using `createSearchAPI('advanced')` combining both sources
- Disabled nav in DocsLayout (each docs layout) since we have a shared root-level Navbar component

## Learnings

- Fumadocs MDX v14+ generates `.source/server.ts`, `.source/browser.ts`, `.source/dynamic.ts` — no index barrel file. Import from `@/.source/server` for server-side source loaders
- Fumadocs UI v16+ split provider into framework-specific subpaths: `fumadocs-ui/provider/next`, `fumadocs-ui/provider/base`, etc.
- For static export with multiple sources, combine search indexes manually in `createSearchAPI('advanced', { indexes: [...] })`
- `@tailwindcss/postcss` is required as devDependency for Next.js + Tailwind CSS v4

## Files / Surfaces

### Created
- `packages/site/package.json` — @agh/site with fumadocs deps
- `packages/site/tsconfig.json` — bundler moduleResolution, @/* paths
- `packages/site/source.config.ts` — two defineDocs (runtime, protocol)
- `packages/site/lib/source.ts` — two loaders with baseUrl
- `packages/site/lib/layout.shared.ts` — baseOptions for future DocsLayout integration
- `packages/site/app/layout.tsx` — root layout with dark theme, search, navbar
- `packages/site/app/global.css` — fumadocs CSS + @agh/ui tokens + fd variable overrides
- `packages/site/app/page.tsx` — placeholder landing page with hero copy
- `packages/site/app/runtime/layout.tsx` — DocsLayout for runtime docs
- `packages/site/app/runtime/[[...slug]]/page.tsx` — runtime doc page
- `packages/site/app/protocol/layout.tsx` — DocsLayout for protocol docs
- `packages/site/app/protocol/[[...slug]]/page.tsx` — protocol doc page
- `packages/site/app/api/search/route.ts` — static search API combining both sources
- `packages/site/components/navbar.tsx` — shared nav (Home, Runtime, Protocol)
- `packages/site/components/search.tsx` — custom search dialog with tag filtering
- `packages/site/content/runtime/index.mdx` — placeholder runtime docs
- `packages/site/content/protocol/index.mdx` — placeholder protocol docs
- `packages/site/content/runtime/meta.json` — sidebar ordering
- `packages/site/content/protocol/meta.json` — sidebar ordering
- `packages/site/next.config.mjs` — static export + createMDX
- `packages/site/postcss.config.mjs` — @tailwindcss/postcss
- `packages/site/mdx-components.tsx` — fumadocs default MDX components

### Modified
- `package.json` (root) — added packages/site to workspaces
- `turbo.json` — added @agh/site#build with out/** outputs
- `Makefile` — added site-dev and site-build targets

## Errors / Corrections

- Initial import `@/.source` failed — changed to `@/.source/server`
- Initial `fumadocs-ui/provider` import failed — changed to `fumadocs-ui/provider/next`
- Initial `toRuntime()` API — changed to `toFumadocsSource()`
- `mdx/types` module not found — removed MDXComponents type import, simplified

## Ready for Next Run
