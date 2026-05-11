# Design System: AGH

**Artificial General Hivemind** — one visual language across the runtime daemon, the shared UI kit, and the marketing + docs site.

AGH ships as three surfaces that must feel like one product:

1. **AGH Runtime** — the local daemon, operator UI (`web/`) and CLI. Sessions, memory, skills, workspaces, automation, bridges, observability.
2. **AGH Network** — `agh-network/v0`, the open agent network protocol with seven message kinds (`greet`, `whois`, `say`, `direct`, `capability`, `receipt`, `trace`) over NATS + JSON.
3. **packages/site** — the marketing landing + Fumadocs MDX docs at `agh.network` with two trees (`/runtime/*`, `/protocol/*`).

The canonical token source is [`packages/ui/src/tokens.css`](packages/ui/src/tokens.css). The canonical reference extraction and UI kits live in [`docs/design/design-system/`](docs/design/design-system/).

## 1. Visual Theme & Atmosphere

AGH is a control surface for running real agent work. The aesthetic is **warm dark operator** — a near-black canvas (#131211) with a slight warm tint away from pure black, broken by a single operator-orange accent (#E8572A). Depth is layered surfaces, not shadows. Type does the heavy lifting: Inter Variable for UI, JetBrains Mono for all metadata, with Playfair Display reserved for the marketing site hero (`packages/site` only) and NuixyberNext reserved for the `agh` wordmark only.

This is a **flat depth model**. No gradients on content, no glassmorphism, no decorative texture. Depth comes from a layered surface ramp (rail → canvas → soft → tint → elevated) and translucent 1 px hairlines. Color is signal, never decoration — accent means _act_, green means _stable_, red means _stop_, yellow means _caution_, purple means _informational_.

**Key qualities:**

- **Warm, not neutral** — the whole gray ramp is tinted warm (`#0c0b0b`, `#131211`, `#1a1918`, `#232220`), not pure iOS gray. Never cool or bluish.
- **Dark mode only** — `color-scheme: dark` is hardcoded. There is no light mode. Assets must never render on a white background.
- **Monochromatic + accent** — a single warm orange (#E8572A) is the only hue that breaks the neutral ramp. Signal colors are desaturated (`#5fbf85`, `#d6a647`, `#e0635a`, `#8e8eb5`) and only appear as low-alpha tints, never as solid banners.
- **Two-token shadow vocabulary** — only `--shadow-overlay` (modals) and `--highlight` (active rim) carry shadow. Cards, popovers, dropdowns, sticky headers, list rows stay flat with a 1 px ring on `--line-soft`.
- **Editorial calm, operator density** — the marketing site uses generous Playfair Display headings via its own font stack; the runtime kit is dense Inter + mono metadata. Different surfaces, different rhythms, same token contract for color and motion.
- **Operational, not decorative** — every color, every chip, every icon carries meaning. No ornament without function.

## 2. Color Palette & Roles

All colors are hex or rgba. No OKLCH, no `color-mix()` at the token level. Hairlines and tints use rgba so transparency layers correctly across the warm ramp.

### Surface ramp

| Token                           | Value     | Role                                                      |
| ------------------------------- | --------- | --------------------------------------------------------- |
| **Rail** `--rail`               | `#0c0b0b` | Workspace rail (the 56 px left strip)                     |
| **Canvas** `--canvas`           | `#131211` | Page bg — warm near-black                                 |
| **Canvas Soft** `--canvas-soft` | `#1a1918` | Card / group / sidebar / popover bg                       |
| **Canvas Tint** `--canvas-tint` | `#1c1b1a` | Kanban card baseline; subtle elevation step inside groups |
| **Sidebar** `--sidebar`         | `#1a1918` | Sidebar panel (semantic alias of `--canvas-soft`)         |
| **Elevated** `--elevated`       | `#232220` | Active row, segment-active, selected state                |
| **Hover** `--hover`             | `#1f1e1d` | Generic neutral hover                                     |
| **Disabled** `--disabled`       | `#4a4847` | Disabled fill                                             |

### Hairlines

The new token contract uses translucent white rails so dividers layer correctly across every ramp step. No solid `#3C3A39` divider.

| Token           | Value                        | Role                              |
| --------------- | ---------------------------- | --------------------------------- |
| `--line`        | `rgba(255, 255, 255, 0.055)` | Generic 1 px hairline             |
| `--line-soft`   | `rgba(255, 255, 255, 0.03)`  | Group bottoms, popover ring       |
| `--line-strong` | `rgba(255, 255, 255, 0.09)`  | Focus ring, scrollbar thumb hover |

### Text

Five-step neutral text scale. Body lands on `--fg`; titles step up to `--fg-strong`; placeholders, separators, and mono ids step down through `--muted` → `--subtle` → `--faint`.

| Token         | Value     | Role                                |
| ------------- | --------- | ----------------------------------- |
| `--fg`        | `#ececef` | Body                                |
| `--fg-strong` | `#f6f6f8` | Titles, active labels               |
| `--muted`     | `#9a9a9f` | Secondary copy, helper text         |
| `--subtle`    | `#76767c` | Placeholders, low-emphasis labels   |
| `--faint`     | `#545458` | Mono ids, separators, disabled text |

### Accent

Color is signal. The accent has exactly one meaning: action.

| Token                  | Value                     | Role                                                       |
| ---------------------- | ------------------------- | ---------------------------------------------------------- |
| `--accent`             | `#e8572a`                 | CTAs, primary buttons, active pills, links, `$ ` prompts   |
| `--accent-hover`       | `#d14e25`                 | Hover / pressed state for accent fills                     |
| `--accent-strong`      | `#f6874f`                 | Rare high-emphasis accent (inline code on dark panels)     |
| `--accent-ink`         | `#17110f`                 | Text on accent fill                                        |
| `--accent-tint`        | `rgba(232, 87, 42, 0.10)` | Chip / pill tint                                           |
| `--accent-tint-strong` | `rgba(232, 87, 42, 0.16)` | Bar fill, sticky highlight                                 |
| `--accent-dim`         | `rgba(232, 87, 42, 0.24)` | Legacy focus ring (deprecated; focus uses `--line-strong`) |
| `--accent-glow`        | `rgba(232, 87, 42, 0.05)` | Pulse keyframe base                                        |

### Signal palette (desaturated)

The proposal mock's saturated iOS signals (`#30D158`, `#FFD60A`, `#FF453A`, `#BF5AF2`) are replaced with desaturated variants that compose against the warm ramp without screaming. Tints sit at 6–10 % alpha — chips are quiet by default; full-color text rides on the tint surface.

| Role    | Token       | Value     | Tint token       | Tint value                  | Tint α |
| ------- | ----------- | --------- | ---------------- | --------------------------- | ------ |
| Success | `--success` | `#5fbf85` | `--success-tint` | `rgba(95, 191, 133, 0.08)`  | 8 %    |
| Warning | `--warning` | `#d6a647` | `--warning-tint` | `rgba(214, 166, 71, 0.08)`  | 8 %    |
| Danger  | `--danger`  | `#e0635a` | `--danger-tint`  | `rgba(224, 99, 90, 0.09)`   | 9 %    |
| Info    | `--info`    | `#8e8eb5` | `--info-tint`    | `rgba(142, 142, 181, 0.12)` | 12 %   |
| Neutral | `--neutral` | `#7a7a80` | `--neutral-tint` | `rgba(150, 150, 155, 0.06)` | 6 %    |

`--info-tint` is the canonical Settings / observability tint (UsageBreakdown session bar, observability dashboards). `--neutral-tint` warms toward `rgba(150, 150, 155, …)` to match the rest of the warm-dark ramp.

### WCAG AA pairs

Required minimum: 4.5:1 for body text and labels, 3:1 for non-text indicators (signal dots, icon-only badges). Pairs pinned in `web/src/__tests__/styles.test.ts`; a hex retune fails the gate.

| Foreground / use            | Background      | Required | Target hex   | Action if below               |
| --------------------------- | --------------- | -------- | ------------ | ----------------------------- |
| `--fg` body text            | `--canvas`      | ≥ 4.5:1  | `#ececef`    | retune toward `#f6f6f8`       |
| `--fg-strong` titles        | `--canvas-soft` | ≥ 4.5:1  | `#f6f6f8`    | retune toward `#ffffff`       |
| `--muted` secondary text    | `--canvas-soft` | ≥ 4.5:1  | `#9a9a9f`    | retune toward `#a8a8ad`       |
| `--subtle` placeholder      | `--canvas-soft` | ≥ 4.5:1  | `#76767c`    | retune toward `#86868c`       |
| `--accent` text on tint     | `--canvas-soft` | ≥ 4.5:1  | `#e8572a`    | retune brand-anchored variant |
| Success text on tint        | `--canvas-soft` | ≥ 4.5:1  | `#5fbf85`    | retune toward `#73d199`       |
| Warning text on tint        | `--canvas-soft` | ≥ 4.5:1  | `#d6a647`    | retune toward `#e2b85b`       |
| Danger text on tint         | `--canvas-soft` | ≥ 4.5:1  | `#e0635a`    | retune toward `#ed7670`       |
| Info text on tint           | `--canvas-soft` | ≥ 4.5:1  | `#8e8eb5`    | retune toward `#9c9cc4`       |
| Signal dot fills (non-text) | `--canvas-soft` | ≥ 3:1    | (each above) | retune as above               |

### Overlays

| Purpose         | Token                         | Value                       | Notes                                                                                                                                                                               |
| --------------- | ----------------------------- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Modal scrim** | `--overlay-scrim`             | `rgba(0, 0, 0, 0.55)`       | Dialog / sheet backdrop (retuned for stronger detach)                                                                                                                               |
| **Modal blur**  | `--overlay-blur`              | `3px`                       | Carve-out — `backdrop-filter: blur(var(--overlay-blur))` ONLY on `.dialog-scrim` / `.sheet-scrim`. Forbidden elsewhere                                                              |
| **Ghost hover** | `--overlay-ghost-hover`       | `rgba(255, 255, 255, 0.06)` | Ghost button hover on dark                                                                                                                                                          |
| **Selection**   | (none — uses `--accent-tint`) | `rgba(232, 87, 42, 0.10)`   | `::selection { background: var(--accent-tint); color: var(--fg-strong); }` shipped in `packages/ui/src/tokens.css`. The legacy `--overlay-selection` token is intentionally absent. |

The runtime kit's blur is bounded to dialog / sheet scrims only. Every other surface stays blur-free (DESIGN.md §10). The marketing site's sticky header (`packages/site`) keeps its own blur via its own stack.

## 2.5 Surface glaze ladder

Translucent white tints layered on top of the warm surface ramp. They compose consistently across `--canvas`, `--canvas-soft`, `--canvas-tint`, and `--elevated` — that's the whole reason they exist. Inline `rgba(255, 255, 255, 0.0XX)` literals are forbidden under `web/src/**` and `packages/ui/src/**`; the lint plugin enforces this through `no-design-glaze-rgba`.

| Token                 | Value                        | Role                                                                |
| --------------------- | ---------------------------- | ------------------------------------------------------------------- |
| `--row-hover`         | `rgba(255, 255, 255, 0.022)` | List row / nav item hover; also aliased as `--hover`                |
| `--row-selected`      | `rgba(255, 255, 255, 0.030)` | List row / nav item selected baseline                               |
| `--surface-glaze`     | `rgba(255, 255, 255, 0.040)` | Selected card surface (RadioCard, kanban card selected, panel head) |
| `--bar-fill`          | `rgba(255, 255, 255, 0.085)` | Bar fills (priority bars, progress strips, usage bars)              |
| `--input-fill`        | `rgba(255, 255, 255, 0.025)` | Composer / textarea / sentinel input surface                        |
| `--btn-default-fill`  | `rgba(255, 255, 255, 0.040)` | Neutral `<Button>` default fill                                     |
| `--btn-default-hover` | `rgba(255, 255, 255, 0.070)` | Neutral `<Button>` hover fill                                       |
| `--badge-fill`        | `rgba(255, 255, 255, 0.050)` | `<PillGroup>` count badge background                                |

`--hover` is declared as `var(--row-hover)` so `hover:bg-(--hover)` resolves the canonical row-hover tint (fixes the live N-45 bug where the alias was referenced but undefined). Surfaces that want a stronger lift (`--row-selected`) or selection emphasis (`--surface-glaze`) consume the explicit token, not `--hover`.

## 2.6 Owner avatar palettes

Owner avatars resolve through `web/src/lib/owner-palette.ts → colorsFor(ownerKind, ownerId)`. The palette is tokenised so Storybook, design ref tools, and the runtime consume the same source. Hash on the owner id selects a slot; agents and humans get distinct families; system owners land on a single neutral slot.

| Family | Slot | Background                                        | Foreground                           |
| ------ | ---- | ------------------------------------------------- | ------------------------------------ |
| Agent  | 0    | `--avatar-agent-0-bg` `rgba(232, 144, 99, 0.18)`  | `--avatar-agent-0-fg` `#F2B895`      |
| Agent  | 1    | `--avatar-agent-1-bg` `rgba(168, 178, 220, 0.16)` | `--avatar-agent-1-fg` `#C5CCE7`      |
| Agent  | 2    | `--avatar-agent-2-bg` `rgba(143, 196, 178, 0.18)` | `--avatar-agent-2-fg` `#A9D9C7`      |
| Agent  | 3    | `--avatar-agent-3-bg` `rgba(214, 168, 192, 0.18)` | `--avatar-agent-3-fg` `#E0BCD0`      |
| Human  | 0    | `--avatar-human-0-bg` `rgba(220, 192, 134, 0.20)` | `--avatar-human-0-fg` `#E5CC9A`      |
| Human  | 1    | `--avatar-human-1-bg` `rgba(195, 178, 156, 0.20)` | `--avatar-human-1-fg` `#D6C5AA`      |
| Human  | 2    | `--avatar-human-2-bg` `rgba(192, 173, 178, 0.20)` | `--avatar-human-2-fg` `#D2BFC5`      |
| System | —    | `--avatar-system-bg` `var(--elevated)`            | `--avatar-system-fg` `var(--subtle)` |

The `<OwnerAvatar>` primitive (`packages/ui/src/components/custom/owner-avatar.tsx`) is the only consumer. Sizes: `sm` (20 × 20), `default` (24 × 24), `lg` (32 × 32). Two-character monogram default, optional `glyph` / `icon` slot for system owners.

## 2.7 Status tone vocabulary

The runtime palette has six tones. `violet` is not a token; the proposal's `violet` mapping for approvals collapses to `info`.

| Tone      | Resolves to                    | Used for                                                       |
| --------- | ------------------------------ | -------------------------------------------------------------- |
| `neutral` | `--neutral` / `--neutral-tint` | Idle, draft, pending, cancelled lanes; default chrome          |
| `accent`  | `--accent` / `--accent-tint`   | Primary CTAs, mention lane, brand identity surfaces (RailLogo) |
| `success` | `--success` / `--success-tint` | Completed, healthy, stable peaks                               |
| `warning` | `--warning` / `--warning-tint` | Partial, degraded, stuck peaks                                 |
| `danger`  | `--danger` / `--danger-tint`   | Blocked, failed, halted lanes; destructive actions             |
| `info`    | `--info` / `--info-tint`       | In-progress, approvals lane, informational chrome              |

`task_status_tone`, `task_run_status_tone`, and `task_lane_tone` resolve through these six tones via the exhaustive `STATUS_TONE` / `RUN_STATUS_TONE` / `TASK_LANE_TONE` dictionaries in `web/src/lib/status-tone.ts`. The dictionaries are typed `satisfies Record<…, PillTone>` against the backend `Task.Status` / `TaskRunStatus` Go enums — drift caught at `make codegen-check`. Adding a new tone requires updating the design-system contract.

## 3. Typography Rules

### Font Families

The runtime kit ships two faces — Inter for everything readable, JetBrains Mono for every metadata surface. Playfair Display and NuixyberNext are NOT in the kit's `tokens.css`; the marketing site loads them through its own font stack.

| Role                | Typeface             | Surface                             | Notes                                                               |
| ------------------- | -------------------- | ----------------------------------- | ------------------------------------------------------------------- |
| **Primary (Sans)**  | **Inter Variable**   | runtime kit + site                  | Body, UI, docs headings, buttons — every readable surface           |
| **Mono**            | **JetBrains Mono**   | runtime kit + site                  | Labels, badges, eyebrows, code, counters, protocol strings          |
| **Display (Serif)** | **Playfair Display** | `packages/site` only (`.site-home`) | Marketing hero + section H2. Loaded via the site's Next.js font set |
| **Wordmark**        | **NuixyberNext**     | `packages/site` only (header logo)  | The literal string `agh` only — nowhere else                        |

The kit's `tokens.css` declares only `--font-sans` and `--font-mono`. `--font-display` and `--font-wordmark` are deleted from the kit; the site re-declares them inside its own `@theme inline` block.

### Type Ladder (Inter-510 runtime kit)

The runtime kit and `@agh/ui` settle on **Inter-510** for almost every UI weight — a custom value between Medium (500) and SemiBold (600) that Inter Variable supports natively. Body text steps down to weight 400 only for the small body size. Pill `--mono` is the one JetBrains Mono row inside the operator UI ladder.

| Role                   | Family         | Size    | Weight | Tracking  | Notes                                                                                                                                                                                                    |
| ---------------------- | -------------- | ------- | ------ | --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Body                   | Inter          | 13.5 px | 400    | -0.006em  | Default reading text                                                                                                                                                                                     |
| Page H1                | Inter          | 22 px   | 510    | -0.026em  | Top-level route title                                                                                                                                                                                    |
| Detail H1              | Inter          | 24 px   | 510    | -0.028em  | Detail-page hero title                                                                                                                                                                                   |
| Topbar title           | Inter          | 14 px   | 510    | -0.014em  | Shell topbar route title                                                                                                                                                                                 |
| Empty title            | Inter          | 18 px   | 510    | -0.022em  | RouteState empty / loading title                                                                                                                                                                         |
| Eyebrow `case="upper"` | JetBrains Mono | 11 px   | 510    | 0.06em UC | UPPERCASE eyebrow scope — sidebar label, table head, run-cell, dashboard label, breadcrumb. Always rendered through `<Eyebrow>` (`@agh/ui`); never inlined. Tokens: `--text-eyebrow` + `--tracking-mono` |
| Eyebrow `size="badge"` | JetBrains Mono | 10 px   | 510    | 0.06em UC | Denser eyebrow variant for chips and metadata badges. Tokens: `--text-badge` + `--tracking-mono`                                                                                                         |
| Eyebrow `size="micro"` | JetBrains Mono | 9 px    | 510    | 0.06em UC | Tightest mono uppercase tier (live-tab counters, kbd hints). Tokens: `--text-micro` + `--tracking-mono`                                                                                                  |
| Pill default           | Inter          | 11 px   | 510    | -0.005em  | Sentence case                                                                                                                                                                                            |
| Pill `--mono`          | JetBrains Mono | 10.5 px | 500    | 0         | Inline mono pills (ids)                                                                                                                                                                                  |
| Button                 | Inter          | 12 px   | 510    | -0.005em  | Primary / secondary button label                                                                                                                                                                         |

### Typography Principles

- **Inter is the runtime ladder.** No Playfair, no NuixyberNext anywhere inside `web/` or `packages/ui` — those faces live in `packages/site` only.
- **Inter weight 510 sits between Medium and SemiBold** — slightly punchier than 500 without the dense feel of 600. Body stays at 400.
- **UPPERCASE is scoped.** Only sidebar labels, table heads, run-cell mono labels, and Eyebrow `case="upper"` go uppercase. Pill default, button labels, and copy stay sentence case (`Eyebrow case="sentence"` is the default).
- **Eyebrow markup is mandatory.** Every uppercase mono label in `web/` and `packages/site` MUST use either (a) the `<Eyebrow>` component (`@agh/ui`) for cases with dynamic tone/size/weight, or (b) the static utility classes `eyebrow` / `eyebrow-badge` / `eyebrow-micro` (defined in `packages/ui/src/tokens.css`) on structural elements (`<dt>`, `<label>`, `<TableHead>`, sidebar section labels, etc.). Inlining `font-mono` + `uppercase` + `text-eyebrow|badge|micro` + `tracking-mono` is forbidden — that combination IS the utility. Use `font-semibold!` (with `!`) when overriding the utility's baked-in `font-medium`. `@agh/ui` ships structural primitives (`<Sidebar.SectionLabel>`, `<TableHead>`, `<MetadataList.Term>`, `<WireCardHead>`) that already apply the utility internally. Canonical tracking is `--tracking-mono` (0.06em). Never reach for arbitrary values like `tracking-[0.05em]`, `text-[10.5px]`, or the legacy `--tracking-badge` (0.08em) for eyebrow text.
- **Negative tracking on titles** (-0.014em to -0.028em) tightens the page-h1 / detail-h1 / topbar-title hierarchy. Body and small text use a slight negative tracking (-0.005em to -0.006em) for crisp ranges on the warm canvas.
- **No bold UI weight.** The ladder tops out at 510. Body is 400, never bold.
- **Signal text on tint surfaces** — text uses the full signal hex, the tint token sits behind. The color carries the meaning.
- **Selection color** uses `var(--accent-tint)` (`rgba(232, 87, 42, 0.10)`). Warm accent, not default blue. The legacy `--overlay-selection` token is intentionally absent.

## 3a. Layout Grammar

A flat, prop-driven layout vocabulary that sits between the token sheet (§2) and the per-component contracts (§4). Every runtime surface is composed from the four primitives below.

### Shell skeleton

```
+---------------------------------------------+
| Topbar (h-12 / 14 px title / count chip)    |  <-- page identity lives here
+---------------+-----------------------------+
| Sidebar (244) | <PageShell density="route"> |  <-- 28/36/80 envelope
| collapse      |   page body                 |
|  244 / 220 /  |   detail surface: <DetailHeader> 24 px H1 + crumbs
|  drawer       |                             |
+---------------+-----------------------------+
```

- **Topbar height redundancy** is intentional: the shell row reserves `grid-rows-[48px_1fr]` so content does not jump while the `<Topbar>` (`h-12` = 48 px) mounts.
- **Sidebar collapse ladder**: default 244 px → 220 px when viewport ≤ 1100 px → drawer when viewport ≤ 880 px. Defined in `packages/ui/src/components/sidebar.tsx` constants (`SIDEBAR_PANEL_WIDTH_DEFAULT`, `SIDEBAR_PANEL_WIDTH_MD`, `SIDEBAR_PANEL_WIDTH_MD_BREAKPOINT`, `SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT`). The previous 768 px breakpoint is removed.

### Page envelope

`<PageShell density="route">` applies the proposal envelope `28 px` block-start, `36 px` inline, `80 px` block-end. No `max-width` cap — content stretches edge-to-edge under the topbar / sidebar. The existing `density="comfortable" | "compact"` callers are unaffected; they opt into `"route"` explicitly as systems migrate.

### Detail surfaces

`<DetailHeader>` is the only primitive emitting a 24 px H1 in-body. It owns the 6-row anatomy: `crumbs / pre-title / 24 px H1 / pills / meta / actions`. The 22 px body-side H1 is forbidden — non-detail routes carry page identity in the topbar's 14 px route title only.

### Modal anatomy

Three modal widths via tokens; no off-ladder literals.

| Token              | Value | Used for                                                |
| ------------------ | ----- | ------------------------------------------------------- |
| `--width-modal-sm` | 560px | Confirm dialog, small editor (single field, single CTA) |
| `--width-modal-md` | 720px | Task editor modal (new / edit), settings field editor   |
| `--width-modal-lg` | 880px | Bridges add-bridge wizard, knowledge create dialog      |

`<Dialog.Overlay>` / `<Sheet.Overlay>` are the only surfaces that resolve `--overlay-blur` (3 px) on top of `--overlay-scrim` (0.55) — every other element stays blur-free (§10).

### Radii ladder (4 / 5 / 6 / 8 / 10 / 14 / pill)

Same ladder as §5, with three notes specific to the layout vocabulary:

- **`--radius-xs` (4 px) and `--radius` (6 px) are pinned at `:root`**. They mirror inside `@theme inline` so Tailwind utilities (`rounded-xs`, `rounded`) stay generated; the `:root` pin makes them resolvable inside any custom-property cascade without depending on the `@theme inline` layer.
- **`--radius-mono-badge` is 4 px** (retuned from 6 px). Topbar count badges, `<MonoId>`, and any other dense mono chip use the sharper rim.
- **`--radius-icon-well` stays in rem (`0.625rem`)** — the single rem-based exception in an otherwise px ladder. Icon-well sizing scales with root `font-size` for in-body affordances. Do not normalise it back to `10px`.

### Sizing tokens

Two-tier logo well; the catalog tier is for browse/install, the provider tier for connected/configured.

| Token                       | Value            | Used for                                                                       |
| --------------------------- | ---------------- | ------------------------------------------------------------------------------ |
| `--size-catalog-logo`       | `1.5rem` (24 px) | `<CatalogCard logoSize="default">` (skill / model / bridge browse)             |
| `--size-provider-logo-well` | `2.5rem` (40 px) | `<CatalogCard logoSize="lg">` (settings provider card, configured bridge card) |

`<CatalogCard logoSize="default" | "lg">` maps onto these. Inline arbitrary sizes are forbidden.

## 4. Component Stylings

Every component anatomy below quotes `var(--token)` references against `packages/ui/src/tokens.css`. Hex literals are forbidden in this section — token names are stable and resolve through the canonical CSS variables. Historical hex values that no longer match the palette are deleted, not annotated.

### Buttons

`<Button>` ships 9 additive variants and 10 sizes. The 6 below cover the runtime ladder; `outline` / `secondary` / `link` are retained for legacy callsites only and forbidden in new code. CTA radius is `var(--radius)` (6 px) — **never pill-shaped CTAs.**

#### `default` / `primary` (accent CTA)

Solid accent fill. `primary` is an initial-parity alias for `default`.

| State    | Background            | Text                | Radius          | Height (default / lg) | Padding |
| -------- | --------------------- | ------------------- | --------------- | --------------------- | ------- |
| Default  | `var(--accent)`       | `var(--accent-ink)` | `var(--radius)` | 30px / 36px           | 0 12 px |
| Hover    | `var(--accent-hover)` | `var(--accent-ink)` | same            | same                  | same    |
| Active   | `var(--accent-hover)` | `var(--accent-ink)` | same            | same                  | same    |
| Disabled | `var(--disabled)`     | `var(--muted)`      | same            | same                  | same    |

Active rim emits `box-shadow: var(--highlight)`; never accent on hover-border.

#### `neutral` (filled secondary)

Background glaze. Use when `secondary` is too quiet and `default` is too loud.

| State    | Background                 | Text               | Border |
| -------- | -------------------------- | ------------------ | ------ |
| Default  | `var(--btn-default-fill)`  | `var(--fg-strong)` | none   |
| Hover    | `var(--btn-default-hover)` | `var(--fg-strong)` | none   |
| Disabled | `var(--disabled)`          | `var(--muted)`     | none   |

#### `ghost`

Text-only. Hover lifts via the row-hover glaze.

| State     | Background         | Text               | Height | Padding    |
| --------- | ------------------ | ------------------ | ------ | ---------- |
| Default   | transparent        | `var(--muted)`     | 26 px  | 0 12 px    |
| Hover     | `var(--row-hover)` | `var(--fg-strong)` | 26 px  | 0 12 px    |
| Icon Only | transparent        | `var(--muted)`     | 26 px  | 0 (square) |

#### `destructive` (danger)

Text-only by default; tint on hover. The variant name stays `destructive` (the AGH name for `danger`).

| State    | Background           | Text            | Border |
| -------- | -------------------- | --------------- | ------ |
| Default  | transparent          | `var(--danger)` | none   |
| Hover    | `var(--danger-tint)` | `var(--danger)` | none   |
| Disabled | `var(--disabled)`    | `var(--muted)`  | none   |

#### `success`

Text-only by default; tint on hover. Same anatomy as `destructive` with `--success` / `--success-tint`.

#### `icon`

Square 26 × 26 icon-only button. Background `var(--row-hover)` on hover; ghost otherwise. Stroke comes from `<Icon size>`.

Marketing CTAs use `cta` (36 px) and `cta-lg` (44 px) sizes with `var(--radius-md)` (8 px). Header nav pills (site only) keep `var(--radius-pill)` per §5 "Layout Principles".

### Pill / Status / Mono identifiers

`<Pill>` is the canonical runtime chip. Every pill ships **radius `var(--radius-xs)` (4 px) across sizes** — the per-size radius tiers (`--radius-chip`, `--radius-mono-badge`) are reserved for legacy / off-runtime consumers; `--radius-pill` is marketing nav only. The `uppercase` prop is gone; callers that need uppercase emit `className="uppercase"` themselves or compose `<Eyebrow>`.

#### `<Pill>` sizes + tones

| Size | Height | Padding | Type                        | Notes                            |
| ---- | ------ | ------- | --------------------------- | -------------------------------- |
| `xs` | 17 px  | 0 6 px  | Inter 10.5 / 510 / -0.005em | Inline tag chips, run-cell pills |
| `sm` | 19 px  | 0 7 px  | Inter 11 / 510 / -0.005em   | Default size, row pills          |
| `md` | 22 px  | 0 8 px  | Inter 12 / 510 / -0.005em   | Toolbar pills, detail-head pills |

Neutral pill is **borderless tint-only**: `bg-var(--neutral-tint)` / `text-var(--fg)`. Signal tones (`success | warning | danger | info | accent`) ride 8-10 % tint backgrounds with full-signal text — never solid signal fills.

#### `<Pill mono>` (identifier mono variant)

All mono pills emit **10.5 px / 600 / tracking 0** regardless of size. Use only for inline status / kind chips that carry a tone — for bare row identifiers reach for `<MonoId>`, never `<Pill mono>`.

#### Active toggle state

Active pill toggles drop their border in favor of `bg-var(--elevated)` + `text-var(--fg-strong)` + `box-shadow: var(--highlight)`. The legacy `solid-accent` variant is reserved for unread-count chips in channel/nav rows.

#### Kind Chip (wire protocol marker)

Wire-protocol kind marker (`say`, `greet`, `direct`, `receipt`, `recipe`, `trace`, `whois`).

- **Radius:** `var(--radius-xs)` (4 px) — pinned
- **Padding:** 1 px 6 px.
- **Type:** JetBrains Mono 9.5 px / 600 / uppercase / tracking 0.06em via `--tracking-mono`.
- **Surface:** `var(--canvas-soft)`, ring `box-shadow: 0 0 0 1px var(--line-soft)`. **Text:** `var(--muted)`.
- **Wire-dot prefix:** 7 × 7 circle whose color resolves from the kind palette (`say → --muted`, `greet → --info`, `direct → --accent`, `receipt → --success`, `recipe → --warning`, `trace → --info`, `whois → --info`).
- Unknown kinds (platform names, event ids) render the chrome without a dot.

#### ALPHA Chip (brand)

Sits next to the `agh` wordmark. Mono, uppercase, `--tracking-mono` (0.06em), 10 px. Transparent fill, 1 px `var(--line)` ring, `var(--muted)` text.

### `<PillGroup>` segmented selector

Header-level segmented selector (`ALL / GLOBAL / WORKSPACE`, `LIST / KANBAN / DASHBOARD / INBOX`, `JOBS / TRIGGERS`).

- **Track.** Borderless `bg-var(--canvas-soft)`, radius `var(--radius-md)` (8 px), padding 2 px, segment gap 1 px.
- **Segment.** Min-height 24 px (`md`) / 20 px (`sm`), padding 0 10 px, Inter sentence-case **12 px / 510 / -0.005em** (NOT mono, NOT uppercase). Default text `var(--muted)`; hover `var(--fg)`.
- **Active.** `bg-var(--elevated)` / `text-var(--fg-strong)` + `box-shadow: var(--highlight)`. **Never solid accent.**
- **Count badge.** 3 px radius literal, `bg-var(--badge-fill)`, `text-var(--muted)`, `tabular-nums`, sentence-case (NOT uppercase, NOT `var(--accent)`).

The chipped `default` variant of `<Tabs>` is deprecated in favor of `<PillGroup>`

### `<Tabs>` underline tabs

`<Tabs>` ships two variants. `default` (chipped) is deprecated.

#### `variant="line"`

- **Indicator.** `bg-var(--fg-strong)` 1.5 px underline (NOT `var(--accent)` 2 px).
- **Trigger.** 12 / 500 default → 12 / 510 active.
- **Active count chip.** `bg-[rgba(255,255,255,0.07)]` / `text-var(--fg)`. Never `var(--accent)`.

#### `variant="lane"`

Lane-tab pattern (used in `.page-head` analogs):

- **Separator.** `·` rendered between triggers via the primitive (CSS pseudo).
- **Count slot.** Bare mono 10.5 px `text-var(--faint)` inline next to the label — no chip wrapper.
- **Active indicator.** Same `var(--fg-strong)` 1.5 px underline.

### `<Empty>` empty state

- **Layout.** Top-padded `64 px 0 0` (NOT flex-centered).
- **Icon-well.** 38 × 38, radius `var(--radius-lg)` (10 px), `bg-var(--canvas-soft)`, no border, `text-var(--muted)`.
- **Title.** 18 / 510 / -0.022em via `--text-empty-h1` / `--tracking-empty-h1`.
- **Description.** Max-width 54ch, `text-var(--muted)`.
- **CTA.** `margin-top: 22 px`, `<Button variant="default">` (accent CTA).

### `<Metric>` / `<KpiCard>`

`<DashboardCard>` was hard-renamed to `<KpiCard>` with no alias.

#### `<Metric>` (compact, in-body)

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-lg)` (10 px), padding 14 / 16, gap 8 px, **no border**.
- **Label.** Sentence-case Inter **11.5 px / 510 / -0.005em** via `--text-form-hint`, `text-var(--muted)`. **Never `<Eyebrow>`** — see DESIGN.md §11 "Eyebrow misuse".
- **Value.** Inter 22 px / 510 / -0.014em via `--text-metric-value`, `text-var(--fg-strong)`.
- **Semantic values.** Tone-tinted text via `data-tone` (`positive → --success`, `negative → --danger`, `warning → --warning`).

#### `<KpiCard>` (dashboard KPI)

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-lg)` (10 px), padding 16 / 18, gap 10 px, **no border**.
- **Label.** Sentence-case Inter **12 px / 510 / -0.005em** via `--text-form-label`, `text-var(--muted)`.
- **Value.** Inter 28 px / 510 / -0.018em via `--text-kpi-value`, `text-var(--fg-strong)` — never tone-colored.
- **Subtext / sparkline.** Optional Inter 13 px `text-var(--muted)` line + inline `<QueueHealthSparkline>`.

### Feature Card (Marketing — `packages/site` only)

The canonical marketing card. Pattern: icon well → eyebrow → verb-forward title → mechanism sentence → optional mono source cite.

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-xl)` (14 px), `box-shadow: 0 0 0 1px var(--line-soft)`, padding 24 px.
- **Icon well.** 40 × 40, radius `var(--radius-icon-well)` (10 px), `bg-var(--elevated)`, accent-colored Lucide icon via `<Icon size="default">`.
- **Eyebrow.** `<Eyebrow>` (Inter UC 11 / 600 / 0.05em UC), `text-var(--muted)`.
- **Title.** Inter 20 / 510 / -0.014em, `text-var(--fg-strong)`, three-word verb-forward phrase.
- **Description.** Inter 14 / 400, `text-var(--muted)`, one-sentence mechanism.
- **Source cite (optional).** Mono, `text-var(--subtle)` + `ArrowUpRight` (12 px).
- **Hover.** Background lifts to `var(--elevated)`. **No lift, no scale, no accent border hover.**

### Form input

Form-grade `<Input>` / `<Textarea>` / `<Select>` used inside `<Field>` and `<FormSection>` composites.

- **Container.** `bg-var(--input-fill)` (≈ rgba(255,255,255,0.025)), radius `var(--radius-md)` (8 px), height 32 px, padding 0 11 px, gap 8 px. **No border at rest** — only `box-shadow: 0 0 0 1px transparent`.
- **Type.** Inter 12.5 px / 510 / -0.005em via `--text-form-input`, `text-var(--fg-strong)`.
- **Focus.** `bg-var(--canvas)` + `box-shadow: 0 0 0 1px var(--line-strong)` — never accent ring.
- **Placeholder.** `text-var(--subtle)`.
- **Icon (prefix).** Via `<Icon size="sm">` (12 px), `text-var(--muted)`.
- **Disabled.** `bg-var(--disabled)`, `text-var(--muted)`.
- **Inside `<FormSection>` modals.** Inputs MUST NOT override height through `h-9` / `h-10`.

#### `<Textarea variant="mono">`

Adds `font-family: var(--font-mono)` + 12 px size. Min-height 92 px, line-height 1.55. Same fill / focus contract as the default input.

#### Search / Filter Input (sidebar + panel)

Compact `SearchInput` row used inside sidebars and list panels.

- **Container.** `bg-var(--sidebar)`, radius `var(--radius-md)` (8 px), height 28 px, padding 0 8 px, gap 8 px.
- **Border.** `box-shadow: 0 0 0 1px var(--line-soft)` at rest → `box-shadow: 0 0 0 1px var(--line-strong)` on focus-within. **No accent ring.**
- **Text.** Inter 13 / 510 / -0.005em, `text-var(--fg-strong)`. Placeholder `text-var(--subtle)`.
- **Search icon.** Via `<Icon size="sm">` (12 px), `text-var(--muted)`.
- **Kbd hint (`⌘K`, `jump`, …).** JetBrains Mono 9 px uppercase via `--tracking-mono`, padding 1 / 4, radius `var(--radius-xs)` (4 px), `bg-var(--canvas-soft)`, `text-var(--subtle)`, ring `box-shadow: 0 0 0 1px var(--line-soft)`. Hidden on mobile.

#### Header Search Trigger (Site only)

Round-full search trigger: `bg-var(--canvas-soft)`, ring `box-shadow: 0 0 0 1px var(--line-soft)`, mono `⌘K` hint on the right, 36 px height. Round-full radius is the marketing-nav carve-out only.

### `<Dialog>` / Modal

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-lg)` (10 px), `box-shadow: var(--shadow-overlay)`.
- **Width ladder.** `--modal-w-sm` (560 px) / `--modal-w-md` (720 px, New Task default) / `--modal-w-lg` (880 px, Bridges wizard). Never inline `sm:max-w-*` literals.
- **Header.** Padding 13 / 18, title 13.5 / 510 / -0.012em via `--text-modal-title`, close button 26 × 26 ghost icon via `<Button variant="icon">`.
- **Body.** Padding 18 px. Composes `<FormSection>` blocks for editor surfaces; `<RadioCard>` grid for template pickers.
- **Footer.** Padding 11 / 18, `bg-var(--canvas-soft)`, left-aligned hint + right-aligned actions.
- **Scrim.** `bg-var(--overlay-scrim)` (alpha 0.55) + `backdrop-filter: blur(var(--overlay-blur))` (3 px). Every other surface stays blur-free.

