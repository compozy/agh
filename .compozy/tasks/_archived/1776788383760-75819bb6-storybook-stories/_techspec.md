# TechSpec — Storybook Stories Rollout

## Executive Summary

Author Storybook 10 stories for every production React component under `packages/ui/src/components` (~12), `web/src/components/ui` (~27), and `web/src/systems/<name>/components` (~40) in a single PR. Deliver two Storybook instances — one scoped to the `@agh/ui` primitives workspace, one scoped to the web app — so that each layer loads only the decorators it needs. The web instance adds Mock Service Worker + global TanStack Query and router decorators; the `@agh/ui` instance stays render-only.

The primary technical trade-off is additional infrastructure (a second Storybook process + MSW bootstrap) in exchange for clean dependency direction (`packages/ui` never depends on `web`), faster base-component startup, and reusable MSW fixtures that double as test infrastructure for Vitest and future Playwright runs.

## System Architecture

### Component Overview

```
packages/ui/                           web/
├── .storybook/           (NEW)        ├── .storybook/          (extended)
│   ├── main.ts                        │   ├── main.ts          (unchanged glob)
│   └── preview.ts        → tokens     │   └── preview.ts       → MSW + QC + Router
└── src/components/                    ├── public/
    └── stories/          (NEW)        │   └── mockServiceWorker.js  (NEW, generated)
        *.stories.tsx                  └── src/
                                           ├── components/
                                           │   ├── design-system/stories/   (existing)
                                           │   └── ui/stories/              (NEW)
                                           └── systems/<name>/
                                               ├── components/stories/      (NEW)
                                               └── mocks/                   (NEW)
                                                   ├── handlers.ts
                                                   ├── fixtures.ts
                                                   └── index.ts
```

Two Storybook processes (ports 6006 for web, 6007 for `@agh/ui`) coexist. Each reads `@agh/ui/tokens.css` to share the design-token surface. The web instance additionally wires MSW, a `QueryClientProvider`, and a memory-history router stub so that system components render with realistic data.

Per-system `mocks/` folders own their MSW handlers and typed fixtures. Story authors override handlers locally via `parameters.msw.handlers` to expose loading, success, error, and empty states.

## Implementation Design

### Core Interfaces

#### Story module contract (all layers)

```ts
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button } from "@agh/ui";

const meta: Meta<typeof Button> = {
  title: "ui/Button",
  component: Button,
  parameters: {
    layout: "centered",
    docs: { description: { component: "Primary button with variants and sizes." } },
  },
  tags: ["autodocs"], // primitives only (packages/ui) — see ADR-003
};
export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = { args: { children: "Action", variant: "default" } };
```

#### Per-system MSW contract

```ts
// web/src/systems/session/mocks/handlers.ts
import { http, HttpResponse, type HttpHandler } from "msw";
import { sessionFixture, sessionListFixture } from "./fixtures";

export const handlers: HttpHandler[] = [
  http.get("/api/sessions", () => HttpResponse.json(sessionListFixture)),
  http.get("/api/sessions/:id", ({ params }) =>
    HttpResponse.json({ ...sessionFixture, id: params.id })
  ),
];
```

#### Web Storybook global decorators

```ts
// web/.storybook/preview.ts (essential fragments)
import { initialize, mswLoader } from "msw-storybook-addon";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { handlers as sessionHandlers } from "@/systems/session/mocks";
// …other systems

initialize({ onUnhandledRequest: "bypass" });

const qc = () => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });

export default {
  loaders: [mswLoader],
  parameters: { msw: { handlers: [...sessionHandlers, /* … */] } },
  decorators: [
    (Story) => <QueryClientProvider client={qc()}><Story /></QueryClientProvider>,
    // …router + theme decorators
  ],
};
```

### Data Models

- **Fixtures** — plain TypeScript objects typed against adapter response types. One file per system: `web/src/systems/<name>/mocks/fixtures.ts`. Named exports only, no default exports, no computed values at import time.
- **Handlers** — MSW `HttpHandler[]` arrays per system. Compose into a default set at `preview.ts`; per-story overrides replace the array.
- **Story meta** — `Meta<typeof Component>` with explicit annotation (skill rule). Titles follow path: `ui/<Name>` for primitives, `components/ui/<Name>` for composed shadcn layer, `systems/<name>/<Name>` for domain.

### API Endpoints

