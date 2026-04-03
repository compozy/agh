# Web Frontend

React 19 SPA with Vite 8, TanStack Router (file-based), TanStack Query v5, Tailwind CSS v4, shadcn/ui (base-nova), Zustand, Zod. Formatted with oxfmt, linted with oxlint.

## Critical Rules

- **`make web-lint` and `make web-typecheck` MUST pass** before completing ANY web task. Zero warnings, zero errors.
- **Oxlint has zero tolerance** — any warning is a blocking failure
- **Follow shadcn kebab-case naming** for all files in `web/`
- **Never add JS dependencies by hand in `package.json`** — always use `bun add`
- **Check dependent package APIs** before writing integration code or tests

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required skills:

| Domain              | Required Skills                                               | Conditional Skills               |
| ------------------- | ------------------------------------------------------------- | -------------------------------- |
| React / Web UI      | `react` + `tailwindcss` + `vercel-react-best-practices`       | `shadcn`                         |
| Routing             | `tanstack-router-best-practices`                              | `tanstack`                       |
| Data fetching       | `tanstack-query-best-practices` + `app-renderer-systems`      |                                  |
| State management    | `zustand`                                                     |                                  |
| Schema / Validation | `zod`                                                         | `typescript-advanced`            |
| Web testing         | `vitest` + `react` + `testing-anti-patterns`                  |                                  |
| TypeScript (types)  | `typescript-advanced`                                         | `context7`                       |
| UI/UX Design        | `frontend-design`                                             | `interface-design` + `shadcn-ui` |
| Component patterns  | `vercel-composition-patterns` + `vercel-react-best-practices` |                                  |
| AI / Streaming      | `ai-sdk`                                                      | `tanstack-query-best-practices`  |
| Bug fix             | `systematic-debugging` + `no-workarounds`                     | `testing-anti-patterns`          |

## Build Commands

```bash
make web-dev             # Start Vite dev server on :3000 (proxies /api to :2123)
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
- **Vite proxy**: `/api` → `localhost:2123` (AGH daemon)