### `<RadioCard>`

Used in modal template pickers, scope pickers, approval policy pickers, max-attempts.

- **Container.** Padding 9 / 11, radius `var(--radius)` (6 px), no border at rest.
- **Resting fill.** `bg-var(--canvas-soft)`.
- **Hover.** Lifts to `bg-var(--elevated)`.
- **Selected.** `bg-var(--surface-glaze)` + `box-shadow: inset 0 0 0 1px var(--line-strong)`. **Never** `border-(--accent)`, never `bg-(--accent-tint)`.
- **API.** `selected` / `onSelect` / `title` / `description` / `icon` / `badge`.
- **Use.** Option groups with descriptions (template, scope, approval). `<PillGroup>` is reserved for short numeric / enumerated selectors (priority, attempts).

### `<CatalogCard>`

Catalog grid card for templates, bridge providers, model providers, marketplace skills.

- **Container.** Padding 16 px, radius `var(--radius-lg)` (10 px), `bg-var(--canvas-soft)`, no border.
- **Hover.** `bg-var(--elevated)`.
- **Selected.** `bg-var(--surface-glaze)` + `box-shadow: inset 0 0 0 1px var(--line-strong)` — mirrors `<RadioCard>`, never accent.
- **Icon-well (`logoSize="default"`).** 24 × 24, radius `var(--radius)` (6 px), `bg-var(--surface-glaze)`, glyph `text-var(--muted)`. The `size-catalog-logo` token (24 px) pins this.
- **Icon-well (`logoSize="lg"`).** 40 × 40 (`--size-provider-logo-well`) — connected-provider chrome.
- **Title.** Inter 13 / 510 / -0.012em, `text-var(--fg-strong)`.
- **Tones on the logo.** `accent | neutral | success | warning | danger | info` color the glyph only — card chrome stays neutral.

