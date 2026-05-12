# Web Frontend

React 19 SPA with Vite 8, TanStack Router (file-based), TanStack Query v5, Tailwind CSS v4, shadcn/ui (base-nova), Zustand, Zod. Formatted with oxfmt, linted with oxlint.

## Design System — generated `DESIGN.md` + token source

**`packages/ui/src/tokens.css` is the canonical token source. `DESIGN.md` is the generated token specification plus stable visual rationale** for the web app, the `@agh/ui` kit, and every marketing/docs surface. Before writing or changing UI:

- Pull every color, font, radius, spacing step, and motion value from `packages/ui/src/tokens.css` and the generated `DESIGN.md` tables/frontmatter — never invent tokens.
- If `packages/ui/src/tokens.css` or site `@theme inline` tokens change, run `make codegen` and then `make codegen-check`. Do not hand-edit generated `DESIGN.md` frontmatter or `<!-- BEGIN:tokens:* -->` regions.
- Flat depth model only: no freehand `box-shadow`, no gradients on content, no glass except the sticky site header. Depth comes from the warm surface ramp, `--color-line*` hairlines, and exported `--shadow-*` overlay/focus tokens only.
- Type stack: **Inter** (UI + body), **JetBrains Mono** (all metadata, uppercase, tracking 0.06em+), **Playfair Display** (marketing `.site-home` only), **NuixyberNext** (the `agh` wordmark only).
- Signal palette — color is information, never decoration: accent `#E8572A` = action, `#5FBF85` = success, `#E0635A` = danger, `#D6A647` = warning, `#8E8EB5` = info. Status/kind chips use exported tint tokens; no solid semantic banners.
- Tokens live in `packages/ui/src/tokens.css`; never override with ad-hoc hex values in components.
- For design-system or UI redesign tasks, implementation runs through the `designer` agent (`.claude/agents/designer.md`) in **execution mode only** and MUST activate the mandatory design skills below.

## Copy System - `COPY.md` is the source of truth

**`COPY.md` (repo root) is the authoritative product-language specification** for web UI labels, headings, empty states, errors, onboarding/settings text, toasts, page metadata, and any runtime UI copy. Before writing or changing product-facing text:

- Read `COPY.md` and use backend nouns exactly; UI labels must match runtime/domain terminology.
- Do not imply a metric, control, state, or repair path exists unless the runtime exposes it.
- Follow `docs/_memory/glossary.md` for canonical terms, especially `capability`, `skill`, `bridge`, `channel`, `session`, and `task run`.
- Keep `DESIGN.md` as the visual authority and `COPY.md` as the verbal/product-language authority.

## Greenfield Alpha — Zero Legacy Tolerance

No production users exist. Never sacrifice code quality for backward compatibility. Never write migration, compat, or defensive code for old state — delete the old thing instead of working around it.

## Critical Rules

- **`make web-lint`, `make bun-typecheck`, and `make bun-test` MUST pass** before completing ANY web task. Zero warnings, zero errors.
- **Frontend typecheck/test validation MUST use Turborepo from the repo root.** Do not use `make web-typecheck`, `make web-test`, `cd web && bun run test`, `bun run --cwd web test`, or package-local equivalents as validation evidence; they bypass Turbo's cache/task graph.
- **Oxlint has zero tolerance** — any warning is a blocking failure
- **Follow shadcn kebab-case naming** for all files in `web/`
- **Native DOM wrappers** — if a component’s root is a single native element (`button`, `input`, `a`, …), its props MUST extend that element’s intrinsic type (`React.ComponentProps<"…">`), merge `className`, and spread `{...props}` onto the node (use `forwardRef` when refs apply). CVA + `VariantProps`: follow the `shadcn` skill. Canonical rule: `.agents/skills/react/SKILL.md` → _Extend native element props_.
- **Eyebrow markup is mandatory.** Every uppercase label MUST use either (a) the `<Eyebrow>` component from `@agh/ui` (children + `className` only — no `case` / `family` / `tone` / `size` / `weight` props) or (b) the single static utility class `eyebrow` (defined in `packages/ui/src/tokens.css`) on `<dt>`, `<label>`, table/sidebar primitives, and other structural elements. Color tone is applied through `className` (`text-(--muted)`, `text-(--subtle)`, `text-(--accent)`, signal palette). Do NOT inline `font-mono` + `uppercase` + arbitrary `text-[…]` + `tracking-[…]` tuples — that combination IS the eyebrow utility. The deleted `eyebrow-badge` / `eyebrow-micro` utility-class literals are forbidden (`compozy-design-system/no-inline-eyebrow` flags them). Canonical tokens: `--text-eyebrow` (11 px), `--tracking-eyebrow` (-0.005em); the contract is **Inter UC 11/600/-0.005em**. See `DESIGN.md` §3 and lesson `L-022`.
- **Never add JS dependencies by hand in `package.json`** — always use `bun add`
- **Check dependent package APIs** before writing integration code or tests
- **Test placement is mandatory before creating Vitest or Playwright coverage.** Name the invariant, owning layer, and canonical suite; update existing route/hook/component/story/e2e suites before creating a new file. Do not add CSS literal, snapshot, generated-output, or prose-string tests unless that artifact is the product contract and no stronger gate exists.
- **Local QA against an isolated daemon MUST read `AGH_WEB_API_PROXY_TARGET` from the active bootstrap manifest/env** — never hardcode `http://localhost:2123` when `agh-qa-bootstrap` or another isolated QA envelope is in use.

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required skills:

| Domain                        | Required Skills                                                 | Conditional Skills              |
| ----------------------------- | --------------------------------------------------------------- | ------------------------------- |
| React / Web UI                | `react` + `tailwindcss` + `vercel-react-best-practices`         | `shadcn`                        |
| Routing                       | `tanstack-router-best-practices`                                | `tanstack`                      |
| Data fetching                 | `tanstack-query-best-practices` + `app-renderer-systems`        |                                 |
| State management              | `zustand`                                                       |                                 |
| Schema / Validation           | `zod`                                                           | `typescript-advanced`           |
| Web testing                   | `consolidate-test-suites` + `vitest` + `react` + `testing-boss` |                                 |
| TypeScript (types)            | `typescript-advanced`                                           | `context7`                      |
| UI / UX Design (any surface)  | `agh-design` + `ui-craft`                                       | `shadcn` + `agh-ui-screenshot`  |
| UI verification / visual diff | `agh-ui-screenshot`                                             |                                 |
| UI microcopy / product labels | `copywriting` + `documentation-writer` + `ui-craft`             |                                 |
| Storybook / component stories | `storybook-stories`                                             | `shadcn`                        |
| Animation / motion            | `motion-react` + `ui-craft`                                     | `motion`                        |
| Component patterns            | `vercel-composition-patterns` + `vercel-react-best-practices`   | `ui-craft`                      |
| AI / Streaming                | `ai-sdk`                                                        | `tanstack-query-best-practices` |
| Bug fix                       | `systematic-debugging` + `no-workarounds`                       | `testing-boss`                  |
| External docs lookup          | `context7`                                                      | `exa-web-search-free`           |
| Task completion               | `cy-final-verify`                                               |                                 |

**Design-system / redesign passes**: you MUST run the `designer` agent in execution mode (not plan mode) AND activate `agh-design` + `ui-craft` before touching any component. `agh-design` lives at `.agents/skills/agh/agh-design/SKILL.md`. `packages/ui/src/tokens.css` and generated `DESIGN.md` tokens win over anything informal already in the codebase. `ui-craft` is reference-routed — match the task to a row in `.agents/skills/ui-craft/SKILL.md` and read the listed files in full (e.g. `accessibility-floor.md`, `component-patterns.md`, `ai-slop-patterns.md`, `anti-defaults.md`, `microcopy-quality.md`, `motion-patterns.md`, `dark-mode.md`).

**Visual verification with `agh-ui-screenshot` is mandatory for every UI change in this workspace.** Tests verify code, not pixels.

- Capture the matching Storybook story (`components-button--*`, `routes-app-stories-*`) on port 6006 and diff against a trusted prior baseline.
- Surface-wide passes (primitive swap, token retune): capture before + after.
- Use `list-stories.mjs` to resolve valid story ids — misaligned ids land on the "Couldn't find story" fallback (sub-20 KB PNG).
- Cite the capture file(s) when reporting done. Claiming success without screenshots is non-compliant.

**Web test placement**: `consolidate-test-suites` runs before any new Vitest or Playwright file. A web task needs a test decision: invariant, owning layer, canonical suite, and verification command. "No new automated test" is valid when visual QA, lint, typecheck, codegen, Storybook capture, or an existing suite already owns the invariant.

