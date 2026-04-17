---
status: completed
title: "Scaffold packages/site with Fumadocs"
type: infra
complexity: medium
dependencies: [task_01]
---

# Task 03: Scaffold packages/site with Fumadocs

## Overview

Create the Fumadocs-based documentation site at `packages/site/` with two content collections (runtime and protocol), DESIGN.md theming via `@agh/ui`, Orama search, and static export configuration. This is the documentation site shell — it produces a buildable Next.js app with route groups for `/runtime/` and `/protocol/`, placeholder content in both collections, and the shared navigation bar. See TechSpec "System Architecture > Component Overview" for the full directory structure and "Implementation Design > Core Interfaces" for the source configuration.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- **IMPECCABLE (non-blocking, Compozy-safe)** — For root layout, nav, Fumadocs theme overrides, and placeholder home: read and apply the **impeccable** skill (`/impeccable` — already in Claude Code; no installs). Use `.impeccable.md` when it exists; **never** run `/impeccable teach` during automated runs. Map DESIGN.md/TechSpec tokens into Fumadocs/CSS variables; respect OKLCH, typography, spatial rhythm, and **absolute_bans**. Full landing belongs to task_05; avoid generic template chrome here.
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST initialize a Fumadocs app in `packages/site/` using `npx create-fumadocs-app` as a starting point, then customize
- MUST create `source.config.ts` with two `defineDocs()` calls: `runtime` (dir: `content/runtime`) and `protocol` (dir: `content/protocol`) per TechSpec and ADR-003
- MUST create `lib/source.ts` with two loaders: `runtimeDocs` (baseUrl: `/runtime`) and `protocolDocs` (baseUrl: `/protocol`)
- MUST create route group `app/runtime/[[...slug]]/page.tsx` using `runtimeDocs` loader
- MUST create route group `app/protocol/[[...slug]]/page.tsx` using `protocolDocs` loader
- MUST create `app/runtime/layout.tsx` and `app/protocol/layout.tsx` with Fumadocs `DocsLayout` wrapping each collection's sidebar
- MUST create `app/layout.tsx` as the root layout with DESIGN.md dark theme applied (canvas #121212 background, Inter + JetBrains Mono fonts, dark mode class)
- MUST import `@agh/ui/tokens.css` in the root layout or global CSS to apply shared design tokens
- MUST create a shared navigation bar component linking to `/`, `/runtime`, and `/protocol`
- MUST create `app/page.tsx` as a minimal placeholder landing page (full implementation in task_05)
- MUST configure `next.config.mjs` with `output: 'export'` for static site generation
- MUST create at least one placeholder MDX page in each collection:
  - `content/runtime/index.mdx` with title, description frontmatter
  - `content/protocol/index.mdx` with title, description frontmatter
- MUST configure Orama search to index both collections, with collection badges in results ("Runtime" / "Protocol")
- MUST create `packages/site/package.json` with name `@agh/site`, dependencies on `@agh/ui`, `fumadocs-core`, `fumadocs-mdx`, `fumadocs-ui`, `next`, `react`, `react-dom`
- MUST add `packages/site` to root `package.json` workspaces array
- MUST add `packages/site` to `turbo.json` with output `out/**` (not `.next/**` — static export outputs to `out/`)
- MUST add Makefile targets: `site-dev` (runs `next dev` in packages/site), `site-build` (runs `next build`)
- MUST configure `meta.json` files in content directories for sidebar ordering
- MUST override Fumadocs default theme colors to match DESIGN.md (accent #E8572A, backgrounds #121212/#1C1C1E/#2C2C2E, text colors, flat depth model)
</requirements>

## Subtasks

- [x] 3.1 Initialize Fumadocs app in `packages/site/` via `create-fumadocs-app`
- [x] 3.2 Create `packages/site/package.json` with `@agh/site` name and dependencies
- [x] 3.3 Create `source.config.ts` with two `defineDocs()` sources
- [x] 3.4 Create `lib/source.ts` with two loaders (`runtimeDocs`, `protocolDocs`)
- [x] 3.5 Create route group `app/runtime/[[...slug]]/page.tsx` and `layout.tsx`
- [x] 3.6 Create route group `app/protocol/[[...slug]]/page.tsx` and `layout.tsx`
- [x] 3.7 Create `app/layout.tsx` root layout with DESIGN.md dark theme
- [x] 3.8 Import `@agh/ui/tokens.css` and override Fumadocs theme variables
- [x] 3.9 Create shared navigation bar component (links to /, /runtime, /protocol)
- [x] 3.10 Create `app/page.tsx` placeholder landing page
- [x] 3.11 Create placeholder content: `content/runtime/index.mdx` and `content/protocol/index.mdx`
- [x] 3.12 Add `meta.json` files for sidebar ordering in both content directories
- [x] 3.13 Configure `next.config.mjs` with `output: 'export'`
- [x] 3.14 Configure Orama search for both collections
- [x] 3.15 Add `packages/site` to root `package.json` workspaces
- [x] 3.16 Add `packages/site` to `turbo.json` with `out/**` outputs
- [x] 3.17 Add `site-dev` and `site-build` Makefile targets
- [x] 3.18 Verify `turbo run build --filter=@agh/site` succeeds
- [x] 3.19 Verify `make site-build` succeeds

## Implementation Details

See TechSpec sections: "System Architecture > Component Overview", "Implementation Design > Core Interfaces", "Operational Considerations > Deployment Mode", "Operational Considerations > Search and Indexing".

The two-source configuration is the critical architectural decision (ADR-003). Each source gets its own `defineDocs()` in `source.config.ts` (which is the Fumadocs MDX config file — it does NOT set `baseUrl`). The `baseUrl` is set in `lib/source.ts` where the loaders are created. Each route group (`app/runtime/`, `app/protocol/`) consumes its own loader and renders its own sidebar.

Fumadocs theme customization: Fumadocs uses CSS variables for theming. Override these in the root layout CSS to match DESIGN.md tokens from `@agh/ui`. Key overrides: `--fd-background` to `--color-canvas`, `--fd-foreground` to `--color-text-primary`, `--fd-primary` to `--color-accent`, and sidebar/card backgrounds to surface colors. The flat depth model means removing any box-shadow defaults from Fumadocs.

Static export: `output: 'export'` in `next.config.mjs` produces an `out/` directory. Turbo cache must target `out/**`, not `.next/**`. The dev server (`next dev`) still uses the normal Next.js dev mode.

### Relevant Files

- `turbo.json` — Add packages/site workspace; current tasks config at lines 14-32
- `package.json` (root) — Add `packages/site` to workspaces array; current workspaces at line 3-8
- `Makefile` — Add `site-dev` and `site-build` targets; current web targets at lines 42-61
- `packages/ui/src/tokens.css` — Imported by site for design tokens (task_01 deliverable)
- `packages/ui/src/components/` — Base components available for site (task_01 deliverable)
- `DESIGN.md` — Canonical design reference for theme overrides

### Dependent Files

- `packages/site/content/runtime/` — Will be populated with docs content (tasks 06-18)
- `packages/site/content/protocol/` — Will be populated with protocol spec (tasks 19-21)
- `packages/site/content/runtime/reference/cli/` — Will receive generated CLI docs (task_04)
- `packages/site/app/page.tsx` — Will be replaced with full landing page (task_05)
- `packages/site/components/landing/` — Will be created for landing page sections (task_05)

### Related ADRs

- [ADR-001: Fumadocs as Documentation Framework](adrs/adr-001.md) — Fumadocs chosen for React/Next.js alignment and component sharing
- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Two `defineDocs()` sources for runtime and protocol with separate sidebars and URL prefixes

## Deliverables

- `packages/site/` directory with complete Fumadocs scaffolding
- `packages/site/package.json` with `@agh/site` name and all dependencies
- `packages/site/source.config.ts` with two doc sources
- `packages/site/lib/source.ts` with two loaders
- `packages/site/app/layout.tsx` root layout with DESIGN.md theming
- `packages/site/app/page.tsx` placeholder landing page
- Route groups: `app/runtime/[[...slug]]/` and `app/protocol/[[...slug]]/`
- Placeholder content: `content/runtime/index.mdx` and `content/protocol/index.mdx`
- Orama search configured for both collections
- `next.config.mjs` with `output: 'export'`
- Updated root `package.json` workspaces
- Updated `turbo.json` with `out/**` outputs
- New Makefile targets: `site-dev`, `site-build`
- Build passing: `turbo run build --filter=@agh/site`

## Tests

- Build verification:
  - [x] `turbo run build --filter=@agh/site` completes without errors
  - [x] `make site-build` completes without errors
  - [x] Static export produces `packages/site/out/` directory with HTML files
  - [x] `out/index.html` exists (landing page)
  - [x] `out/runtime/index.html` exists (runtime docs index)
  - [x] `out/protocol/index.html` exists (protocol docs index)
- Routing:
  - [x] `/runtime` route renders the runtime docs placeholder with sidebar
  - [x] `/protocol` route renders the protocol docs placeholder with sidebar
  - [x] `/` route renders the placeholder landing page
  - [x] Navigation bar links work between all three routes
- Theming:
  - [x] Root layout applies `dark` class and canvas background (#121212)
  - [x] Fumadocs theme variables overridden to match DESIGN.md tokens
  - [x] Inter and JetBrains Mono fonts load correctly
  - [x] No default Fumadocs shadows visible (flat depth model)
- Search:
  - [x] Orama search indexes content from both collections
  - [x] Search results display collection badge ("Runtime" or "Protocol")
- Monorepo integration:
  - [x] `turbo run build` (full monorepo) succeeds (pre-existing @agh/extension-sdk error unrelated)
  - [x] `make web-build` still passes (no regressions)
- Test coverage target: N/A (infrastructure scaffolding — verified by build and manual route checks)

## Success Criteria

- `make site-build` succeeds and produces `packages/site/out/` with static HTML
- Both route groups (`/runtime`, `/protocol`) render with Fumadocs layout and sidebar
- DESIGN.md theme applied: dark background, accent color, correct fonts, flat depth
- Orama search works across both collections
- Root monorepo build (`turbo run build`) succeeds with no regressions
- Makefile targets `site-dev` and `site-build` work
- `turbo.json` and root `package.json` updated correctly