### `<FormSection>`

Tinted form-section block used inside `<TaskEditorModal>`, settings editor forms, and any other editable surface. Replaces `<Section>` inside modals.

- **Container.** Padding 18 / 20 (`size="comfortable"`) or 14 / 14 (`size="compact"`), radius `var(--radius-lg)` (10 px), `bg-var(--canvas-soft)`, **no border**.
- **Head.** 16 px margin-bottom. Composed of: optional leading icon (`<Icon size="default">`, `text-var(--subtle)`) + title (13 / 510 / -0.008em via `--text-section-head` + `--tracking-section-head`, `text-var(--fg-strong)`) + right-aligned 11 px sub eyebrow slot.
- **Row.** Flex column, gap 6 px, `margin-top: 14 px` between rows. Each row head baseline-aligned: label + optional required glyph + optional inline hint.
- **Children.** Form controls (`<Input>`, `<Textarea>`, `<Select>`, `<RadioCard>`, `<Filters>` fields). Form-row hints sit inline with the label — never below the input.
- **Use.** Inside modals and editor surfaces. `<Section>` (13 px head, opt-in border) is reserved for in-body content grouping.

### `<ContextBox>` (read-only context strip)

Used inside `<TaskEditorModal>` and detail surfaces to render parent / dependency context as a key-value strip.