## Build Commands

Turbo-backed validation commands run from the repo root:

```bash
make bun-typecheck                         # full Bun workspace typecheck through turbo
make bun-test                              # full Bun workspace test suite through turbo
bunx turbo run typecheck --filter=./web    # focused agh-web typecheck
bunx turbo run test --filter=./web         # focused agh-web tests
```

Local web shortcuts are for development/build/lint only:

```bash
make web-dev     # Start Vite dev server on :3000 (proxies /api to :2123 by default; for isolated QA export AGH_WEB_API_PROXY_TARGET from bootstrap.env first)
make web-build   # Production build (vite build + tsc --noEmit)
make web-lint    # Format (oxfmt) + lint (oxlint)
make web-fmt     # Format with oxfmt
```

## Structure

```
web/src/
├── routes/              # TanStack file-based routes (auto code-splitting)
├── systems/             # Domain feature modules (app-renderer-systems pattern)
│   └── <domain>/
│       ├── index.ts          # Public API barrel (explicit named exports)
│       ├── types.ts          # Domain types
│       ├── adapters/         # API service layer (<domain>-api.ts + error class)
│       ├── lib/              # query-keys.ts, query-options.ts, schemas, constants
│       ├── hooks/            # Query hooks, mutation hooks, view-model hooks
│       ├── contexts/         # React contexts + providers (optional)
│       ├── stores/           # Zustand/XState stores (optional)
│       ├── components/       # Domain-specific UI components
│       └── guards/           # Route guards / access checks (optional)
├── components/          # Shared components (ui/ for shadcn)
├── lib/                 # Shared utilities (utils.ts)
├── integrations/        # Third-party integrations (tanstack-query/)
├── styles.css           # Tailwind v4 theme + shadcn
└── routeTree.gen.ts     # Auto-generated route tree (never edit)
```

## Systems Architecture (app-renderer-systems)

Domain features are organized as **systems** under `src/systems/<domain>/`. Each system is self-contained and owns its API calls, query layer, hooks, components, and public API. See `app-renderer-systems` skill for full patterns.

**Dependency flow**: `adapters → lib → hooks → components` (unidirectional, never reversed).

**Cross-system imports**: Only through the public barrel (`@/systems/<domain>`). Never reach into another system's internals.

**Key conventions**:

- Co-locate `queryKey` + `queryFn` via `queryOptions` factories in `lib/query-options.ts`
- Hierarchical query keys in `lib/query-keys.ts` for granular invalidation
- Typed error classes in adapters — never throw raw errors
- Pass `AbortSignal` from query context through to every API call
- Always invalidate after mutations (`onSettled`)
- Optimistic updates require rollback via `onMutate`/`onError` snapshots

## Frontend Architecture Rules

- **UI components MUST be pure and presentational**; orchestration lives in pages/routes
- **State hierarchy**: local state (`useState`/`useReducer`) > Zustand > TanStack Query > URL state
- **Server state via TanStack Query only** — never duplicate into client state
- **Data fetching at route/page level** — components receive data via props
- **Components MUST NOT import from stores/ or adapters directly** — pass via props or route context
- **File naming**: kebab-case for all files — components (`kebab-case.tsx`), hooks (`use-kebab-case.ts`), utilities (`kebab-case.ts`), API services (`<domain>-api.ts`)
- **Prefer named exports** for components and utils; no `export * from`
- **Functional components only** — no class components, no `React.FC`
- **useEffect is an escape hatch** — only for external system sync; never for derived state or event responses
- **Handle all states** — loading, error, and empty (never assume `data` exists)
- **Composition over booleans** — compound components instead of boolean prop proliferation
- **Path alias**: `@/*` maps to `./src/*`

## Tooling

- **Package manager**: Bun (workspaces)
- **Monorepo**: Turborepo
- **Linting**: Oxlint (zero warnings)
- **Formatting**: Oxfmt (printWidth: 100, double quotes, semicolons)
- **Testing**: Vitest + Testing Library (jsdom)
- **Commits**: Conventional Commits + commitlint + husky + lint-staged
- **Icons**: lucide-react
- **Notifications**: sonner
- **Vite proxy**: `/api` → `localhost:2123` by default; for isolated daemon QA, read `AGH_WEB_API_PROXY_TARGET` from `<qa-output-path>/qa/bootstrap-manifest.json` or `bootstrap.env` instead of hardcoding the port
