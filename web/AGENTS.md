# Web Frontend

React 19 SPA with Vite 8, TanStack Router (file-based), TanStack Query v5, Tailwind CSS v4, shadcn/ui (base-nova), Zustand, Zod. Formatted with oxfmt, linted with oxlint.

## Design System — `DESIGN.md` is the source of truth

**`DESIGN.md` (repo root) is the authoritative design-system specification** for the web app, the `@agh/ui` kit, and every marketing/docs surface. Before writing or changing UI:

- Pull every color, font, radius, spacing step, and motion value **from `DESIGN.md`** — never invent tokens.
- Flat depth model only: no `box-shadow`, no gradients on content, no glass except the sticky site header. Depth comes from 4 background steps + 1px `#3C3A39` dividers.
- Type stack: **Inter** (UI + body), **JetBrains Mono** (all metadata, uppercase, tracking 0.06em+), **Playfair Display** (marketing `.site-home` only), **NuixyberNext** (the `agh` wordmark only).
- Signal palette — color is information, never decoration: accent `#E8572A` = action, `#30D158` = success, `#FF453A` = danger, `#FFD60A` = warning, `#BF5AF2` = info. Status/kind chips use the 15%-tint formula; no solid semantic banners.
- Tokens live in `packages/ui/src/tokens.css`; never override with ad-hoc hex values in components.
- For tasks under `.compozy/tasks/redesign/`, implementation runs through the `designer` agent (`.claude/agents/designer.md`) in **execution mode only** and MUST activate the mandatory design skills below.

## Greenfield Alpha — Zero Legacy Tolerance

No production users exist. Never sacrifice code quality for backward compatibility. Never write migration, compat, or defensive code for old state — delete the old thing instead of working around it.

## Critical Rules

- **`make web-lint` and `make web-typecheck` MUST pass** before completing ANY web task. Zero warnings, zero errors.
- **Oxlint has zero tolerance** — any warning is a blocking failure
- **Follow shadcn kebab-case naming** for all files in `web/`
- **Native DOM wrappers** — if a component’s root is a single native element (`button`, `input`, `a`, …), its props MUST extend that element’s intrinsic type (`React.ComponentProps<"…">`), merge `className`, and spread `{...props}` onto the node (use `forwardRef` when refs apply). CVA + `VariantProps`: follow the `shadcn` skill. Canonical rule: `.agents/skills/react/SKILL.md` → _Extend native element props_.
- **Never add JS dependencies by hand in `package.json`** — always use `bun add`
- **Check dependent package APIs** before writing integration code or tests
- **Local QA against an isolated daemon MUST read `AGH_WEB_API_PROXY_TARGET` from the active bootstrap manifest/env** — never hardcode `http://localhost:2123` when `agh-qa-bootstrap` or another isolated QA envelope is in use.

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required skills:

| Domain                        | Required Skills                                                  | Conditional Skills                              |
| ----------------------------- | ---------------------------------------------------------------- | ----------------------------------------------- |
| React / Web UI                | `react` + `tailwindcss` + `vercel-react-best-practices`          | `shadcn`                                        |
| Routing                       | `tanstack-router-best-practices`                                 | `tanstack`                                      |
| Data fetching                 | `tanstack-query-best-practices` + `app-renderer-systems`         |                                                 |
| State management              | `zustand`                                                        |                                                 |
| Schema / Validation           | `zod`                                                            | `typescript-advanced`                           |
| Web testing                   | `vitest` + `react` + `testing-anti-patterns`                     |                                                 |
| TypeScript (types)            | `typescript-advanced`                                            | `context7`                                      |
| UI / UX Design (generic)      | `frontend-design` + `design-taste-frontend`                      | `interface-design` + `shadcn` + `minimalist-ui` |
| **AGH UI / Redesign tasks**   | `agh-design` + `design-taste-frontend` + `minimalist-ui`         | `frontend-design` + `interface-design`          |
| Storybook / component stories | `storybook-stories`                                              | `shadcn`                                        |
| Animation / motion            | `motion-react`                                                   | `motion`                                        |
| Component patterns            | `vercel-composition-patterns` + `vercel-react-best-practices`    |                                                 |
| AI / Streaming                | `ai-sdk`                                                         | `tanstack-query-best-practices`                 |
| Bug fix                       | `systematic-debugging` + `no-workarounds`                        | `testing-anti-patterns`                         |
| Design polish passes          | `impeccable:polish` + `impeccable:layout` + `impeccable:typeset` | `impeccable:delight` + `impeccable:critique`    |
| External docs lookup          | `context7` + `find-docs`                                         | `exa-web-search-free`                           |
| Task completion               | `cy-final-verify`                                                |                                                 |

**Redesign tasks (`.compozy/tasks/redesign/*`)**: you MUST run the `designer` agent in execution mode (not plan mode) AND activate `agh-design` + `design-taste-frontend` + `minimalist-ui` before touching any component. `DESIGN.md` tokens win over anything informal already in the codebase.

## Build Commands

```bash
make web-dev             # Start Vite dev server on :3000 (proxies /api to :2123 by default; for isolated QA export AGH_WEB_API_PROXY_TARGET from bootstrap.env first)
make web-build           # Production build (vite build + tsc --noEmit)
make web-lint            # Format (oxfmt) + lint (oxlint)
make web-fmt             # Format with oxfmt
make web-typecheck       # Type check with tsc
make web-test            # Run tests (Vitest)
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
