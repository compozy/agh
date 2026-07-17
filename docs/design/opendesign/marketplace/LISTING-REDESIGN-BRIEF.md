# Listing redesign — rollout briefing

> **Purpose.** Issue-level briefing for a future TechSpec (`cy-create-techspec` input). It records what the OpenDesign listing artifacts lock in, which `web/` routes they were designed for, and which remaining list pages must be brought onto the same standard. The spec derived from this brief owns final scope, task decomposition, and the daemon-truth audit per page.

## 1. Problem

The `web/` list pages grew independently and now ship several competing patterns for the same job: different toolbars (kind/category PillGroup strips vs. ad-hoc filters vs. none), different row anatomies, tables where rows should be, bordered cards next to flat cards, inconsistent kind/type chips (mono-uppercase badges vs. Pills), and topbar action pairs with mismatched button sizes. The product reads as a collage of one-offs instead of one operator surface.

## 2. Design artifacts (source of truth for the pattern)

All under `docs/design/opendesign/`, all aligned this cycle against `packages/ui/src/tokens.css`, `DESIGN.md`, and the real component recipes in `packages/ui/src/components/**`:

| Artifact | Role |
| --- | --- |
| [`catalog-design-system.html`](./catalog-design-system.html) | **The canonical spec.** Live anatomy, class contract, do/don't tables, interactive playground, and the P0/P1/P2 redesign checklist (§11) for inventory pages. Start here. |
| [`loops-catalog.html`](./loops-catalog.html) | Reference implementation A — operator catalog: grouped rows (Built-in / Custom), status pills, 30d success rate, inline Run, Filters chip bar, Rows\|Cards toggle. |
| [`vault-redesign.html`](./vault-redesign.html) | Reference implementation B — inventory with mono refs, namespace Pills (`sessions` → info tone), delete flow (typed confirmation modal), create modal, cards without body copy. |
| [`LISTING-STANDARD.md`](./LISTING-STANDARD.md) | Written companion: rows-vs-cards decision table, toolbar/topbar anatomy, filter-field schema, row/card slot contracts, anti-patterns. |

### Locked pattern (summary — details live in the artifacts)

- **Topbar**: icon well (24px, `radius-sm`, elevated, accent stroke) + title (14/500, `tracking-tight`) + mono count chip; trailing = `PageActionsTopbarSlot` pair, **both `size="sm"` (22px)** — ghost secondary + accent primary. No breadcrumbs, no search in the topbar.
- **Page head**: `--text-detail-h1` title (1.4rem/600/−0.028em) + count chip + one dot-separated meta line.
- **Listing toolbar**: `SearchInput` (26px, min 220px, `/` shortcut) → reui `Filters` chip bar (label · op · value · ×, one chip per field, AND) → spacer → `PillGroup` Rows\|Cards (md, borderless track, elevated active). Default view **rows**; `view` persisted in URL search params.
- **Rows**: grid `[34px][minmax(0,1fr)][auto]`, name 14/500, kind tags as `Pill xs neutral` (sans), slugs/ids as `MonoId`, one-line goal/desc, mono meta facts, trail = type Pill / status Pill / stat / always-visible primary action. Hover = `--row-hover` only.
- **Cards**: `CatalogCard` — **borderless**, `canvas-soft` → `elevated` on hover, Meta line = Eyebrow contract (ids stay mono), type Pill in footer opposite the action.
- **States**: flat `<Empty>` (38px icon well, no dashed frame) with two copy paths (zero inventory vs. no matches + Clear filters); error state with Retry (see vault).
- **Anti-patterns** (blocking): accent side-stripes, hover lift, bordered/elevated invented chips, second segmented control duplicating a Filter field, "Sorted by…" without a real sort, fake counts/metrics.

## 3. Designed pages → `web/` routes

| Artifact | Route | Route file | Current implementation to replace/align |
| --- | --- | --- | --- |
| `loops-catalog.html` | `/loops` | `web/src/routes/_app/loops.tsx` | `web/src/systems/loops/components/catalog/loop-catalog.tsx` + `loop-catalog-row.tsx` + `loop-catalog-filters.tsx` — row anatomy already matches; **missing**: SearchInput, reui Filters chip bar (today kind/category PillGroup strips), Rows\|Cards toggle + CatalogCard view, URL-persisted `view`. |
| `vault-redesign.html` | `/vault` | `web/src/routes/_app/vault.tsx` | `web/src/systems/vault/components/vault-secrets-table.tsx` — today a table; redesign to listing rows/cards, namespace/prefix Filters, create + typed-delete modals per the artifact. |