- **Container.** Padding 13 / 14, radius `var(--radius)` (6 px), `bg-var(--input-fill)`, no border.
- **Layout.** 2-column grid, gap 10 / 22.
- **Label.** Mono UC 10.5 px / 500 / 0.04em via `--tracking-mono-meta`, `text-var(--faint)` (eyebrow utility).
- **Value.** Inter 12 px / 510 / -0.005em, `text-var(--fg-strong)`. Identifier values render through `<MonoId>` — never `<Pill mono>`.

### `<RunCard>`

Active-run renderer used in `tasks-detail-overview-panel`, `/tasks/$id/runs/$runId`, and the agent-card timeline strip.

- **Container.** Padding 14 / 16, radius `var(--radius-lg)` (10 px), `bg-var(--canvas-soft)`, **no border, no `border-l-* border-l-(--accent)` rail**.
- **Top row (pill stack).** Status `<Pill>` + `<MonoId>` run-id + session info pill (`tone="info"`) + ghost mono attempt pill + optional warning pill tinted via `var(--warning-tint)` / `var(--danger-tint)`.
- **Body (stat grid).** `grid-cols-4`, gap 14 px, `margin-top: 14 px`, `border-top: 1px solid var(--line-soft)`, `padding-top: 14 px`. Cells: `CHANNEL` / `QUEUED` / `STARTED` / `ELAPSED`.
- **Cell value.** `<Time mode="relative">` for `QUEUED` / `STARTED`; `formatDuration` from `@agh/ui` for `ELAPSED`.
- **Status enum.** `pending | in_progress | completed | failed | canceled` mapped via `RUN_STATUS_TONE` to `neutral | info | success | danger | neutral`.

### `<DetailHeader>`

Replaces every inlined detail-header copy. Renders the 6-row anatomy and owns the back slot.

- **Row 1 — Crumbs.** Optional `DetailHeaderCrumb[]` (`·` separator) or `ReactNode`. Inter 11.5 / 510 / -0.005em, `text-var(--faint)`.
- **Row 2 — Pre-title.** Optional eyebrow / kicker line via `<Eyebrow>` (Inter UC 11 / 600 / 0.05em UC).
- **Row 3 — H1.** Inter **24 / 510 / -0.028em** via `--text-detail-h1` + `--tracking-detail-h1`, `text-var(--fg-strong)`. **The only body-side H1 in the runtime.**
- **Row 4 — Pills.** Status / lane / owner pill row. `<MonoId>` for the entity id (NOT `<Pill mono>`).
- **Row 5 — Meta.** Mono UC tags via `<Eyebrow>` + separators.
- **Row 6 — Actions.** Trailing buttons. Primary CTA is the only accent surface in this row.
- **Back slot.** `back?: () => void` mounts a 20 × 20 ghost chevron next to the title; default `backLabel = "Go back"`. Behaviour resolves through `router.history.back` with parent-route fallback.
- **No `<PageHead>` primitive.** Page identity for non-detail surfaces lives in the global `<Topbar>` (14 px route title + count chip + meta slots).

### Other runtime cards & containers

#### `<Card>` generic

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-lg)` (10 px), padding 16 / 20, **no border** (the `ring-1 ring-(--line)` was removed).
- Used as the base for runtime feature cards, action cards, etc.

#### `<WireCard>` (chat / network protocol)

- **Shell.** `bg-var(--canvas-soft)`, radius `var(--radius)` (6 px), `box-shadow: 0 0 0 1px var(--line-soft)`, max-width 520 px.
- **Head (`WireCardHead`).** `bg-var(--rail)`, `border-bottom: 1px solid var(--line)`, padding 6 / 10, JetBrains Mono 10.5 / 600 / 0.06em UC via `--tracking-mono`, `text-var(--muted)`.
- **Body (`WireCardBody`).** Padding 8 / 12, JetBrains Mono 11.
- **Foot (`WireCardFoot`).** `bg-var(--rail)`, `border-top: 1px solid var(--line)`, padding 6 / 10, ghost action buttons.
- **Inline variant.** Single-line strip, padding 6 / 10, gap 8.

#### Code Block

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-lg)` (10 px), padding 16-20 px.
- **Font.** JetBrains Mono, 13-14 px, line-height 1.6.
- **Prompt.** `$ ` in `var(--accent)`, command body `text-var(--fg-strong)`.
- **Copy button.** `<Button variant="icon">`, absolute top-right, `text-var(--subtle)` → `var(--fg-strong)` on hover. Checkmark swap for 1.5 s on copy success.

#### Metadata Table (`<MetadataList>`)

- **Odd rows.** Transparent.
- **Even rows.** `bg-var(--canvas-soft)`.
- **Key.** Inter 13 / 400, `text-var(--muted)`.
- **Value.** Inter 14 / 510, `text-var(--fg-strong)`. Identifier values render through `<MonoId>`.

#### Comparison Highlighted Row (marketing only)

The ONE "colored left border" pattern in the system — and reserved for marketing comparison tables.

- **Border-left.** 4 px solid `var(--accent)`.
- **Background.** `var(--accent-tint)`.
- **Do not proliferate** to runtime lists — see DESIGN.md §11 "Side-stripe accent rail" for the ban on every other side-stripe.

### `<DescriptionCard>`

Streamdown-rendered markdown card. Used for `task.description`, agent descriptions, knowledge bodies.

- **Container.** `bg-var(--canvas-soft)`, radius `var(--radius-lg)` (10 px), padding 16 / 18.
- **Prose.** Line-height 1.7, max-width 72ch. Inline code via `var(--font-mono)`. Headings (`h1`-`h6`), links, code, kbd, strong, em, mark, blockquote ride canonical HTML and resolve through `var(--canvas-soft)` / `var(--surface-glaze)` / `var(--line)` / `var(--text-*)` tokens.
- **Sanitization.** `STREAMDOWN_SAFE_CONFIG` composes five defenses (`skipHtml`, 13-element `disallowedElements`, URL-scheme allowlist via `defaultUrlTransform`, no streamdown chrome, canonical HTML overrides + `SafeImage` external-URL fallback). The XSS regression corpus in `description-card.test.tsx` is the security gate — do NOT relax without an ADR.

### `<ChatToolCard>`

Inline card showing tool execution inside chat threads.

- **Container.** Padding 10 / 16, radius `var(--radius-md)` (8 px), `bg-var(--canvas-soft)`, no border.
- **Head.** `<MonoId>` for the tool id + status `<Pill>` (`pending → neutral`, `in_progress → info`, `success → success`, `failed → danger`) + optional `<Time>` + actions slot.
- **Sections.** Collapsible Input / Output regions (chevron `<button aria-expanded>` toggle). String outputs above `CHAT_TOOL_OUTPUT_COLLAPSE_LINES = 200` default-collapse.
- **Failed state.** Tints the container with `var(--danger-tint)` (success / in_progress paint `var(--canvas-soft)`).

### Chat Components

#### User Message

Right-aligned bubble.

- **Bubble.** `bg-var(--elevated)`, radius `var(--radius-lg)` (10 px), padding 16 / 20.
- **Text.** Inter 14 / 400, `text-var(--fg-strong)`.
- **Meta.** "YOU" + `<Time>`, `<Eyebrow>` (Inter UC 11 / 600 / 0.05em UC), `text-var(--muted)`, right-aligned above the bubble.

#### Agent Message

Left-aligned, no bubble.

- **Agent label.** `<OwnerAvatar size="sm">` + agent name (`<Eyebrow>`, `text-var(--fg)`) + `<Time>`.
- **Text.** Inter 14 / 400, `text-var(--muted)`.

#### Typing Dots

Three-dot typing indicator paired with `<peer> is typing…` copy.

- 3 × 4 × 4 dots, gap 2 px, radius 50 %, `bg-var(--muted)`.
- Animation: `typing-bounce` 1.2 s infinite ease-in-out, with 0 / 0.15 / 0.3 s stagger.
- Container copy: `<Eyebrow>` 11 px / `text-var(--muted)`.

#### Chat Input (composer)

- **Container.** `bg-var(--input-fill)`, radius `var(--radius-lg)` (10 px), padding 12 / 16, no border at rest.
- **Focus.** `bg-var(--canvas)` + `box-shadow: 0 0 0 1px var(--line-strong)` (never accent ring).
- **Placeholder.** Inter 14 / 400, `text-var(--subtle)`.
- **Send button.** 36 px square (`<Button variant="default">`), `bg-var(--accent)`, white send icon via `<Icon size="sm">`.

### `<OwnerAvatar>`

Owner-identity avatar used by kanban cards, task rows, agent cards, timeline events, run cards, chat threads.

- **Sizes.** `sm` (20 × 20) / `default` (24 × 24) / `lg` (32 × 32).
- **Background / foreground.** Resolves through `colorsFor(ownerKind, ownerId)` against the tokenised owner palette — `var(--avatar-agent-{0..3}-{bg,fg})` (4 slots), `var(--avatar-human-{0..2}-{bg,fg})` (3 slots), `var(--avatar-system-{bg,fg})`. Slot selection is deterministic via FNV-1a hash mod slot count.
- **Glyph.** 2-character monogram by default; optional `glyph` slot for system owners or branded icons.
- **A11y.** Emits `role="img"` + `aria-label="{Role} {Name}"` automatically.
- **Live state.** Optional `Pill(accent, pulsing) live` chip rendered alongside in agent cards uses the `pill-pulse` keyframe.

### `<StatusDot>` indicator

Single-LED status indicator used by `<RuntimeConnectionIndicator>` and inline status surfaces.

| Tone    | Solid            | Ring                       |
| ------- | ---------------- | -------------------------- |
| accent  | `var(--accent)`  | `var(--accent-tint)` ring  |
| success | `var(--success)` | `var(--success-tint)` ring |
| warning | `var(--warning)` | `var(--warning-tint)` ring |
| danger  | `var(--danger)`  | `var(--danger-tint)` ring  |
| faint   | `var(--faint)`   | `var(--line)` ring         |

