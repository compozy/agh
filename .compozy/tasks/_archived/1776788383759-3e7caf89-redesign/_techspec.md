# TechSpec: Web Redesign — align packages/ui + web/ with the new design system

## Executive Summary

This TechSpec plans a full visual and structural redesign of the AGH operator UI. It consolidates three overlapping primitive layers (`@agh/ui`, `web/src/components/ui/`, `web/src/components/design-system/`) into a single source (`@agh/ui`), adopts the visual language captured in `DESIGN.md` and the redesign mock at `docs/design/web-inspiration/`, and replaces every screen in `web/src/systems/**` with a rewrite that consumes the new primitives. Domain components (query hooks, stores, SSE wiring) keep their current structure; only the visual layer is rewritten.

The migration is greenfield-in-place — old primitives, styles, and variants are deleted the same PR the new ones land, with no compat shims or feature flags (ADR-002). Rollout is phased in six steps: tokens + primitives, app shell, Tasks, Session, remaining domains, Settings (ADR-004). Motion is handled by the `motion` package for unmount/route transitions; CSS owns everything simpler (ADR-003). Visual regression is enforced via Playwright `toHaveScreenshot` over every Storybook story and every route (ADR-005). **Primary trade-off:** large per-phase PRs in exchange for a single coherent design system, at the cost of higher review density and a 30KB motion-lib bundle addition.

## System Architecture

### Component Overview

The redesign spans three runtime units and one visual reference:

- **`packages/ui/` (`@agh/ui`)** — single source for generic primitives + tokens. Grows from 12 to ~35 exports. Owns its own Storybook. Exports one `<UIProvider>` for motion config.
- **`web/`** — the operator SPA. Rewritten screen-by-screen (Phases 3–6) to consume `@agh/ui`. `web/src/components/ui/` and `web/src/components/design-system/` folders are deleted. `web/src/systems/<domain>/` retains domain logic (adapters, query hooks, stores, types, mocks) unchanged in structure; only its `components/` subdirs are rewritten.
- **`docs/design/web-inspiration/`** — frozen visual reference. Never imported at runtime. Source of truth for the shape of each screen; primitives derive from it but are coded fresh in `@agh/ui`.
- **Motion config** — one `<MotionConfig reducedMotion="user">` provided by `@agh/ui`, wrapping the entire app in `web/src/main.tsx`.

### Data flow

No changes to data flow. TanStack Query hooks, Zustand stores, MSW mocks, OpenAPI types (`web/src/generated/`), and SSE wiring all stay as-is. The redesign is purely presentational; every rewritten component receives the same props shape it does today (where props change, that change is limited to presentational surface — size, variant, slot name).

### External system interactions

None. No API, SSE, IPC, or storage layer is touched.

### Dependency graph (after migration)

```
@agh/ui (tokens + primitives + motion config)
   ↑
web/src/components (app-sidebar shell + cross-system compositions)
   ↑
web/src/systems/<domain>/components (domain compositions)
   ↑
web/src/routes/** (TanStack Router routes)
```

Import rules enforced via CI grep + tsconfig paths:

- `@agh/ui` must not import from `web/src/**`.
- `web/src/systems/<A>/**` must not import from `web/src/systems/<B>/**` (cross-domain isolation, already the rule).
- `web/src/components/**` imports only from `@agh/ui` and its own internals.

## Implementation Design

### Core Interfaces

#### `@agh/ui` root provider

The single entry wrap for theming + motion config. Consumed once in `web/src/main.tsx`.

```tsx
// packages/ui/src/components/ui-provider.tsx
import { MotionConfig } from "motion/react";
import type { ReactNode } from "react";

export interface UIProviderProps {
  children: ReactNode;
  reducedMotion?: "user" | "always" | "never";
}

export function UIProvider({ children, reducedMotion = "user" }: UIProviderProps) {
  return (
    <MotionConfig reducedMotion={reducedMotion} transition={{ duration: 0.15, ease: "easeOut" }}>
      {children}
    </MotionConfig>
  );
}
```

