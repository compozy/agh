# Web Frontend

React 19 SPA with Vite 8, TanStack Router (file-based) + Query v5, Tailwind v4, shadcn/ui (base-nova), Zustand, Zod. oxfmt + oxlint. (Root `CLAUDE.md` rules apply — this file adds web-specific ones.)

## Critical Rules

- **`make web-lint`, `make bun-typecheck`, and `make bun-test` MUST pass** before completing any web task. Zero warnings/errors; oxlint is zero-tolerance.
- **Frontend typecheck/test validation MUST use Turborepo from the repo root.** Never use `make web-typecheck`, `make web-test`, `cd web && bun run test`, `bun run --cwd web test`, or package-local equivalents as evidence — they bypass Turbo's cache/task graph.
- **Files are kebab-case** (shadcn convention): components `kebab-case.tsx`, hooks `use-kebab-case.ts`, utils `kebab-case.ts`, API services `<domain>-api.ts`.
- **Native DOM wrappers**: if a component's root is a single native element, its props MUST extend that element's intrinsic type (`React.ComponentProps<"…">`), merge `className`, and spread `{...props}` (use `forwardRef` when refs apply). CVA + `VariantProps` per the `shadcn` skill. Canonical: `.agents/skills/react/SKILL.md` → _Extend native element props_.
- **Eyebrow markup is mandatory** for every uppercase label: use `<Eyebrow>` from `@agh/ui` (children + `className` only) **or** the `eyebrow` utility class (from `packages/ui/src/tokens.css`) on structural elements; tone via `className` (`text-(--muted)`, `text-(--accent)`, signal palette). Inlining `font-mono` + `uppercase` + `text-[…]` + `tracking-[…]` tuples IS the utility and is **forbidden** (`compozy-design-system/no-inline-eyebrow`), as are the removed `eyebrow-badge`/`eyebrow-micro` literals. Contract: **Inter UC 11/600/-0.005em** (`--text-eyebrow`, `--tracking-eyebrow`). Full rule: `DESIGN.md` §3 + lesson `L-022`.
- **Test placement before any Vitest/Playwright file.** Name the invariant, owning layer, and canonical suite; update existing route/hook/component/story/e2e suites first. No CSS-literal/snapshot/generated/prose tests unless that artifact is the product contract and no stronger gate exists.
- **Storybook setup is validation infrastructure, not product behavior.** Don't test `.storybook/main`, decorator arrays, story globs, or bootstrap unless the task names the visual-QA contract it protects. Prefer Storybook build/capture, `list-stories`, and rendered-story smoke. Static exceptions need a `KEEP:` note.
- **Isolated-daemon QA reads `AGH_WEB_API_PROXY_TARGET`** from the active bootstrap manifest/env — never hardcode `http://localhost:2123`.

## Design & Copy (web surface)

Tokens come from `packages/ui/src/tokens.css` + generated `DESIGN.md` — never invent; never override with ad-hoc hex. Web-specific grammar:

- **Type stack**: Inter (UI + body), JetBrains Mono (metadata, uppercase, tracking 0.06em+), Playfair Display (marketing `.site-home` only), NuixyberNext (the `agh` wordmark only).
- **Flat depth only**: no freehand `box-shadow`, no content gradients, no glass except the sticky site header. Depth = warm surface ramp + `--color-line*` hairlines + exported `--shadow-*` tokens.
- **Signal palette is information**: accent `#E8572A` action · `#5FBF85` success · `#E0635A` danger · `#D6A647` warning · `#8E8EB5` info. Status chips use tint tokens, never solid semantic banners.
- **Copy**: read `COPY.md`; use backend nouns exactly (labels match runtime/domain terms); never imply a metric/control/state/repair path the runtime doesn't expose. Terms per `docs/_memory/glossary.md`.

## Skill Dispatch

Activate skills **before** writing code. Match task domain → activate all required:

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