- **Sizes.** 6 px (default) / 5 px (`sm`). Wrap in a larger clickable area when interactive.
- **A11y.** Defaults to `aria-hidden="true"` unless a `label` prop is provided.
- **Daemon LED states.** `connected + !degraded → success solid`; `connected + degraded → success pulse`; `connecting → success pulse`; `disconnected | error → danger solid`.

### Sidebar (Operator UI)

#### Structure

- **Workspace rail.** Width `var(--width-rail)` (56 px). Cells render at 30 × 30 with `var(--radius-md)` (8 px) squircle radius.
  - **App logo.** `bg-var(--accent)`, `text-var(--accent-ink)` letter A, active rim `box-shadow: var(--highlight)`.
- **Active workspace.** 2 × 16 px `bg-var(--fg-strong)` nub anchored at the inside edge — never `var(--accent)`.
- **Inactive.** `bg-var(--elevated)`, `text-var(--muted)` letter.
- **Hover.** `bg-var(--row-hover)`.
- **New.** `bg-var(--elevated)`, ring `box-shadow: 0 0 0 1px var(--line-soft)` dashed, `+` icon via `<Icon size="default">`.
- **Sidebar panel.** `bg-var(--sidebar)`, width 244 px default / 220 px ≤ 1100 px / drawer overlay ≤ 880 px.

#### Section Header (`<SidebarSectionLabel>`)

Inter UC 10.5 / 510 / 0.05em via `--text-section-label` + `--tracking-section-label`, `text-var(--muted)`. Padding 12 / 12 / 6 / 12. Same primitive used for `AGENTS`, `WORKSPACE`, `STARRED`, `CHANNELS`, `DIRECT MESSAGES`, panel-internal subheaders. Sidebar group labels use the **Inter UC family** (NOT mono UC)

The AGENTS section label renders `{live}/{total} live` whole-tree count via `computeAgentsCount(agents, sessions)`; the count slot hides when `total === 0`.

#### Nav Row (top-level + channel rows)

Flat row, no border or card chrome.

- **Row.** Padding 6 / 8, radius `var(--radius)` (6 px), gap 8 px (top-level) / 2.5 (channel rows).
- **Icon.** Via `<Icon size="sm">` (12 px) or `default` (14 px), `text-var(--muted)` default → `text-var(--fg-strong)` when active.
- **Label.** Inter 13 (top-level) / JetBrains Mono 12 (channel rows). Default `text-var(--muted)`, active `text-var(--fg-strong)` 510. Unread channel rows render the label `text-var(--fg-strong)` 600.
- **Hover.** `bg-var(--row-hover)`.
- **Active.** `bg-var(--row-selected)` **plus** a 2 px-wide indicator rail rendered via `ACTIVE_NAV_INDICATOR_CLASS` (`bg-var(--fg-strong)`) anchored against the panel edge. **Never accent.**
- **Unread badge.** `<Pill mono>` with `solid-accent` variant + count (the legacy carve-out — accent text-on-tint).

### Topbar (Operator UI)

48 px shell row + 12 px gap + 22 px inline padding. Topbar slot vocabulary: `{ title, trailing, count, back, backLabel, meta, overflow }`.

- **Container.** Height 48 px, `bg-var(--canvas)`, `border-bottom: 1px solid var(--line-soft)`.
- **Leading icon.** 22 × 22, `text-var(--muted)`, no chip, no accent tint.
- **Title.** Inter 14 / 510 / -0.014em via `--tracking-tight`, `text-var(--fg-strong)`.
- **Count chip.** Inter 11 px / 510 / tabular-nums, `bg-var(--badge-fill)` / `text-var(--faint)`, radius `var(--radius-mono-badge)` (4 px after the token retune) — never accent.
- **Separator.** `·` rendered via `.topbar__sep` (mono 10.5 / `text-var(--faint)`).
- **Detail-mode topbar.** Swaps to `back + title + id-chip + sep + meta + spacer + actions + overflow`. Back resolves through `router.history.back` with parent-route fallback.
- **Primary CTA.** "New {noun}" sentence-case via `<Button variant="default">`. Never `variant="outline"`.

### Site Header (Marketing + Docs — site only)

- **Shell.** Sticky top, `bg-color-mix(in srgb, var(--canvas) 92%, transparent)` + `backdrop-blur-xl`, `border-bottom: 1px solid var(--line)`. The marketing site keeps its own blur via its own stack
- **Wordmark.** NuixyberNext "agh" + ALPHA chip (mono 10 px, `var(--line)` ring).
- **Nav pills.** Round-full (marketing carve-out), hover + active tint `var(--accent-tint)`.
- **Search trigger.** Round-full, 36 px, mono `⌘K` hint.
- **GH button.** Round-full ghost with GitHub logo via `<Icon size="default">`.

### Docs Masthead

- **Eyebrow.** `<Eyebrow>` (JetBrains Mono 12 / 600 / 0.06em UC via `--tracking-mono`), `text-var(--muted)`.
- **Title.** Inter 510, clamp ramp (see DESIGN.md §13 site type clamps).
- **Sub-lead.** Inter 18 / 400, `text-var(--muted)`, max-width 58ch.

## 5. Layout Principles

### Spacing Scale (Base: 4px)

No formal token scale — use Tailwind defaults. The working grid is `4 / 8 / 12 / 16 / 24 / 32 / 48 / 64`.

| Token        | Value | Usage                                           |
| ------------ | ----- | ----------------------------------------------- |
| **space-1**  | 4px   | Icon-label gap, tight inner padding             |
| **space-2**  | 8px   | Tag padding, inline element gap, badge internal |
| **space-3**  | 12px  | List row padding, stacked element gap           |
| **space-4**  | 16px  | Standard card padding, section gap              |
| **space-5**  | 20px  | List item gap, card inline padding              |
| **space-6**  | 24px  | Card outer padding, feature card padding        |
| **space-8**  | 32px  | Section-level spacing                           |
| **space-12** | 48px  | Major section separation                        |
| **space-16** | 64px  | Hero / landing section breathing room           |

Section vertical padding on marketing is set by a `SectionFrame` `padY` prop (`md` / `lg` / `xl`).

### Border Radius Scale

The runtime kit ships a single radii ladder: **4 / 5 / 6 / 8 / 10 / 14 / pill**. There is no separate `--radius-diagram` literal — `--radius-lg` is the canonical 10 px card / panel radius.

| Token           | Value   | Usage                                               |
| --------------- | ------- | --------------------------------------------------- |
| `--radius-xs`   | 4 px    | Tightest chip rim                                   |
| `--radius-sm`   | 5 px    | Kind chip                                           |
| `--radius`      | 6 px    | Default control radius (status badges, mono badges) |
| `--radius-md`   | 8 px    | Inputs, buttons, avatars                            |
| `--radius-lg`   | 10 px   | Cards, panels, popovers, modals                     |
| `--radius-xl`   | 14 px   | Sheet shell, hero card, large surface               |
| `--radius-pill` | 9999 px | Search trigger, header nav pills, status dots       |

Aliases retained for ergonomic component usage: `--radius-chip` (5 px), `--radius-mono-badge` (6 px), `--radius-icon-well` (10 px).

CTAs use `--radius-md` (8 px). **Never pill-shaped CTAs.**

### Elevation (Flat Depth Model)

No box-shadows. Depth is communicated purely through background lightness + 1px dividers.

| Level | Name            | Background                   | Shadow               | Usage                               |
| ----- | --------------- | ---------------------------- | -------------------- | ----------------------------------- |
| -1    | **Canvas Deep** | `#0E0E0F`                    | none                 | Code blocks, deep landing sections  |
| 0     | **Canvas**      | `#141312`                    | none                 | Page background                     |
| 1     | **Surface**     | `#1E1C1B`                    | optional `ring-1/10` | Cards, sidebar, panels, modals      |
| 2     | **Elevated**    | `#2E2C2B`                    | `shadow-xs` (subtle) | Popovers, search inputs, icon wells |
| —     | **Scrim**       | `rgba(0,0,0,0.5)`            | —                    | Modal/dialog backdrop               |
| —     | **Sticky Blur** | `rgba(20,19,18,0.92)` + blur | —                    | Header only                         |

### Grid & Layout

- **Site layout width:** `--site-layout-width: 1200px` for landing content. Centered, generous gutters.
- **Docs layout:** `--site-doc-layout-width: 96rem` with `--site-doc-sidebar-width: 16rem` (left tree) + `--site-doc-toc-width: 14rem` (right TOC).
- **Hero grid:** `grid-cols-[minmax(0,1fr)_minmax(0,540px)]` — copy-left / visual-right, reversed on mobile.
- **Sidebar (Operator UI):** workspace rail (40px) + panel (~220px) = ~260px total.
- **Content area:** flex-1, left-aligned.
- **Three-panel layout (Skills, Knowledge):** sidebar + list panel + detail panel.
- **Metric grids:** 4-column row of equal-width cards with 16px gap.
- **List layouts:** full-width rows with consistent padding, 1px border-bottom separators.
- **Landing sections:** alternate `canvas` / `surface` / `deep` backgrounds for rhythm.

### Whitespace Philosophy

- **Flat hierarchy, clear grouping** — sections separated by 48–64px on marketing, 24–32px in UI. Items within lists by 8–12px.
- **Left-aligned, not centered** — page titles, list items, descriptions all start from the same left edge (except empty states and hero CTAs).
- **Dense but not crowded** — 36px interactive heights, 22px badge heights, 8px inline gaps. Marketing gets more breathing room (24px card padding, 48px+ section gaps).
- **Max reading widths** — lead copy 58ch, docs body 72ch, UI body 62ch.

## 6. Depth & Elevation

AGH uses a **flat depth model** — depth comes from the warm surface ramp + 1 px translucent hairlines. The two-token shadow vocabulary is defined by the two-token shadow vocabulary; every other surface stays flat.

### Surface ramp

```
--rail (#0c0b0b) → --canvas (#131211) → --canvas-soft (#1a1918)
                → --canvas-tint (#1c1b1a) → --elevated (#232220)
```

Each step is a small, deliberate lightness increase. Hairlines (`--line` / `--line-soft` / `--line-strong`) carry the rest of the separation.

### Whitelisted shadows

The shadow vocabulary is **exactly two tokens**. Introducing a third (`--shadow-card`, `--shadow-pop`, or any sibling) is forbidden and §11 "Anti-patterns".

| Token              | Value                                                                         | Allowed on                                                                             |
| ------------------ | ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `--shadow-overlay` | `0 24px 48px -12px rgba(0, 0, 0, 0.65), 0 0 0 1px rgba(255, 255, 255, 0.045)` | `dialog`, `confirm-dialog`, `sheet`                                                    |
| `--highlight`      | `inset 0 1px 0 rgba(255, 255, 255, 0.035)`                                    | `button --primary`, active `pill-group` segment, active filter `pill`, rail logo plate |

Every other surface stays flat. Popovers, dropdowns, tooltips, and command menus use `box-shadow: 0 0 0 1px var(--line-soft)` plus `bg: var(--canvas-soft)` — the 1 px ring carries the separation. Kanban cards stay flat with the same inset ring; the 1 px black baseline currently inlined on `<Button>` remains a literal next to `var(--highlight)` (no third token).

### Depth Patterns

| Pattern                     | How it works                                                                                                                        |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| **Card on canvas**          | `--canvas-soft` on `--canvas` — 1 px `--line` ring carries the edge                                                                 |
| **Nested card / icon well** | `--elevated` inside `--canvas-soft` — e.g. search input, icon well                                                                  |
| **Selected list item**      | `--elevated` + 2 px `--fg-strong` indicator rail. Side-stripe `border-l-* border-l-(--accent)` is banned — see §11 "Anti-patterns". |
| **Hover state**             | `--hover` replaces the current surface fill                                                                                         |
| **Divider**                 | 1 px solid `--line` between rows; `--line-soft` for softer subgroup splits                                                          |
| **Focus ring**              | `box-shadow: 0 0 0 1px var(--line-strong)` — white, never accent                                                                    |
| **Floating overlay**        | `box-shadow: var(--shadow-overlay)` on dialog / sheet only                                                                          |
| **Active rim**              | `box-shadow: var(--highlight)` on primary button, active pill segment, rail logo plate                                              |

No ambient shadows, no `shadow-md` / `shadow-lg`, no glows. The styles regression test rejects every Tailwind shadow utility outside the two whitelisted resolutions and grep-bans `--shadow-card` / `--shadow-pop` from the runtime contract.

## 7. Voice & Content

AGH copy has a specific, operator-first voice. Design without the voice is incomplete.

This section is the visual-system summary for copy that appears inside designed surfaces. For the canonical product-language contract - positioning, claims, evidence, vocabulary, CTA patterns, docs prose, release copy, package metadata, UI microcopy, and agent prompt guidance - read [`COPY.md`](COPY.md). `DESIGN.md` governs visual grammar; `COPY.md` governs verbal/product grammar.

### Tone