#### `Sidebar` primitive (new)

Replaces `web/src/components/ui/sidebar.tsx` and `web/src/components/app-sidebar.tsx` shell. Domain content (workspace list, agent tree, nav items) is passed as slots.

```tsx
// packages/ui/src/components/sidebar.tsx
export interface SidebarProps {
  rail: ReactNode;            // 40-44px workspace switcher
  header?: ReactNode;         // wordmark + version
  nav: ReactNode;             // section headers + rows
  footer?: ReactNode;         // connection status + settings
  collapsed?: boolean;
  onCollapse?: (next: boolean) => void;
}

export function Sidebar(props: SidebarProps): JSX.Element;
```

#### `SplitPane` primitive (new)

Two-column layout: fixed-width list (default 340px) + flex detail. Used across Network, Automation, Bridges, Knowledge, Skills, Tasks list view, Session.

```tsx
export interface SplitPaneProps {
  list: ReactNode;
  detail: ReactNode;
  listWidth?: number;     // default 340
  detailEmpty?: ReactNode; // shown when nothing selected
}
```

#### `PageHeader`, `Pills`, `SearchInput`, `Empty`, `Section`, `Metric` (new; rebuilt from `design-system/`)

Signatures match `docs/design/web-inspiration/src/primitives.jsx` with TypeScript types and variants. Each ships with `.stories.tsx` covering variants + empty + loading states.

#### `MonoBadge`, `KindChip`, `CodeBlock`, `StatusDot`, `ConnectionIndicator` (new)

Small visual primitives per DESIGN.md §4. `StatusDot` accepts `tone: "success" | "warning" | "danger" | "info" | "accent" | "neutral"` + `pulse?: boolean`.

#### `ChatMessageBubble`, `ToolCallCard` (new, shells only)

Style-only shells. They receive children (body, meta, status) from domain code; they do not know about session state.

### Data Models

No domain data models change. The TechSpec impacts only component props and CSS tokens.

#### Token additions (`packages/ui/src/tokens.css`)

| Token | Value | Rationale |
|-------|-------|-----------|
| `--radius-chip` | `5px` | Kind chips (DESIGN.md §5) |
| `--radius-mono-badge` | `6px` | Already documented, not in tokens.css |
| `--font-display` | `"Playfair Display", "Inter Variable", serif` | Marketing hero (not used in web/, kept for packages/site parity) |
| `--font-wordmark` | `"NuixyberNext", var(--font-sans)` | Wordmark lockup |
| `--duration-fast` | `100ms` | Tooltip / fast hover |
| `--duration-base` | `150ms` | Standard hover/focus |
| `--duration-slow` | `200ms` | Panel / modal / sidebar |
| `--ease-out` | `cubic-bezier(0.2, 0, 0, 1)` | Default easing |
| `--ease-in-out` | `cubic-bezier(0.4, 0, 0.2, 1)` | Symmetric transitions |
| `--color-accent-tint-strong` | `#E8572A3D` | ~24% alpha — hover on accent-tinted surfaces |

### API Endpoints

N/A — no backend changes.

## Integration Points