- **Design-system / redesign passes**: run the `designer` agent in execution mode (not plan) and activate `agh-design` + `ui-craft` before touching any component. `ui-craft` is reference-routed — read the matched rows in full. `tokens.css` + generated `DESIGN.md` win over anything informal in the codebase.
- **Visual verification is mandatory for every UI change** (`agh-ui-screenshot`; tests verify code, not pixels): capture the matching Storybook story on port 6006 (resolve ids via `list-stories.mjs` — bad ids hit the sub-20 KB "Couldn't find story" fallback), diff against a trusted baseline (before + after for surface-wide passes), and cite the capture when reporting done.

## Build Commands

Turbo-backed validation runs from the repo root:

```bash
make bun-typecheck / bun-test              # full Bun workspace typecheck / test through turbo
bunx turbo run typecheck --filter=./web    # focused agh-web typecheck
bunx turbo run test --filter=./web         # focused agh-web tests
```

Web-local dev/build only (not validation evidence):

```bash
make web-dev     # Vite dev on :3000 (proxies /api to :2123; for isolated QA export AGH_WEB_API_PROXY_TARGET first)
make web-build   # Production build (vite build + tsc --noEmit)
make web-lint    # Repo-root frontend lint gate (web + packages/ui + packages/site)
make web-fmt     # oxfmt
```

## Structure

```
web/src/
├── routes/              # TanStack file-based routes (auto code-splitting)
├── systems/<domain>/    # Self-contained domain modules (app-renderer-systems)
│   ├── index.ts             # Public API barrel (explicit named exports)
│   ├── types.ts
│   ├── adapters/            # API service layer (<domain>-api.ts + error class)
│   ├── lib/                 # query-keys.ts, query-options.ts, schemas, constants
│   ├── hooks/               # query / mutation / view-model hooks
│   ├── contexts/ stores/    # React contexts / Zustand|XState (optional)
│   ├── components/          # domain UI
│   └── guards/              # route guards (optional)
├── components/          # Shared components (ui/ for shadcn)
├── lib/ integrations/   # Shared utils / third-party (tanstack-query/)
├── styles.css           # Tailwind v4 theme + shadcn
└── routeTree.gen.ts     # Auto-generated (never edit)
```

## Systems Architecture (app-renderer-systems)

Domain features are self-contained **systems** under `src/systems/<domain>/`, each owning its API calls, query layer, hooks, components, and public API. See the `app-renderer-systems` skill for full patterns.

- **Dependency flow** `adapters → lib → hooks → components` (unidirectional, never reversed).
- **Cross-system imports** only through the public barrel (`@/systems/<domain>`) — never reach into internals.
- Co-locate `queryKey` + `queryFn` via `queryOptions` factories in `lib/query-options.ts`; hierarchical keys in `lib/query-keys.ts` for granular invalidation.
- Typed error classes in adapters (never throw raw); pass `AbortSignal` through to every API call.
- Invalidate after mutations (`onSettled`); optimistic updates need rollback via `onMutate`/`onError` snapshots.

## Frontend Architecture Rules

- **UI components are pure and presentational**; orchestration lives in pages/routes.
- **State hierarchy**: local (`useState`/`useReducer`) > Zustand > TanStack Query > URL. Server state via TanStack Query only — never duplicate into client state.
- **Data fetching at route/page level**; components receive data via props and MUST NOT import from `stores/` or `adapters/` directly (pass via props or route context).
- **Named exports** only (no `export * from`). **Functional components only** — no class components, no `React.FC`.
- **`useEffect` is an escape hatch** — external-system sync only; never derived state or event responses.
- **Handle all states** — loading, error, empty (never assume `data` exists). **Composition over booleans** (compound components).
- Path alias `@/*` → `./src/*`.

## Tooling

Bun (workspaces) + Turborepo · Oxlint (zero warnings) · Oxfmt (printWidth 100, double quotes, semicolons) · Vitest + Testing Library (jsdom) · Conventional Commits + commitlint + husky + lint-staged · lucide-react icons · sonner toasts. Vite proxy `/api` → `:2123` by default; isolated QA reads `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest/`bootstrap.env`.