- **Operator-first, engineer-to-engineer, dry-confident.** Assume the reader knows what an agent runtime is.
- **Nouns over adjectives.** "Everything logged, everything replayable." "One operator surface."
- **No hype, no exclamations, no emoji. Anywhere.** Not in UI, not in docs, not in marketing.
- **"You" sparingly.** Most copy is imperative or third-person about the product: "Install the runtime." "The runtime lives on your machine."
- **Never "we" or "our" in marketing body.** Product is the subject: "AGH does X", "AGH Network gives agents Y".
- **Honest constraints, shown not hidden:** "macOS and Linux today", "Alpha" chip on the wordmark, "8 ACP CLIs".

### Casing

- **Body:** sentence case. "Resume any agent run", "Context that survives restarts".
- **Eyebrows:** UPPERCASE, mono, `tracking: 0.06em`. Examples: `SESSIONS`, `MEMORY`, `WHAT YOU GET`, `GETTING STARTED`.
- **Product names:** `AGH`, `AGH Runtime`, `AGH Network` — title case.
- **Protocol name:** `agh-network/v0` — lowercase mono, always.
- **Brand wordmark:** `agh` — all lowercase, NuixyberNext.

### Structural Patterns

- **Eyebrow + big title + short lead + visual** — the canonical section shell.
- **Three-word card titles, verb-forward** — "Resume any agent run", "Reusable playbooks", "Per-project everything".
- **Feature cards pair an eyebrow (concept) + a verb-forward title (benefit) + a one-sentence mechanism (proof).**
- **Docs H2s carry a top border** — they announce a new section on a dense page.

### Typographic Marks

- **Em-dash `—`** for copy pauses: "Everything logged — everything replayable."
- **Middle dot `· `** as meta separator: "macOS · recommended".
- **Arrow:** Lucide `ArrowUpRight` as the "continue reading" / "source link" indicator. Never `→` as glyph.
- **Shell prompt `$ `** in accent orange inside code blocks. The only glyph marker used.

### Imagery

- **No photography.** The landing uses SVG diagrams (`NetworkProtocolVisual`, `RuntimeMicroDiagram`, `ArchitectureDiagram`) made of lines + chips + mono labels.
- **If imagery is ever added**, it must be warm, desaturated, grainy, dark-ground — never bright, bluish, or cool.
- **Hero mesh is the only background texture.** Single `/hero-bg.png` at 20% opacity with `mix-blend-screen`. Not a purple gradient, not a blue gradient — a warm near-black mesh.

## 8. Iconography

### Primary System: Lucide React

Every icon comes from `lucide-react`. Standard set used: `Check`, `Minus`, `ArrowUpRight`, `Activity`, `Boxes`, `Database`, `FileCode2`, `Network`, `Plug`, `Sparkles`, `Timer`, `Star`, `Copy`.

### Stroke & Style

- **Stroke weight:** 2 (Lucide default). **Never** filled, duotone, or multi-color variants.
- **Stroke color:** accent `#E8572A` inside a card icon well; otherwise `currentColor` inheriting from text role (secondary / tertiary).

### Sizes

- `size-3` / `h-3 w-3` (12px) — inline mono chips, copy buttons, source-cite `ArrowUpRight`.
- `h-3.5 w-3.5` (14px) — check / minus cells in comparison table.
- `h-4 w-4` (16px) — default, inside 40×40 wells.
- `h-5 w-5` (20px) — rare; hero dismiss, nav glyphs.
- `h-12 w-12` (48px) — empty-state illustrations only.

### Logos

- **Partner / agent logos** (Claude, OpenAI, Gemini, GitHub, Cursor, OpenCode, Kiro, Pi, Slack, Telegram, Discord, Linear, Microsoft Teams, WhatsApp, Google Chat) live in `packages/ui/src/logos/*.tsx` as custom SVGs and are imported via the `@agh/ui/logos` subpath. **Do not regenerate from scratch.**
- **Brand wordmark:** NuixyberNext `agh` + adjacent `ALPHA` chip. Only place NuixyberNext appears.

### Glyph Rules

- **No emoji.** Anywhere.
- **`$ `** is the only shell-prompt marker.
- **Em-dash `—`** is the only typographic flourish in copy.

## 9. Motion & Animation

### Principles

- **Minimal, purposeful motion.** Transitions serve state changes, not decoration.
- **One fast tier (`--dur` 140ms) + one slow tier (`--dur-slow` 200ms).** Same `--ease` (`cubic-bezier(0.2, 0, 0, 1)`) everywhere.
- **No entrance animations.** Content appears immediately; no staggered reveals, no scroll-triggered fades.
- **No bounce / spring easing.** `--ease` (warm ease-out) is the default; `--ease-in-out` only for symmetric panel transitions.
- **Reduced motion is respected globally** via a universal-selector guard (see below).

### Tokens

| Token           | Value                          | Notes                                  |
| --------------- | ------------------------------ | -------------------------------------- |
| `--dur`         | `140ms`                        | Default duration                       |
| `--dur-slow`    | `200ms`                        | Sidebar / sheet / panel transitions    |
| `--ease`        | `cubic-bezier(0.2, 0, 0, 1)`   | Default ease (warm ease-out)           |
| `--ease-in-out` | `cubic-bezier(0.4, 0, 0.2, 1)` | Symmetric easing for opens/closes only |

### Transitions

| Element          | Property                 | Duration     | Easing          |
| ---------------- | ------------------------ | ------------ | --------------- |
| Button hover     | background-color         | `--dur`      | `--ease`        |
| Link hover       | color                    | `--dur`      | `--ease`        |
| Border focus     | border-color, box-shadow | `--dur`      | `--ease`        |
| List item hover  | background-color         | `--dur`      | `--ease`        |
| Sidebar collapse | width                    | `--dur-slow` | `--ease-in-out` |
| Modal enter      | opacity                  | `--dur-slow` | `--ease`        |
| Tooltip          | opacity                  | `--dur`      | `--ease`        |

No `transform` transitions on hover. No scale, no lift, no translate on card / button hover.

### Reduced motion (PRD F4 + M5)

`tokens.css` carries a universal-selector reduced-motion guard that zeros every animation and transition — not only those that read `--dur*`. This is required because `tw-animate-css` utilities and the `shimmer` / `typing-bounce` keyframes hardcode their durations and would otherwise keep playing. the duration is `0.001ms` (not `0`) so CSS `animationend` callbacks still fire for libraries that rely on them.

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.001ms !important;
    animation-delay: 0.001ms !important;
    transition-duration: 0.001ms !important;
  }
}
```

The `web/src/__tests__/styles.test.ts` regression test asserts the guard exists, applies to `*, *::before, *::after`, and pins each of the three declarations at `0.001ms !important`.

### Loading States

- **Shimmer:** `@utility animate-shimmer` — 200% background shift, 2s, infinite. For skeleton loaders.
- **Spinner:** rotating icon for inline loading.
- **Pulse:** `animate-ping` on a 1.5 px success dot — signals live state. Used sparingly.

## 10. Do's and Don'ts

### Do

- **Use the warm surface ramp** — `--rail` → `--canvas` → `--canvas-soft` → `--canvas-tint` → `--elevated` for hierarchy. No gradients on content. No stacked shadows.
- **Use warm values.** Every gray is tinted warm. Never substitute cool iOS grays.
- **Use JetBrains Mono for run-cell mono labels and protocol strings only.** Sidebar labels, table heads, and Eyebrow `case="upper"` go uppercase; everywhere else the runtime kit is sentence case.
- **Use Inter Variable for everything readable in the runtime kit** — Inter-510 ladder for titles + buttons + pill default; weight 400 for body. Playfair Display and NuixyberNext are `packages/site` only.
- **Use signal color consistently** — accent = action, `--success` = stable, `--danger` = stop, `--warning` = caution, `--info` = informational, `--neutral` = idle.
- **Use the 6–10 % tint formula** — `--success-tint` (8 %), `--warning-tint` (8 %), `--danger-tint` (9 %), `--info-tint` (7 %), `--neutral-tint` (6 %), `--accent-tint` (10 %). Full-color text rides on the tint.
- **Use the radii ladder** — 4 / 5 / 6 / 8 / 10 / 14 / pill. Cards land on `--radius-lg` (10 px); buttons / inputs land on `--radius-md` (8 px); pills on `--radius-pill`.
- **Use translucent hairlines** — `--line` for the universal divider, `--line-soft` for popover ring + group bottoms, `--line-strong` for focus rings. Focus is **white**, never accent.
- **Whitelisted shadows only** — `--shadow-overlay` (modals / sheets) and `--highlight` (active rim). Everything else is flat with a 1 px ring on `--line-soft`.
- **Left-align content** — page titles, list items, descriptions start from the same left edge.
- **Use colored signal values** — positive `--success`, negative `--danger`, neutral `--fg`. The color IS the information.
- **Respect reduced motion** — the universal-selector guard already zeros durations; never override `!important`.
- **Route identity lives in the global `<Topbar>`** — 14 px route title + count chip + meta slots. Do NOT add a body-side 22 px H1 as a duplicate. Detail surfaces are the only exception — they emit a 24 px `<DetailHeader>` hero in addition to the topbar swap.

### Don't

- **Don't use Tailwind shadow utilities outside the whitelist.** `shadow-md`, `shadow-lg`, `shadow-xl`, `shadow-2xl`, `shadow-xs`, `shadow-sm`, `shadow-inner`, `shadow-none` are all banned by `web/src/__tests__/styles.test.ts`. Use `box-shadow: var(--shadow-overlay)` or `var(--highlight)` only on the components named in §6.
- **Don't use gradients** — no content gradients, no gradient text, no gradient borders. The marketing site's hero mesh PNG is the single exception inside `packages/site`.
- **Don't use `backdrop-blur` or `mix-blend-*`** anywhere in the runtime kit (`web/`, `packages/ui`). The marketing site's sticky header keeps its own blur via its own stack.
- **Don't use decorative texture** — no hatching, no grain, no noise overlays.
- **Don't use emoji. Anywhere.** Not in UI, docs, marketing, commits, or copy samples.
- **Don't switch fonts.** Inter Variable for readable content; JetBrains Mono for metadata. Playfair Display + NuixyberNext are `packages/site` only. No Bricolage, no Geist, no alternative display face.
- **Don't add new signal colors.** The five-signal system (accent, success, warning, danger, info) plus `--neutral` is complete.
- **Don't use large decorative icons** — icons are 12–20 px inline. 48 px is reserved for empty-state illustrations.
- **Don't make badges with borders** — status badges and kind chips use tinted backgrounds only. Neutral / mono badges use the 1 px hairline.
- **Don't use pure white (`#FFFFFF`) for backgrounds** — reserved only for button text on accent fill.
- **Don't use `rounded-full` for CTAs.** CTAs are `--radius-md` (8 px). Only header nav pills, search triggers, and status dots use `--radius-pill`.
- **Don't paint a side-stripe accent rail on cards or rows.** `border-l-* border-l-(--accent)` (and `border-l-accent`) is banned. Active rows use the `--fg-strong` indicator rail; accent stays reserved for primary CTAs and the active rail logo. Enforced by `compozy-design-system/no-side-stripe-accent`. See §11 "Anti-patterns".
- **Don't inline glaze rgba literals.** `bg-[rgba(255,255,255,0.0NN)]` is forbidden — resolve through `--row-hover` / `--row-selected` / `--surface-glaze` / `--bar-fill` / `--input-fill` / `--btn-default-fill` / `--btn-default-hover` / `--badge-fill`. Enforced by `compozy-design-system/no-design-glaze-rgba`.
- **Don't introduce `--shadow-card` or `--shadow-pop` to the runtime kit.** The two-token shadow vocabulary in §6 is exhaustive. Cards stay flat with `box-shadow: 0 0 0 1px var(--line-soft)`.
- **Don't apply `backdrop-filter: blur(...)` outside `.dialog-scrim` / `.sheet-scrim`.** Modal and sheet scrims resolve `var(--overlay-blur)` (3 px) on top of `var(--overlay-scrim)` (alpha 0.55); every other runtime surface stays blur-free. The marketing site's sticky header keeps its own blur via its own stack.
- **Don't trigger accent on hover.** `hover:border-(--accent)`, `hover:bg-(--accent)`, `hover:text-(--accent)` are banned. Hover changes background tint (`--hover` / `--row-hover`), never border or text color. Accent is for the resting active/CTA state only.
- **Don't use `<Button variant="outline">`.** Runtime variants are `default | primary | ghost | danger | success | icon | neutral`. The outline silhouette is gone. Enforced by `compozy-design-system/no-banned-props`.
- **Don't import `Loader2` from `lucide-react` in production routes.** Use `<Spinner>` for "this surface is loading data" and `.dot--pulse` for "this entity is in an active running state". Enforced by `compozy-design-system/no-banned-imports`.
- **Don't write "we" / "our" in marketing body.** Product is the subject.
- **Don't render on a white background.** Dark mode only.

## 11. Anti-patterns

Anti-patterns are bans that are easy to slip into and expensive to unwind across many call-sites. Each item below is grep-banned, lint-banned, or test-banned in the runtime contract — `web/` and `packages/ui`. The `agh-design` skill's rules block carries the author-side mirror; this section is the human-readable spec.

### Side-stripe accent rail on cards or rows