None. The redesign is contained to the React frontend + design system.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `packages/ui/src/tokens.css` | modified | Add ~9 new tokens; all existing tokens stay value-stable. Low risk. | Extend file; update stories that consume tokens. |
| `packages/ui/src/components/` | modified + new | Grows from 12 to ~35 primitives. 23 new components (Dialog, Popover, Combobox, Command, Sheet, Select, Tabs, Tooltip, ScrollArea, Switch, Toggle, Sidebar, SplitPane, PageHeader, Pills, SearchInput, Empty, Section, Metric, MonoBadge, KindChip, CodeBlock, StatusDot, ConnectionIndicator, ChatMessageBubble, ToolCallCard, UIProvider). Medium risk — primitive API shifts can cascade. | Build primitives story-first; merge each with Playwright snapshots in place (ADR-005). |
| `packages/ui/package.json` | modified | Add `motion` as peer + dev dep. New keywords. | `pnpm add -D motion`; update peer deps. |
| `packages/ui/src/index.ts` | modified | ~25 new named exports. No removals in Phase 1 (additions only). | Regenerate export list; tests check public surface. |
| `web/src/components/ui/**` | deprecated → deleted | 27 shadcn primitives deleted during Phase 1–2. High risk if any path is missed. | Grep-audit import sites; move to `@agh/ui` or delete if unused. |
| `web/src/components/design-system/**` | deprecated → deleted | 13 branded components deleted. The `/design-system` route (`design-system-showcase.tsx`) stays but becomes a consumer of `@agh/ui`. Medium risk. | Rewrite showcase against `@agh/ui` exports. |
| `web/src/components/app-sidebar.tsx` | rewritten | Becomes a thin composition over `@agh/ui` `Sidebar`. High visual impact. | Full rewrite in Phase 2. |
| `web/src/routes/__root.tsx`, `_app.tsx` | modified | Shell layout + motion wiring. Medium risk — affects every route. | Phase 2 PR; Playwright snapshots per route. |
| `web/src/systems/tasks/**` | modified (visual) | 27 components rewritten against new primitives. Largest domain. | Phase 3, own PR. |
| `web/src/systems/session/**` | modified (visual) | 19 components. Critical path. | Phase 4, own PR. |
| `web/src/systems/network/**` | modified (visual) | Split-pane list+detail. | Phase 5. |
| `web/src/systems/automation/**` | modified (visual) | Jobs + triggers tabs. | Phase 5. |
| `web/src/systems/bridges/**` | modified (visual) | List + detail. | Phase 5. |
| `web/src/systems/knowledge/**` | modified (visual) | List + detail. | Phase 5. |
| `web/src/systems/skills/**` | modified (visual) | Installed + marketplace tabs. | Phase 5. |
| `web/src/systems/workspace/**`, `daemon/**`, `agent/**` | modified (visual, derived) | Not in mock; derive from Sidebar + PageHeader + Metric patterns. | Phase 5 batched. |
| `web/src/routes/_app/settings/**` | modified (visual) | 11 sub-routes. Volumetrically large but isolated. | Phase 6, own PR. |
| `web/src/styles.css` | modified | Imports new tokens; drops local primitive class definitions. Low risk after primitives are in `@agh/ui`. | Trim file to imports + globals. |
| `web/package.json` | modified | Add `motion` as runtime dep. | `pnpm add motion`. |
| `web/e2e/__snapshots__/` | new | Playwright visual baselines per route + state. | Generated during each phase. |
| `packages/ui/src/components/stories/__snapshots__/` | new | Playwright visual baselines per story. | Generated in Phase 1. |
| 104 existing `.stories.tsx` | modified or moved | Primitive stories move to `packages/ui`; domain stories stay in `web/` but are rewritten against new primitives. | Move in Phase 1; rewrite during each domain phase. |
| MSW handlers (`web/src/integrations/tanstack-query/**`) | unchanged | Deliberately untouched. | None. |
| `web/src/generated/agh-openapi.d.ts` | unchanged | API layer untouched. | None. |

## Testing Approach

### Unit Tests

- **`@agh/ui`** — every primitive gets a Vitest test covering: render-without-crash, variant coverage, a11y attrs (axe via `@storybook/addon-a11y` per story), keyboard interaction where applicable (Dialog trap focus, Popover open/close with Esc).
- **`web/` systems** — existing Vitest tests stay. Where a component's props change, the test is updated in the same PR. Where a component is rewritten but props are identical, existing tests must keep passing unchanged — this is the compatibility signal.
- Mocks: `@agh/ui` tests never mock React or motion; motion is real with `prefers-reduced-motion: reduce` forced globally in `packages/ui/src/test-setup.ts`.
- Coverage threshold: 80% per package (per `CLAUDE.md`).