No new runtime endpoints. Storybook consumes the existing daemon surface through MSW handlers, which cover the subset of `/api/**` routes each system queries (sessions, agents, bridges, automations, skills, workspaces, knowledge, network channels/peers, daemon health).

## Integration Points

No external services. MSW intercepts at the service-worker layer inside the browser serving Storybook; no network traffic escapes the iframe.

## Impact Analysis

| Component                                        | Impact Type | Description and Risk                                                             | Required Action                                                                 |
| ------------------------------------------------ | ----------- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `packages/ui/.storybook/*`                       | new         | Second Storybook instance for primitives. Low risk; isolated from web app.       | Create `main.ts`, `preview.ts`; add deps and scripts.                           |
| `packages/ui/package.json`                       | modified    | Add Storybook 10 devDependencies + `storybook`/`build-storybook` scripts.        | `bun add -D` in that workspace.                                                 |
| `packages/ui/src/components/stories/*.stories.tsx` | new       | ~12 story files, 2-5 stories each.                                               | Author per skill conventions.                                                   |
| `web/.storybook/preview.ts`                      | modified    | Add MSW loader, QueryClient + router decorators, default handler composition.    | Extend without breaking existing `design-system/*` stories.                     |
| `web/package.json`                               | modified    | Add `msw@^2` and `msw-storybook-addon@^2`.                                       | `bun add -D` in `web/`.                                                         |
| `web/public/mockServiceWorker.js`                | new         | Generated by `msw init web/public --save`.                                       | Run generator once, commit the worker file.                                     |
| `web/src/components/ui/stories/*.stories.tsx`    | new         | ~27 stories for shadcn-layer composites.                                         | Author per skill conventions.                                                   |
| `web/src/systems/<name>/components/stories/*`    | new         | ~40 stories across 9 systems.                                                    | Author per skill conventions with MSW overrides for states.                     |
| `web/src/systems/<name>/mocks/`                  | new         | Handlers + fixtures per system.                                                  | 9 new folders; typed against adapter response types.                            |
| `web/src/components/design-system/stories/*`     | unchanged   | Existing stories already follow the chosen convention.                           | None.                                                                           |
| `.claude/skills/storybook-stories/SKILL.md`      | modified    | Replace `@compozy/ui` references with `@agh/ui`; clarify autodocs policy.        | Edit during rollout.                                                            |

## Testing Approach

### Unit Tests

- No new Vitest tests are introduced for story files themselves.
- **Fixture contract**: export `*.fixture.test.ts` only when a fixture is consumed by Vitest; otherwise the TypeScript compiler is sufficient proof of shape.
- **Storybook build**: `bun run --cwd web build-storybook` and `bun run --cwd packages/ui build-storybook` must succeed as part of CI. A failing build is a failing test.
- **a11y**: `@storybook/addon-a11y` already installed in the web instance; retain and add to `packages/ui/.storybook`. Stories with critical violations block merge; only block on critical (not moderate/minor) to keep signal high.

### Integration Tests

- No dedicated integration suite. The MSW handlers ship a parallel set of assertions via type-checking against adapter types — a contract mismatch surfaces as a `tsgo --noEmit` failure.
- Sanity pass: run `bun run --cwd web storybook` and `bun run --cwd packages/ui storybook` locally, open each story, and confirm render + no console errors before PR review.

## Development Sequencing

### Build Order

1. **Bootstrap `@agh/ui` Storybook** — no dependencies. Create `packages/ui/.storybook/{main.ts,preview.ts}`, add Storybook 10 devDeps, wire `storybook`/`build-storybook` scripts. Confirm `bun run --cwd packages/ui storybook` boots with an empty index.
2. **Bootstrap MSW in web Storybook** — depends on step 1 only for convention parity. Install `msw` + `msw-storybook-addon`, run `bunx msw init web/public --save`, extend `web/.storybook/preview.ts` with the MSW loader, `QueryClientProvider`, and memory-history router decorator. Validate existing `design-system/*` stories still render.
3. **Author `packages/ui` stories** — depends on step 1. ~12 story files under `packages/ui/src/components/stories/`. Use `tags: ["autodocs"]`. Each: Default + 1-4 variants. Keep renders pure.
4. **Author `web/src/components/ui` stories** — depends on step 2. ~27 story files under `web/src/components/ui/stories/`. No autodocs by default. Compound components use `render`.
5. **Add per-system `mocks/` folders** — depends on step 2. Nine folders (`agent`, `automation`, `bridges`, `daemon`, `knowledge`, `network`, `session`, `skill`, `workspace`) each with `handlers.ts`, `fixtures.ts`, `index.ts`. Types from existing adapter modules.
6. **Compose default handler set** — depends on step 5. Import all system handler arrays into `web/.storybook/preview.ts` and spread into `parameters.msw.handlers`.
7. **Author system stories** — depends on steps 4, 5, 6. ~40 story files under `web/src/systems/<name>/components/stories/`. Each: Default + loading + error + (optional) empty. Override handlers per story.
8. **Update skill doc** — depends on step 7. Rewrite `.claude/skills/storybook-stories/SKILL.md` to reference `@agh/ui`, document autodocs policy (primitives only), and point to ADRs.
9. **Verify & land** — depends on all prior steps. Run `make web-lint`, `make web-typecheck`, `bun run --cwd web build-storybook`, `bun run --cwd packages/ui build-storybook`. Open a single PR.