- **Pattern:** `border-l-2 border-l-(--accent)` / `border-l-accent` painted on `Metric`, `KpiCard`, `Section`, `RadioCard`, `CatalogCard`, list rows, or any "active" surface as a left-edge indicator.
- **Why banned:** The selected-rail rule moves the selected-rail color to `var(--fg-strong)` (white, 2 px) — the accent rail collapses the "one accent per viewport" budget into row-level decoration. Accent stays reserved for primary CTAs, the active workspace nub, and the rail logo plate.
- **Replace with:** `bg-(--row-selected)` or `bg-(--surface-glaze)` plus the white 2 px rail emitted by `ACTIVE_NAV_INDICATOR_CLASS` (sidebar nav) or the list-row equivalent.
- **Enforcement:** `compozy-design-system/no-side-stripe-accent`.

### Eyebrow misuse — inline mono uppercase tuples and structural-vs-metric confusion

- **Pattern A — inline tuple:** `font-mono uppercase tracking-[0.06em] text-[11px]` (or any arbitrary `tracking-[Nem]` tuple) open-coded next to a label. That tuple IS the eyebrow contract; reinventing it skips the `<Eyebrow>` primitive and the `--text-eyebrow` / `--text-badge` / `--text-micro` / `--tracking-mono` tokens.
- **Pattern B — wrong register:** `<Eyebrow>` applied to `KpiCard` / `Metric` value labels. Eyebrows are for structural section heads, table heads, run-cells, and protocol identifiers. KPI / Metric labels are sentence-case Inter 12 px / 510 / -0.005em — NOT uppercase mono Eyebrow.
- **Replace with:** `<Eyebrow>` (prop-less / L-022 — `{ children, className }` only) for structural labels; tone via `className="text-(--muted|subtle|accent|success|warning|danger|info)"`. KPI / Metric labels stay bare sentence-case Inter.
- **Enforcement:** `compozy-design-system/no-inline-eyebrow` + `compozy-design-system/no-inline-design-tuples`.

### Accent overload — more than one accent target per viewport

- **Pattern:** Multiple accent surfaces co-occurring in the same viewport — accent CTA + accent side-stripe + accent tab indicator + accent count chip + accent border-on-hover. Accent leaks into selected rails, active tabs, hover states, and count chips when its single contract is "the next action".
- **Why banned:** Accent is the highest-signal token in the palette. When every surface competes for it, none of them carry information — the eye has nowhere to land. The "one accent target per viewport" rule (`agh-design` SKILL.md rules block) protects the CTA hierarchy.
- **Replace with:** `--fg-strong` for selected rails; `--fg-strong` 1.5 px for `<Tabs variant="line">` indicators; `--badge-fill` / `--muted` 3 px chip for `<PillGroup>` count badges; `--row-hover` / `--row-selected` glaze for hover/selected backgrounds. Accent stays on the active CTA, the rail logo plate, and the active workspace nub.
- **Enforcement:** Combined effect of `no-side-stripe-accent` + class-list snapshots on `Tabs` / `PillGroup` / `Pill` primitives.

### `Section` as page-head — body-side 22 px H1 duplicating route identity

- **Pattern:** `<Section title={routeTitle}>` rendering a 22 px (or larger) H1 at the top of a non-detail page (List / Kanban / Dashboard / Inbox / Knowledge / Bridges / Skills / Agents / Settings / Sandbox / Triggers / Jobs / Network). The route's identity already lives in the global `<Topbar>` (14 px route title + count chip + meta slots); the body-side H1 doubles it.
- **Why banned:** The hierarchy rule splits `<Section>` (body H2, 13 px) from `<DetailHeader>` (24 px H1, detail-only). The proposal's 22 px page-head tier does not exist in the runtime contract — `Section` may render a body H2 but never a page-level H1.
- **Replace with:** Flow content directly under the topbar. For detail surfaces only, emit `<DetailHeader>` with the 6-row anatomy (crumbs / pre-title / 24 px H1 / pills / meta / actions). The runtime does NOT introduce a `<PageHead>` primitive.
- **Enforcement:** `compozy-design-system/no-inline-design-tuples` flags the `text-[22px].*tracking-[-0.026em]` page-h1 tuple; class-list snapshots on `<Section>` lock the 13 px body H2.

## 12. Responsive Behavior

### Breakpoints

Desktop-first, optimized for 1440px wide screens. Site content is capped at `--site-layout-width: 1200px`; docs extend to `96rem`.

- **Sidebar (UI):** fixed width on desktop. On narrow viewports, collapse to icon-rail-only mode (40px).
- **Three-panel (UI):** list + detail share the content area. On narrow viewports, stack as full-width views with back navigation.
- **Metric cards:** 4-column grid → 2-column or stack on narrow viewports.
- **Filter groups:** horizontal flex-wrap. Pills and dropdowns wrap to a second line when space is constrained.
- **Hero (marketing):** `grid-cols-[minmax(0,1fr)_minmax(0,540px)]` → single column on mobile, visual above copy.
- **Docs layout:** left sidebar + right TOC → hidden on narrow viewports; sidebar becomes a drawer.

### Touch Targets

- **Minimum interactive height:** 28px (ghost/compact buttons).
- **Standard interactive height:** 36px (buttons, inputs, list items).
- **Marketing CTA:** 44px (`lg`).
- **Filter pills:** 32px.
- **Sidebar nav items:** full-width hit area with 8px 12px padding.
- **Status dots:** 8px visual, wrapped in a larger clickable area when interactive.

### Sidebar Modes

- **Expanded:** icon rail (40px) + panel (~220px). Agent list, nav items, system footer visible.
- **Collapsed:** icon rail only (40px). Workspace circles visible, panel content hidden.

## 13. Site Profile (`packages/site` extensions)

The marketing + Fumadocs site at `agh.network` ships through `packages/site` and is the only AGH surface that loads Playfair Display (`--font-display`) and NuixyberNext (`--font-wordmark`). It SHARES the runtime contract for color, surface, text, accent, hairlines, signal, overlays, motion, and shadows — runtime-side primitives in `packages/ui/src/tokens.css` are authoritative for both surfaces. Site EXTENDS the contract with responsive typography clamps, layout widths, marketing/doc body styles, and bento overlays declared in `packages/site/app/global.css`. Inventing a new `--color-*` or non-clamp `--text-*` token in site-scope drifts from the contract and MUST be added to this section first.

### What site shares with runtime (do NOT redefine)

- Surface ramp: `--rail`, `--canvas`, `--canvas-soft`, `--canvas-tint`, `--sidebar`, `--elevated`, `--hover`, `--disabled`.
- Hairlines: `--line`, `--line-soft`, `--line-strong`.
- Text: `--fg`, `--fg-strong`, `--muted`, `--subtle`, `--faint`.
- Accent + signal: `--accent`, `--accent-{hover,strong,ink,tint,tint-strong,dim,glow}`, `--success`, `--warning`, `--danger`, `--info`, `--neutral` (+ `*-tint` siblings).
- Overlays: `--overlay-scrim`, `--overlay-ghost-hover` (the runtime `::selection` rule resolves `var(--accent-tint)` directly; `--overlay-selection` is intentionally absent).
- Motion: `--dur`, `--dur-slow`, `--ease`, `--ease-in-out`, `--duration-fast/base/slow`, `--ease-out`.
- Shadow whitelist: `--shadow-overlay`, `--highlight` (modals + active rim only).
- Eyebrow utility: single `eyebrow` `@utility` (Inter UC 11 / 600 / 0.05em — `--text-eyebrow` + `--tracking-section-label`) and the prop-less `<Eyebrow>` component from `@agh/ui`. The legacy `eyebrow-badge` / `eyebrow-micro` tiers are deleted from the runtime contract.

### What site adds — fonts

| Token             | Family               | Loaded by                       | Scope                                         |
| ----------------- | -------------------- | ------------------------------- | --------------------------------------------- |
| `--font-display`  | **Playfair Display** | `next/font` in `app/layout.tsx` | `.site-home` only (landing hero + section H2) |
| `--font-wordmark` | **NuixyberNext**     | `@agh/ui/logos`                 | `agh` wordmark in `home-header` only          |

`--font-sans` (Inter Variable) and `--font-mono` (JetBrains Mono) inherit from runtime. Site re-declares them in its own `@theme inline` so Next.js font CSS variables (`--font-inter`, `--font-jetbrains-mono`, `--font-playfair`) bind correctly.

### What site adds — responsive type clamps

All site clamps live in `packages/site/app/global.css` `@theme inline`. They are the only legitimate place to declare site-scoped `--text-site-*` typography tokens. Each clamp follows the `clamp(min, vw, max)` formula so hero / page / card titles fluidly scale across the home page, blog, changelog, runtime docs, and protocol docs.

| Group            | Tokens                                                                                                                                                                                                                                  |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Hero             | `--text-site-hero` (clamp 2.8–5.4 rem), `--text-site-error-title` (2.6–4.8), `--text-site-hero-section` (2.6–4.2)                                                                                                                       |
| Page / article   | `--text-site-blog-title` (2.6–4.6), `--text-site-doc-title` (2.55–4.0), `--text-site-protocol-title` (2.4–3.8), `--text-site-page-title` (2.4–4.0), `--text-site-article-title` (2.4–3.6), `--text-site-section-title` (2.2–3.6)        |
| Sub-sections     | `--text-site-category-title` (2.2–3.4), `--text-site-cta-title` (2.0–3.2), `--text-site-feature-title` (2.0–2.8), `--text-site-subsection-title` (1.9–2.6), `--text-site-doc-heading` (1.7–2.45), `--text-site-release-title` (1.6–2.1) |
| Cards / quote    | `--text-site-empty-title` (1.55–2.0), `--text-site-quote` (1.5–1.95), `--text-site-card-title` (1.45–1.9), `--text-site-subheading` (1.3–1.7)                                                                                           |
| Bento (fixed)    | `--text-site-bento-2xl` (2.5 rem), `-xl` (2.35), `-lg` (2.0), `-md` (1.9), `-sm` (1.8), `-xs` (1.65)                                                                                                                                    |
| Lead / body      | `--text-site-lead` (1.1875 rem, line-height 1.5), `--leading-doc-body` (1.8 — used by `.site-doc-body`)                                                                                                                                 |
| Inline / accent  | `--text-inline-code` (0.9 em), `--text-accent-glyph` (0.85 em), `--text-ui-title-lg` (1.35 rem), `--text-display-2xl` (1.75 rem)                                                                                                        |
| Eyebrow (shared) | `--text-eyebrow`, `--text-badge`, `--text-micro` (re-declared so site clamps stay in the same `@theme` block; values match runtime)                                                                                                     |

Use the Tailwind utilities `text-site-hero`, `text-site-blog-title`, etc. directly — site clamps are exposed automatically through `@theme inline`. Do NOT reach for arbitrary `text-[clamp(...)]`; if a new size is needed, add a new `--text-site-*` token to this table first.

### What site adds — layout primitives

Declared in `:root` of `packages/site/app/global.css`:

| Token                      | Value    | Role                               |
| -------------------------- | -------- | ---------------------------------- |
| `--site-layout-width`      | `1200px` | Landing / marketing page max-width |
| `--site-doc-layout-width`  | `96rem`  | Fumadocs notebook max-width        |
| `--site-doc-sidebar-width` | `16rem`  | Left sidebar in docs               |
| `--site-doc-toc-width`     | `14rem`  | Right TOC rail in docs             |

Reach these via Tailwind v4 arbitrary syntax — `max-w-(--site-layout-width)`, `w-(--site-doc-toc-width)`, etc.

### What site adds — CSS classes

| Class                                                                    | Where                                         | Purpose                                                                                                                                                                                                |
| ------------------------------------------------------------------------ | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `.site-home`                                                             | landing route only                            | Gates `--font-display` (Playfair) so it never leaks into runtime UI or non-home site routes. `.site-home h1, .site-home h2` switch to `var(--font-display)`.                                           |
| `.site-doc-body`                                                         | Fumadocs MDX bodies (runtime + protocol docs) | Doc reader styles — `color: var(--muted)`, `font-size: 1rem`, `line-height: 1.8`, `max-width: 72ch` on paragraphs/lists/blockquotes, h2 with top-border separator, code chips with `--canvas-soft` bg. |
| `.site-bento-overlay-{runtime,network,bridges,memory,extensibility}`     | bento illustrations                           | Five gradient overlays that fade illustration art into `--rail` (`#0c0b0b`) at the top edge so each bento card has a consistent dark vignette.                                                         |
| `.agh-mermaid` (and `.agh-mermaid .node`, `.cluster`, `.edgePath`, etc.) | Mermaid diagrams (docs)                       | Mermaid theme overrides — node bg `--elevated`, node border `--accent`, edges `--subtle`, cluster bg `--canvas-soft`, text `--font-mono`.                                                              |

Fumadocs neutral preset is also overridden here via `--color-fd-*` aliases — see `packages/site/app/global.css :81–107` and the `#nd-sidebar` / `#nd-toc` selectors.

### Authoring rules