### Integration Tests

- **Storybook interaction tests** — each primitive story has a `play()` function for stateful primitives (Dialog, Popover, Combobox, Pills, Sidebar collapse). Runs under `pnpm test:storybook`.
- **Route stories** — each route has at least one story rendering the full page with MSW fixtures. Storybook interaction tests assert the primary user action works (select a row, open detail, submit form).
- **Playwright e2e** — existing suite in `web/e2e/` stays. Add: Playwright visual snapshots per story + per route (ADR-005). Snapshots generated on Ubuntu 22.04 CI runner for determinism.
- **Visual regression** — `pnpm test:visual` runs Playwright against built Storybook + dev server of `web/`. Threshold 0.1% pixel diff per snapshot.

### Verification checklist per phase

1. `make verify` passes (fmt + lint + test + build) on Go + frontend.
2. `pnpm test` passes (unit + storybook interaction).
3. `pnpm test:visual` passes (no baseline drift, or intentional drift with updated baselines committed).
4. Manual review of the affected routes in dev mode.
5. No remaining imports from deleted paths (CI grep: `grep -r "from '@/components/ui/" web/src` must be empty after Phase 2; `grep -r "from '@/components/design-system/" web/src` must be empty after Phase 2).

## Development Sequencing

### Build Order

1. **Phase 1 — Tokens + primitives (packages/ui)**. No dependencies. Deliverables: token additions, 23 new primitives with stories + Playwright snapshots, `UIProvider` with `motion` wiring, `packages/ui/README.md` contributor guide.
2. **Phase 2 — App shell (web/)**. Depends on step 1. Deliverables: rewritten `app-sidebar.tsx`, `__root.tsx`, `_app.tsx` layout; delete `web/src/components/ui/**` and `web/src/components/design-system/**`; update all imports to `@agh/ui`; route-level motion; Playwright snapshots of the shell surrounding each existing route (page interiors may still visually lag until their own phase).
3. **Phase 3 — Tasks domain (web/src/systems/tasks)**. Depends on step 2. Deliverables: rewritten 27 components; 4 views (List, Kanban, Dashboard, Inbox); 5 routes; empty/loading/error snapshots.
4. **Phase 4 — Session domain (web/src/systems/session)**. Depends on step 2 (not step 3 — parallelizable in theory, but serial in practice to let reviewers focus). Deliverables: rewritten 19 components; message thread + composer + inspector; end-to-end run against real SSE.
5. **Phase 5 — Remaining domains (network, automation, bridges, knowledge, skills, workspace, daemon, agent)**. Depends on step 2. Can be batched per domain; each domain is its own PR. Derivation rule for workspace/daemon/agent: use Sidebar + PageHeader + SplitPane + Metric + Empty consistently; no pixel-perfect mock available.
6. **Phase 6 — Settings (web/src/routes/_app/settings)**. Depends on step 2. 11 sub-routes; last because volumetrically large but isolated.

### Technical Dependencies

- **Infrastructure:** none new. Existing CI (GitHub Actions), `make verify`, Vitest, Playwright, Storybook 10, Tailwind v4 stay.
- **External:** `motion` package (npm, open source). Pin to a known-good minor version in `packages/ui` peer deps.
- **Team deliverables:** none blocking. The design reference (`DESIGN.md` + `docs/design/web-inspiration/`) is final.
- **Out of scope:** `packages/site/` visual changes. This TechSpec does not touch the marketing site; `packages/site/` may adopt the expanded `@agh/ui` independently later.

## Monitoring and Observability

No runtime monitoring changes — redesign is presentational.

Build-time observability additions:

- **Bundle size** — Vite bundle analyzer report committed to PR as artifact. Fail CI if first-route bundle grows >10% versus `main` (one-shot PR exception for the Phase 1 motion addition, approved explicitly).
- **Storybook build** — `pnpm build-storybook` must succeed in CI; size reported.
- **Snapshot count** — track # of snapshots committed per phase; if a PR deletes snapshots without replacement, flag for review.

## Technical Considerations

### Key Decisions

See ADRs for full rationale.

- **Single primitive library** (ADR-001). Chosen over two-layer or three-layer split. Trade-off: larger `@agh/ui` surface vs. one source of truth.
- **Greenfield migration** (ADR-002). Chosen over shims or flags. Trade-off: bigger PRs vs. clean tree.
- **`motion` library** (ADR-003). Chosen over motion-one and CSS-only. Trade-off: ~30KB bundle vs. correct unmount/route animations.
- **Phased rollout** (ADR-004). Chosen over big-bang and per-page. Trade-off: sequential vs. coherent foundations.
- **Playwright visual snapshots** (ADR-005). Chosen over Chromatic and manual. Trade-off: in-repo PNG storage vs. third-party cost.
- **Ownership split** (`@agh/ui` primitives, `web/src/systems` compositions). No ADR — derived from ADR-001 + app-renderer-systems pattern already in `web/CLAUDE.md`.

### Known Risks

- **Primitive API churn between Phase 1 and Phase 3.** If Phase 3 reveals a primitive contract that does not fit, we must either amend the primitive (cascades to every caller added in Phase 2) or accept a local workaround. **Mitigation:** Phase 1 primitives are built from `docs/design/web-inspiration/src/primitives.jsx` with real props observed in the mock; any API change during Phase 3 triggers a focused 1-primitive PR before the Phase 3 PR continues.
- **Visual snapshot flakiness on CI.** Font anti-aliasing drift, SSE timing, fixture non-determinism. **Mitigation:** pin CI image, use Playwright's bundled Chromium, force `prefers-reduced-motion`, use frozen fixtures in Storybook (MSW handlers with stable data).
- **Storybook + Playwright build time.** Current 104 stories + added primitives + route snapshots. Potential CI regression of several minutes. **Mitigation:** shard Playwright visual job in GitHub Actions (4 shards by default); budget exceeded → move snapshot CI to matrix.
- **Bundle growth from motion.** +30KB gzip. **Mitigation:** analyzer-enforced budget; if trouble, lazy-load motion for non-critical surfaces. Not pre-optimized now.
- **Cross-domain regressions.** A primitive change in Phase 1 inadvertently affects Phase 3 Tasks. **Mitigation:** `@agh/ui` Playwright snapshots catch primitive-level drift before any domain phase ships.
- **Developer onboarding cost.** Contributors in the middle of other work have to learn the new primitive set. **Mitigation:** `packages/ui/README.md` contributor guide (Phase 1 deliverable); DESIGN.md at repo root is already current; pair reviewers during first week of each phase.

## Architecture Decision Records

ADRs documenting key decisions made during technical design:

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md) — Single primitive library; delete `web/src/components/ui/` and `web/src/components/design-system/`; domain compositions stay in `web/src/systems/`.
- [ADR-002: Greenfield migration — delete without backwards-compat](adrs/adr-002.md) — No compat shims, no feature flags, no coexistence; old code leaves with the PR that replaces it.
- [ADR-003: Adopt `motion` (framer-motion successor) for UI animations](adrs/adr-003.md) — `motion` package for unmount + route transitions; CSS for hover/focus/keyframes; global `reducedMotion="user"`.
- [ADR-004: Phased rollout — tokens+primitives → shell → Tasks → Session → remaining domains → Settings](adrs/adr-004.md) — Six-phase rollout with explicit build order and per-phase end gates.
- [ADR-005: Visual parity via Playwright snapshots over Storybook stories](adrs/adr-005.md) — Playwright `toHaveScreenshot` over every story + route; in-repo PNGs; 0.1% pixel-diff threshold.