### Technical Dependencies

- Storybook 10.3.5 (already pinned in `web/package.json`).
- `msw@^2`, `msw-storybook-addon@^2` — new devDeps.
- Workspace hoisting (`bun workspaces`) must resolve `@agh/ui` from `packages/ui` — already working.
- `@agh/ui/tokens.css` must be importable from both `.storybook/preview.ts` files.

## Monitoring and Observability

Not applicable — Storybook is a dev/CI artifact. Operational health is covered by:
- CI job success on `bun run --cwd web build-storybook` and `bun run --cwd packages/ui build-storybook`.
- `@storybook/addon-a11y` panel during local review.

## Technical Considerations

### Key Decisions

- **Decision**: Two Storybook instances instead of one.
  - **Rationale**: Keeps dependency direction intact (`packages/ui` never imports from `web`) and avoids loading MSW/QueryClient for pure primitive browsing.
  - **Trade-offs**: Two dev servers and two build commands.
  - **Alternatives rejected**: cross-workspace glob (violates dep direction), stories-in-web-only (breaks co-location).

- **Decision**: MSW + global `QueryClientProvider` + memory-history router decorators.
  - **Rationale**: Most system components own their queries; inserting a realistic data layer avoids invasive refactors.
  - **Trade-offs**: Adds two devDeps and a service-worker file.
  - **Alternatives rejected**: prop-drilling refactor (architectural damage), `queryClient.setQueryData` prefill (no loading/error coverage).

- **Decision**: `stories/` subfolder + autodocs opt-in.
  - **Rationale**: Matches existing `design-system/stories/` layout and keeps docs noise low outside the primitive layer.
  - **Trade-offs**: Extra directory per folder; autodocs policy enforced by review, not lint.
  - **Alternatives rejected**: co-location (folder bloat in systems), mixed convention (inconsistency).

- **Decision**: Mocks owned per system under `web/src/systems/<name>/mocks/`.
  - **Rationale**: Matches app-renderer-systems self-containment; fixtures live next to the adapter types.
  - **Trade-offs**: Nine new folders.
  - **Alternatives rejected**: central `web/src/mocks/` (violates system ownership), inline per-story (duplication).

### Known Risks

- **Handler drift vs. real `/api` contract** — Mitigation: fixtures are typed against adapter response types; `tsgo --noEmit` fails on shape divergence.
- **MSW worker not registered correctly** — Symptoms: blank stories or real network calls in Storybook. Mitigation: run `msw init` once; assert in preview that the worker is `active` in development.
- **Divergence of the two Storybook configs** — Mitigation: document shared fragments (tokens, a11y addon, themes addon) in ADR-001 and keep both configs minimal.
- **Skill doc drift (`@compozy/ui` vs `@agh/ui`)** — Mitigation: update as part of this rollout (step 8).
- **Single-PR size** — ~80 story files + infra. Mitigation: PR structured by commit per step (1–9) so review can proceed incrementally even if the merge is single-shot.

## Architecture Decision Records

- [ADR-001: Dual Storybook Topology](adrs/adr-001.md) — Run a dedicated `packages/ui` Storybook next to the existing `web` Storybook to preserve dependency direction.
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md) — Enable `msw-storybook-addon` with global `QueryClientProvider` and router decorators in the web instance.
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — Put stories under a `stories/` subfolder; apply `tags: ["autodocs"]` only to `@agh/ui` primitives.
- [ADR-004: Per-System Mocks Directory](adrs/adr-004.md) — Own MSW handlers and fixtures inside `web/src/systems/<name>/mocks/`, not a central mocks tree.
