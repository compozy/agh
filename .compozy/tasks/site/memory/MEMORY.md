# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- task_01 complete: `packages/ui` (`@agh/ui`) created with design tokens + 12 base components.
- task_02 complete: `web/` migrated to consume `@agh/ui` — tokens imported, 12 components deleted, all imports updated.
- task_03 complete: `packages/site` (`@agh/site`) scaffolded with Fumadocs — two-collection docs site (runtime + protocol), DESIGN.md theming, Orama search with tags, static export producing `out/`.
- task_04 complete: CLI doc generation via Cobra `GenMarkdownTree` + Go post-processor (`internal/cli/docpost/`). Hidden `doc` subcommand, `make cli-docs` target, 108 MDX files generated.
- task_05 complete: Landing page with 8 section components in `packages/site/components/landing/`. Snapshot tests via vitest + @testing-library/react. 16 tests, build passes (122 static pages).
- task_06 complete: Three overview doc pages (what-is-agh, architecture, comparison) under `packages/site/content/runtime/overview/`. ASCII architecture diagram (no Mermaid support). Build passes (125 static pages).
- task_07 complete: Four Getting Started tutorial pages (installation, quick-start, first-agent, web-ui) under `packages/site/content/runtime/getting-started/`. All CLI commands verified against codebase. Build passes (129 static pages).

## Shared Decisions

- **@agh/ui exports source .ts files** — no dist build. Consumed via workspace protocol with bundler moduleResolution. `tsgo --noEmit` for type-checking only.
- **`shadcn/tailwind.css` stays in web/** — it's web-app-specific, not part of the shared token layer.
- **Fumadocs import from `@/.source/server`** — generated `.source` directory has no barrel index; use explicit `/server` subpath.
- **Fumadocs UI v16+ provider** — import from `fumadocs-ui/provider/next`, not `fumadocs-ui/provider`.
- **Static export with `trailingSlash: true`** — produces `/runtime/index.html` paths instead of `/runtime.html`.

## Shared Learnings

- `@base-ui/react` is the UI primitive library used by shadcn components (button, input, separator, badge, progress).
- Fumadocs MDX v14+ uses `toFumadocsSource()` (not `toRuntime()`) for source loader integration.
- For multi-source search in static export, combine indexes manually via `createSearchAPI('advanced', { indexes: [...] })`.
- `@tailwindcss/postcss` required as devDependency for Next.js + Tailwind CSS v4.
- MDX requires escaping `<` and `{` in non-code text (`\<`, `\{`). Tab-indented blocks must be converted to fenced code blocks for MDX compatibility.
- `@agh/site` has vitest configured (`packages/site/vitest.config.ts`) with jsdom environment and `@testing-library/react`. Added to root vitest workspace projects.
- Fumadocs does not render `mermaid` code blocks natively — use ASCII art in plain code blocks for architecture diagrams, or add a rehype-mermaid plugin.

## Open Risks

- Pre-existing `@agh/extension-sdk` build error (SessionState → SessionStatus rename) causes full monorepo `turbo run build` to fail — unrelated to site work.

## Handoffs