1. When working in `packages/site`, color / surface / text / accent / signal / motion ALWAYS resolve from `packages/ui/src/tokens.css`. Never re-declare them in site CSS or invent `--site-color-*` shadows.
2. Typography sizing in marketing or doc copy uses the `--text-site-*` clamp utilities. If a needed clamp is missing, add it to the table above and to `app/global.css @theme inline` in the same change.
3. Layout widths use the `--site-*` layout tokens above. Never inline `max-w-[1200px]` — use `max-w-(--site-layout-width)`.
4. Mono uppercase labels use the shared eyebrow utilities or `<Eyebrow>` component (see §3). Site does NOT get its own eyebrow vocabulary.
5. Playfair Display (`--font-display`) is allowed only inside `.site-home`. NuixyberNext (`--font-wordmark`) is allowed only on the literal `agh` wordmark. Anywhere else in site (blog body, changelog, runtime/protocol docs) uses Inter Variable.

## 14. Agent Prompt Guide

### Quick Color Reference

```
Rail:           #0c0b0b
Canvas:         #131211
Canvas Soft:    #1a1918
Canvas Tint:    #1c1b1a
Sidebar:        #1a1918  (semantic alias of canvas-soft)
Elevated:       #232220
Hover: var(--row-hover) (alias of glaze --row-hover)
Disabled:       #4a4847

Line:           rgba(255,255,255,0.055)
Line Soft:      rgba(255,255,255,0.03)
Line Strong:    rgba(255,255,255,0.09)

Fg:             #ececef
Fg Strong:      #f6f6f8
Muted:          #9a9a9f
Subtle:         #76767c
Faint:          #545458

Accent:         #e8572a
Accent Ink:     #17110f   (text on accent fill)
Accent Hover:   #d14e25
Accent Strong:  #f6874f
Accent Dim:     rgba(232,87,42,0.24)
Accent Tint: rgba(232,87,42,0.10) (::selection background)
Accent Tint S.: rgba(232,87,42,0.16)
Accent Glow:    rgba(232,87,42,0.05)

Success:        #5fbf85   tint rgba(95,191,133,0.08)
Warning:        #d6a647   tint rgba(214,166,71,0.08)
Danger:         #e0635a   tint rgba(224,99,90,0.09)
Info:           #8e8eb5   tint rgba(142,142,181,0.12)  (Settings / observability)
Neutral:        #7a7a80   tint rgba(150,150,155,0.06)  (warmed to match ramp)

Scrim:          rgba(0,0,0,0.55)        (--overlay-scrim)
Scrim blur:     3px                     (--overlay-blur, dialog/sheet ONLY)
Ghost Hover:    rgba(255,255,255,0.06)  (--overlay-ghost-hover)
Selection:      var(--accent-tint)      (::selection { background: var(--accent-tint); color: var(--fg-strong); })
Tint Formula:   <signal-color> at 6–12% alpha for bg, full color for text

Surface glaze ladder (--row-hover .. --badge-fill):
  row-hover         rgba(255,255,255,0.022)   list/nav hover (alias --hover)
  row-selected      rgba(255,255,255,0.030)   list/nav selected
  surface-glaze     rgba(255,255,255,0.040)   RadioCard / panel-head selected
  bar-fill          rgba(255,255,255,0.085)   priority / progress / usage bars
  input-fill        rgba(255,255,255,0.025)   composer / textarea / search
  btn-default-fill  rgba(255,255,255,0.040)   neutral Button default
  btn-default-hover rgba(255,255,255,0.070)   neutral Button hover
  badge-fill        rgba(255,255,255,0.050)   PillGroup count badge

Owner avatar palette (--avatar-{agent,human,system}-{slot}-{bg,fg}):
  agent 0..3        4 warm/cool slots, bg ~18% alpha, fg solid hex
  human 0..2        3 warm slots, bg ~20% alpha, fg solid hex
  system            bg --elevated, fg --subtle

Modal width ladder:
  --width-modal-sm  560px  confirm / single-field editor
  --width-modal-md  720px  task editor (new/edit) / settings field editor
  --width-modal-lg  880px  bridges wizard / knowledge create dialog
```

### Quick Type Reference

```
UI / Body (runtime):     Inter Variable, weight 510 ladder + 400 body
Metadata (runtime):      JetBrains Mono, weight 500 (mono pill), 400 (code)
Marketing h1/h2:         Playfair Display (packages/site only, .site-home scope)
Docs h1/h2:              Inter, weight 600, packages/site only
Wordmark:                NuixyberNext (packages/site only, "agh" lockup)
```

### Example Component Prompts

**"Create a marketing hero"**

> `.site-home` container. Grid: copy-left / visual-right (`minmax(0,1fr)_minmax(0,540px)`). Canvas bg with `/hero-bg.png` mesh at 20% opacity + `mix-blend-screen`. Copy: mono eyebrow `WHAT YOU GET` in `#636366`. Title: Playfair Display, `clamp(2.8rem, 6.5vw, 5.4rem)`, weight 400, tracking -0.035em, line-height 0.96. Sub: Inter 18px regular, `#8E8E93`, max-width 58ch. CTAs: primary accent pill (`#E8572A`, white text, `lg` 44px, rounded-lg) + ghost outline CTA that warms border/text on hover. No shadows, no gradients on content.

**"Create a feature card grid"**

> 3-column grid, 16px gap. Each card: bg `#1E1C1B`, radius 12px, border 1px `#3C3A39`, padding 24px, `ring-1 ring-foreground/10`. Inside: 40×40 icon well (bg `#2E2C2B`, radius 10px, Lucide icon 16px stroke-2 in `#E8572A`) → mono eyebrow 11px uppercase `#636366` → Inter 20px medium title (verb-forward, three words) → Inter 14px regular description `#8E8E93` → optional mono source cite with `ArrowUpRight` 12px. Hover: border → `color-mix(in srgb, #E8572A 40%, #3C3A39)`. 150ms ease-out.

**"Create a docs page"**

> Sticky header: `rgba(20,19,18,0.92)` + `backdrop-blur-xl`, bottom border 1px `#3C3A39`. NuixyberNext `agh` wordmark + `ALPHA` chip. Layout: `96rem` max, `16rem` left sidebar + content + `14rem` right TOC. Masthead: mono eyebrow 12px tracking 0.16em `#636366`, then H1 Inter 600 clamp ramp tracking -0.05em, sub-lead Inter 18px `#8E8E93`. Body: Inter 16px line-height 1.8 `#8E8E93` max-width 72ch. H2: Inter 600 clamp ramp, `border-top 1px #3C3A39`, `padding-top 1rem`.

**"Create a session list panel (operator UI)"**

> Surface bg (`#1E1C1B`). Section header in JetBrains Mono 11px 600 uppercase `#636366` with count right-aligned. List items: 8px status dot + Inter 14px Medium name + Inter 13px version right-aligned. Selected item: bg `#2E2C2B` with 3px left accent bar `#E8572A`. Border-bottom 1px `#3C3A39` between items.

**"Create a metric dashboard row"**

> 4-column grid, 16px gap. Each card: bg `#1E1C1B`, radius 12px, padding 16px 20px, border 1px `#3C3A39`. Eyebrow: JetBrains Mono 11px 600 uppercase 0.06em tracking `#636366`. Value: Inter 24px 700 -0.02em tracking `#E5E5E7`. Semantic values: positive `#30D158`, negative `#FF453A`, warning `#FFD60A`. Optional subtext: Inter 13px 400 `#8E8E93`.

**"Create a code block"**

> Container: bg `#1a1918` (`--canvas-soft`), radius 12px, padding 20px. Font: JetBrains Mono 14px, line-height 1.6. Prompt `$ ` in `--accent`, command in `--fg`. Copy button: absolute top-right, ghost, tertiary icon → accent on hover, checkmark swap for 1.5s on copy success. Optional language eyebrow top-left.

**"Create a filter toolbar"**

> Horizontal flex, 8px gap. Active pill: bg `#E8572A`, white text, radius 20px, height 32px, padding 6px 14px. Inactive pills: border 1px `#3C3A39`, text `#8E8E93`. Dropdown filters: pill, border `#3C3A39`, chevron. Active dropdown: bg `#E8572A1F`, border `#E8572A`, text `#E8572A`. Search input: bg `#2E2C2B`, radius 8px, height 36px, search icon `#636366`. Secondary button: border 1px `#3C3A39`, Inter 14px Medium, icon + text.

**"Create a chat conversation"**

> User message: right-aligned, bg `#2E2C2B`, radius 12px, padding 16px 20px, Inter 14px `#E5E5E7`. "YOU" label above in mono 11px uppercase `#636366`. Agent message: left-aligned, no bubble. Agent name in JetBrains Mono 11px uppercase with status dot. Body in Inter 14px `#8E8E93`. Tool calls: bg `#1E1C1B` cards, border `#3C3A39`, terminal icon, tool name, file path, status badge right-aligned.

### Implementation Checklist

1. **Fonts loaded (runtime):** Inter Variable (400 + 510 + 600), JetBrains Mono (400 + 500 + 600). Site adds Playfair Display + NuixyberNext via its own Next.js font stack. `--font-display` is declared as an Inter alias for symmetry but FORBIDDEN inside type-ramp callers in `web/src/**` / `packages/ui/src/**` — reserved for explicit logo/wordmark slots.
2. **Canvas background:** `--canvas` (`#131211`) on `<body>`. `color-scheme: dark` hardcoded. `.dark` forced on `RootProvider` with `enabled: false`. Body baseline: `font-size: 0.84375rem`, `line-height: 1.5`, `letter-spacing: -0.006em`, `font-feature-settings: "cv01", "ss03", "cv11"`.
3. **Flat depth:** only `--shadow-overlay` (modals / sheets) and `--highlight` (active rim). Every other surface stays flat with a 1 px ring on `--line-soft`. Modal scrim adds `backdrop-filter: blur(var(--overlay-blur))` (3 px) on `Dialog.Overlay` / `Sheet.Overlay` only.
4. **Buttons:** `--radius-md` (8 px), heights 22 / 26 / 30 (Inter 12 px / 510 / -0.005em). Never pill. Neutral default fill resolves `var(--btn-default-fill)` / `var(--btn-default-hover)` (glaze ladder).
5. **Badges:** `--radius-mono-badge` (4 px, retuned from 6 px), 22 px height, signal tint bg, JetBrains Mono 10.5 px / 500 / 0 (Pill `--mono`). Status pill default uses Inter 11 px / 510 sentence case.
6. **Inputs:** `--radius-md` (8 px), bg `var(--input-fill)` for composer / textarea / search, border 1 px `--line`, focus ring `0 0 0 1px var(--line-strong)`.
7. **Cards:** `--radius-lg` (10 px), bg `--canvas-soft`, 1 px `--line` ring. No shadows. Selected card uses `var(--surface-glaze)`.
8. **Filter pills (UI):** `--radius-pill`, accent fill (active) or 1 px `--line` (inactive). `<PillGroup>` count badge fills with `var(--badge-fill)`.
9. **Dividers:** 1 px solid `--line`; group bottoms use `--line-soft`. Focus ring uses `--line-strong` (white). No accent focus ring. Selected list/nav rails use `var(--fg-strong)`, never `--accent`.
10. **Signal colors:** `--accent` `#e8572a`, `--success` `#5fbf85`, `--warning` `#d6a647`, `--danger` `#e0635a`, `--info` `#8e8eb5`, `--neutral` `#7a7a80`. Tinted chips only; never solid banners. `--info-tint` is 12 % (Settings / observability), `--neutral-tint` 6 % warm (`rgba(150,150,155,0.06)`).
11. **Voice:** operator-first, dry, no emoji, no "we". Sentence case copy by default; UPPERCASE only on sidebar labels, table heads, run-cells, and `<Eyebrow>` (which is Inter UC 11 / 600 / -0.005em — single style, no `case` / `family` / `tone` / `size` props).
12. **Icons:** Lucide, stroke 2, 12–20 px inline, 48 px for empty-state only. Accent inside icon wells, `currentColor` elsewhere.
13. **Reduced motion** — the universal-selector `@media (prefers-reduced-motion: reduce)` block in `tokens.css` pins `animation-duration`, `animation-delay`, and `transition-duration` to `0.001ms !important` on `*, *::before, *::after`. The non-zero value preserves CSS `animationend` callback timing. Never bypass with `!important` or per-component opt-in.
14. **Layout grammar (§3a):** Topbar carries page identity (14 px route title). `<DetailHeader>` is the only in-body 24 px H1 surface. `<PageShell density="route">` applies the 28/36/80 envelope. Modal widths pull from `--width-modal-{sm,md,lg}` (560/720/880). Sidebar collapse ladder: 244 → 220 (≤ 1100 px) → drawer (≤ 880 px).
15. **Status tones:** six tones (`neutral / accent / success / warning / danger / info`). `violet` is not a token — approvals collapse to `info`. All tone mappings live in `web/src/lib/status-tone.ts` exhaustive dictionaries (`STATUS_TONE` / `RUN_STATUS_TONE` / `TASK_LANE_TONE`).
16. **Owner avatars:** consume `--avatar-{agent,human,system}-{slot}-{bg,fg}` via `web/src/lib/owner-palette.ts → colorsFor()`. Inline hex is forbidden — Storybook + design ref tools must read from the same token source.