## 4. Rollout backlog — pages to bring onto the standard

Every row below gets the full §2 pattern (topbar pair, page head, listing toolbar, rows default, empty/error states). Filter fields listed are **candidates** — the spec must validate each against real daemon surfaces (truthful UI: no filter/count/status the runtime does not expose).

| Route | Route file | Current implementation (entry point) | Notes for the spec |
| --- | --- | --- | --- |
| `/jobs` | `_app/jobs.tsx` | `web/src/systems/automation/components/automation-list-panel.tsx` + `automation-operations-page.tsx` | Filter candidates: `status`, `target` (loop/task), `schedule`. Rows only (no cards). Primary CTA: New job (`create-job-redesign.html` exists as a sibling artifact). |
| `/triggers` | `_app/triggers.tsx` | shared automation surface (same panel family as Jobs) | Filter candidates: `kind` (schedule/webhook/event), `status`, `target`. Rows only. Primary CTA: New trigger (`create-trigger-redesign.html` exists). |
| `/skills` | `_app/skills.tsx` | route-level PillGroup tabs + skill panels (`web/src/systems/skill/`) | Filter candidates: `source` (installed/marketplace), `kind`. Cards view is genuinely useful here (browse/install density) — same playground demo in the DS artifact uses Skills as its sample. |
| `/bridges` | `_app/bridges.tsx` | SplitPane master-detail (`web/src/systems/bridges/`) | Spec decision required: keep SplitPane and apply the row contract to the master list, or move to full listing + detail route. Filter candidates: `platform`, `status`. |
| `/tasks` | `_app/tasks.tsx` | `web/src/systems/tasks/` (dashboard + list) | Filter candidates: `status`, `priority`, `type`. Keep KPI/dashboard block separate from the listing contract — only the list adopts the standard. |
| `/loop-runs` | `_app/loop-runs.tsx` | `web/src/systems/loops/` runs surface | Filter candidates: `status`, `loop`, time window. Rows only; `runs.html` / `run-detail.html` artifacts are siblings. |
| `/knowledge` | `_app/knowledge.tsx` | SplitPane (`web/src/systems/knowledge/`) | Same SplitPane decision as Bridges. Filter candidates: `namespace`/`source`. |
| `/sandbox` | `_app/sandbox.tsx` | route-local implementation (no dedicated `systems/` dir) | Inventory of sandboxes/sessions; audit which facets the daemon exposes before promising filters. |

**Evaluate in the spec (explicitly in or out of scope):**

- `/` (Home, `_app/index.tsx`) and `agents.$name.tsx` session lists — session rows are a different contract (`RunCard` / session anatomy); decide whether they adopt the toolbar only.
- `/network` channel/thread lists — protocol surfaces with their own wire vocabulary; likely out.
- Settings inventories (`settings/mcp-servers`, `settings/providers`, `settings/skills`, `settings/hooks-extensions`) — list-like but they live inside settings chrome (`FormSection`); the spec must state whether they adopt the listing toolbar or stay settings-native.

## 5. Engineering notes (for the spec)

- **All primitives exist** in `@agh/ui`: `SearchInput`, reui `Filters` (`packages/ui/src/components/reui/filters.tsx`), `PillGroup`, `Pill`/`KindChip`/`MonoId`, `CatalogCard`, `Empty`, `ListGroup`, `Topbar` + `PageActionsTopbarSlot`. No new tokens needed.
- **Extract shared composites** instead of re-implementing per page: a `ListingToolbar` (search + filters + view toggle + URL state) and a `ListingRow` slot component would prevent the per-page drift that motivated this redesign. Candidate home: `packages/ui/src/components/custom/`.
- **URL contract**: `view=rows|cards` + filter state in search params, consistent across all pages (TanStack Router search middleware).
- **`/` shortcut** focuses search on every listing page; must not fire while typing in inputs.
- **Acceptance gate**: the P0/P1/P2 checklist in `catalog-design-system.html` §11, plus `agh-ui-screenshot` captures against the artifacts for visual parity, plus the standard `make verify` lane.
- Per repo policy, each derived task carries its own Web/Docs Impact and AGH Impact Audit; QA rows in `docs/qa/state.csv` reset to `untested` per redesigned route.

## 6. Out of scope for this brief

Create/detail/editor surfaces (`loop-create.html`, `loop-detail.html`, `create-job-redesign.html`, `create-task-redesign.html`, `create-trigger-redesign.html`, `run-detail.html`) are separate artifacts with their own briefs; this rollout covers **listing pages only**.
