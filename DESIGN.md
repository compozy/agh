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
- **Two-token shadow vocabulary (ADR-003)** — only `--shadow-overlay` (modals) and `--highlight` (active rim) carry shadow. Cards, popovers, dropdowns, sticky headers, list rows stay flat with a 1 px ring on `--line-soft`.
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
| Info    | `--info`    | `#8e8eb5` | `--info-tint`    | `rgba(142, 142, 181, 0.07)` | 7 %    |
| Neutral | `--neutral` | `#7a7a80` | `--neutral-tint` | `rgba(122, 122, 128, 0.06)` | 6 %    |

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

| Purpose         | Token                   | Value                       | Notes                             |
| --------------- | ----------------------- | --------------------------- | --------------------------------- |
| **Modal scrim** | `--overlay-scrim`       | `rgba(0, 0, 0, 0.5)`        | Dialog / sheet backdrop           |
| **Ghost hover** | `--overlay-ghost-hover` | `rgba(255, 255, 255, 0.06)` | Ghost button hover on dark        |
| **Selection**   | `--overlay-selection`   | `rgba(232, 87, 42, 0.28)`   | Text selection — warm accent tint |

The runtime kit no longer ships a glass / `backdrop-blur` overlay token. The marketing site's sticky header (`packages/site`) keeps its own blur via its own stack.

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
- **Eyebrow markup is mandatory.** Every uppercase mono label in `web/` and `packages/site` MUST render through `<Eyebrow>` (`@agh/ui`). Inlining `font-mono` + `uppercase` + a `text-*`/`tracking-*` tuple in product spans/paragraphs is forbidden — `@agh/ui` ships the structural primitives (`<Sidebar.SectionLabel>`, `<TableHead>`, `<MetadataList.Term>`, `<WireCardHead>`) for the cases that must inherit eyebrow typography on a non-span element. Canonical tracking is `--tracking-mono` (0.06em). Never reach for arbitrary values like `tracking-[0.05em]`, `text-[10.5px]`, or the legacy `--tracking-badge` (0.08em) for eyebrow text.
- **Negative tracking on titles** (-0.014em to -0.028em) tightens the page-h1 / detail-h1 / topbar-title hierarchy. Body and small text use a slight negative tracking (-0.005em to -0.006em) for crisp ranges on the warm canvas.
- **No bold UI weight.** The ladder tops out at 510. Body is 400, never bold.
- **Signal text on tint surfaces** — text uses the full signal hex, the tint token sits behind. The color carries the meaning.
- **Selection color** uses `--overlay-selection` (`rgba(232, 87, 42, 0.28)`). Warm accent, not default blue.

## 4. Component Stylings

### Buttons

#### Primary (CTA)

Solid accent fill. The main call-to-action on marketing, primary button in UI.

| State    | Background | Text      | Border Radius | Height (default / lg)   | Padding  |
| -------- | ---------- | --------- | ------------- | ----------------------- | -------- |
| Default  | `#E8572A`  | `#FFFFFF` | 8px           | 36px / 44px             | 8px 20px |
| Hover    | `#D14E25`  | `#FFFFFF` | 8px           | same                    | same     |
| Active   | `#D14E25`  | `#FFFFFF` | 8px           | same + `translate-y-px` | same     |
| Disabled | `#4A4847`  | `#636366` | 8px           | same                    | same     |

Marketing CTAs use `lg` (44px, `rounded-lg`). **Never pill-shaped.**

#### Secondary / Outline

Border-only. Hover warms the border toward accent.

| State    | Background  | Border                                               | Text      |
| -------- | ----------- | ---------------------------------------------------- | --------- |
| Default  | transparent | 1px solid `#3C3A39`                                  | `#E5E5E7` |
| Hover    | transparent | 1px solid `color-mix(in srgb, #E8572A 40%, #3C3A39)` | `#E8572A` |
| Disabled | transparent | 1px solid `#3C3A39`                                  | `#636366` |

#### Ghost

Text-only. Hover reveals subtle background (`rgba(255,255,255,0.06)`).

| State     | Background               | Text      | Height | Padding      |
| --------- | ------------------------ | --------- | ------ | ------------ |
| Default   | transparent              | `#8E8E93` | 28px   | 6px 12px     |
| Hover     | `rgba(255,255,255,0.06)` | `#E5E5E7` | 28px   | 6px 12px     |
| Icon Only | transparent              | `#8E8E93` | 28px   | 6px (square) |

Marketing ghost CTA override: `hover:border-accent hover:text-accent hover:bg-transparent`.

#### Danger

Same shape as primary. Red fill for destructive actions.

| State    | Background  | Text      |
| -------- | ----------- | --------- |
| Default  | `#FF453A`   | `#FFFFFF` |
| Hover    | Lighter red | `#FFFFFF` |
| Disabled | `#4A4847`   | `#636366` |

#### Pill Toggle / Filter Tabs (Operator UI)

| State    | Background  | Border              | Text      | Radius |
| -------- | ----------- | ------------------- | --------- | ------ |
| Active   | `#E8572A`   | none                | `#FFFFFF` | 20px   |
| Inactive | transparent | 1px solid `#3C3A39` | `#8E8E93` | 20px   |

Filter pill dimensions: height 32px, padding 6px 14px, gap 6px.

#### Header Nav Pills (Site)

Round-full pills used only in the marketing + docs site header.

| State   | Background                | Text      | Radius |
| ------- | ------------------------- | --------- | ------ |
| Default | transparent               | `#8E8E93` | 9999px |
| Hover   | `rgba(232, 87, 42, 0.12)` | `#E5E5E7` | 9999px |
| Active  | `rgba(232, 87, 42, 0.12)` | `#E8572A` | 9999px |

### Badges, Chips & Mono Labels

Three related but distinct primitives:

#### Status Badge

Semantic-tinted pill for runtime states. Always JetBrains Mono, uppercase.

- **Dimensions:** height 22px, padding 3px 8px, radius 6px (`--radius-mono-badge`).
- **Type:** JetBrains Mono 10px, weight 600, tracking 0.08em, uppercase.
- **Coloring by signal** (background = 15% tint, text = full color):
  - **Accent** — submitted, running, active
  - **Success** — filled, approved, healthy
  - **Danger** — rejected, error, critical, halted
  - **Warning** — partial, pending, degraded
  - **Info** — info, guardian type
  - **Neutral** — cancelled, idle, inactive

#### Mono Badge

Inline mono pill for identifiers (agent IDs, versions, protocol strings).

- **Radius:** 6px (`--radius-mono-badge`).
- **Padding:** 2px 6px.
- **Type:** JetBrains Mono 11px, weight 500, tracking 0.06em.
- **Default:** border 1px `#3C3A39`, text `#98989D`, background transparent.
- **Variants:** same tint-formula as status badges when they carry a signal.
- **`solid-accent` (reserved):** solid `#E8572A` background, `#17110F` text. Reserved for unread-count pills inside channel/nav rows — never for general status.

#### Mono Chip

Neutral inline chip for capability descriptors and tag rows (`code`, `shell`, `file.read`, `plan.delegate`).

- **Radius:** 5px (`--radius-chip`).
- **Padding:** 2px 6px.
- **Type:** JetBrains Mono 10px, weight 500, tracking 0.04em.
- **Background:** `#2E2C2B` (surface-elevated). **Text:** `#8E8E93`.
- Use when an identifier needs a neutral chip without a semantic tone. For tinted variants reach for `MonoBadge`.

#### Kind Chip

Wire-protocol kind marker (`say`, `greet`, `direct`, `receipt`, `recipe`, `trace`, `whois`).

- **Radius:** 3px.
- **Padding:** 1px 6px.
- **Type:** JetBrains Mono 9.5px, weight 600, uppercase, tracking 0.08em.
- **Surface:** transparent with 1px `#3C3A39` border. **Text:** `#636366`.
- **Wire-dot prefix:** 7×7 circle whose color is keyed off `kind` —
  - `say #8E8E93` · `greet #5BA6FF` · `direct #E8572A` · `receipt #30D158` · `recipe #FFD60A` · `trace #B892FF` · `whois #4FD1C5`.
- Unknown kinds (platform names, event ids) render the chrome without a dot.

#### Wire Chip

Free-floating filter chip used in stand-alone filter rows (the network channel header `ALL · SAY · DIRECT · …`). Distinct from `Pills`, which renders a contained segmented track.

- **Radius:** 4px.
- **Padding:** 3px 8px.
- **Type:** JetBrains Mono 10.5px.
- **Inactive:** bg `#1E1C1B`, border 1px `#3C3A39`, text `#8E8E93`.
- **Hover:** border `#636366`, text `#E5E5E7`.
- **Active:** bg `#2E2C2B`, border `#636366`, text `#E5E5E7` — never solid accent.
- **Dot prefix (optional):** 7×7 circle when the chip carries a kind, color from the same map as Kind Chip.

#### ALPHA Chip (brand)

Sits next to the `agh` wordmark.

- Mono, uppercase, tracking-widest, 10px.
- Transparent fill, 1px muted border (`#3C3A39`), `#98989D` text.

### Metric Cards

Three variants, all using surface background with 12px radius (`--radius-diagram`).

#### Simple

- **Container:** bg `#1E1C1B`, radius 12px, padding 16px 20px, gap 8px.
- **Label (eyebrow):** JetBrains Mono 11px, 600, uppercase, 0.06em tracking, `#636366`.
- **Value:** Inter 24px, 700, -0.02em tracking, `#E5E5E7`.
- **Semantic values:** positive `#30D158`, negative `#FF453A`, warning `#FFD60A`.

#### With Subtext

Simple + a secondary line: Inter 13px, 400, `#8E8E93`.

#### With Sparkline

Simple + an inline SVG sparkline aligned right of the value.

### Feature Card (Marketing)

The canonical marketing card. Pattern: icon well → eyebrow → verb-forward title → mechanism sentence → optional mono source cite.

- **Container:** bg `#1E1C1B`, radius 12px, border 1px `#3C3A39`, padding 24px, `ring-1 ring-foreground/10`.
- **Icon well:** 40×40, radius 10px, bg `#2E2C2B`, accent-colored Lucide icon (16px, stroke 2).
- **Eyebrow:** JetBrains Mono 11px 600 uppercase, 0.06em tracking, `#636366`.
- **Title:** Inter 20px Medium 500, `#E5E5E7`, three-word verb-forward phrase.
- **Description:** Inter 14px Regular, `#8E8E93`, one-sentence mechanism.
- **Source cite (optional):** mono, tertiary text + `ArrowUpRight` (12px).
- **Hover:** border → `color-mix(in srgb, #E8572A 40%, #3C3A39)`. No lift, no scale, 150ms transition on color only.

### Inputs

#### Text Input (forms)

Form-grade input used inside Field/InputGroup composites.

- **Container:** bg `#2E2C2B`, radius 8px, height 36px, padding 0 12px, gap 8px.
- **Border:** 1px solid `#3C3A39` (default), 1.5px solid `#E8572A` (focused).
- **Placeholder:** Inter 14px Regular, `#636366`.
- **Text:** Inter 14px Regular, `#E5E5E7`.
- **Icon (if any):** search / prefix icon, 16px, `#636366`.
- **Disabled:** bg `#1E1C1B`, border `#2E2C2B`, text `#4A4847`.

#### Search / Filter Input (sidebar + panel)

Compact search/filter row used inside sidebars and list panels (`SearchInput`).

- **Container:** bg `#181716` (surface-panel), radius 7px, height 28px, padding 0 8px, gap 8px.
- **Border:** 1px solid `#3C3A39` (default), 1px solid `#636366` (focus-within). No accent ring.
- **Placeholder:** Inter 13px Regular, `#636366`.
- **Text:** Inter 13px Regular, `#E5E5E7`.
- **Search icon:** 12px, `#636366`.
- **Kbd hint (`⌘K`, `jump`, …):** JetBrains Mono 9px uppercase, padding 1px 4px, radius 4px, border 1px `#3C3A39`, bg `#181716`, text `#636366`. Hidden on mobile (`sm:inline-flex`).

#### Pills (segmented toggle)

Header-level segmented selector (`ALL / GLOBAL / WORKSPACE`, `LIST / KANBAN / DASHBOARD / INBOX`, `JOBS / TRIGGERS`, room detail tabs).

- **Track:** inline-flex, gap 2px, padding 3px, radius 8px, border 1px `#3C3A39`, bg `#181716` (surface-panel).
- **Segment:** height 22px, radius 5px, padding 0 10px, JetBrains Mono 10px weight 600 uppercase tracking 0.08em.
- **Inactive:** bg transparent, text `#636366`. **Hover:** text `#8E8E93`.
- **Active:** bg `#2E2C2B` (surface-elevated), text `#E5E5E7`. **Never solid accent.**
- **Inline badge (count):** min-width 14px, height 14px, radius 7px, bg `#E8572A`, text `#17110F`, JetBrains Mono 9px weight 700.

#### Dropdown Filter (legacy single trigger)

Pill-shaped trigger for a popover filter (not segmented).

- **Container:** radius 20px, height 32px, padding 6px 14px, gap 6px.
- **Border:** 1px solid `#3C3A39` (default), 1px solid `#E8572A` (active).
- **Active background:** `#E8572A1F` (~12% accent tint).
- **Text:** Inter 14px, `#8E8E93` (default), `#E8572A` (active).

#### Header Search Trigger (Site)

Round-full search trigger: bg `#1E1C1B`, border 1px `#3C3A39`, mono `⌘K` hint on the right, 36px height.

### Cards & Containers

#### Generic Card

Default operator UI card.

- **Container:** bg `#1E1C1B`, radius 12px, padding 16px 20px, border 1px `#3C3A39`.
- Used as the base for metric cards, session cards, etc.

#### Code Block

- **Container:** bg `#0E0E0F` (canvas-deep), radius 12px (`--radius-diagram`), padding 16–20px.
- **Font:** JetBrains Mono, 13–14px, 1.6 line-height.
- **Prompt:** `$ ` in `#E8572A`, rest of command in `#E5E5E7`.
- **Copy button:** ghost, absolute top-right, tertiary icon → accent on hover, checkmark swap for 1.5s on copy success.
- **Language label (optional):** mono eyebrow top-left, tertiary.

#### Metadata Table

Striped key-value rows.

- **Odd rows:** bg transparent.
- **Even rows:** bg `#1E1C1B`.
- **Key:** Inter 13px Regular, `#636366`.
- **Value:** Inter 14px Medium, `#E5E5E7`.
- **Badge values:** rendered inline as status badges.

#### Comparison Highlighted Row

The ONE "colored left border" pattern in the system.

- **Border-left:** 4px solid `#E8572A`.
- **Background:** `#E8572A26` (accent tint).
- Reserved for the marketing comparison table. **Do not proliferate** to other lists.

### Chat Components

#### User Message

Right-aligned bubble.

- **Bubble:** bg `#2E2C2B`, radius 12px, padding 16px 20px.
- **Text:** Inter 14px Regular, `#E5E5E7`.
- **Meta:** "YOU" + timestamp, JetBrains Mono 11px uppercase `#636366`, right-aligned above bubble.

#### Agent Message

Left-aligned, no bubble.

- **Agent label:** 8px dot (semantic color) + agent name (JetBrains Mono 11px uppercase `#98989D`) + timestamp.
- **Text:** Inter 14px Regular, `#8E8E93`.
- **Numbered lists:** Inter 14px Regular, `#8E8E93`.

#### Tool Call Card

Inline card showing tool execution.

- **Container:** bg `#1E1C1B`, radius 8px, padding 10px 16px, border 1px `#3C3A39`.
- **Icon:** terminal `>_`, `#636366`.
- **Tool name:** Inter 14px Medium, `#E5E5E7`.
- **File path:** Inter 13px Regular, `#636366`.
- **Status badge:** right-aligned (DONE / RUNNING / ERROR).

#### Wire Card

Bordered protocol card used to embed wire payloads (recipes, receipts, capability descriptors, room intros) inside message threads.

- **Shell:** border 1px `#3C3A39`, bg `#1E1C1B`, radius 6px, max-width 520px (or `w-full` when stretched inside a message body).
- **Head (`WireCardHead`):** bg `#0E0E0F` (canvas-deep), border-bottom 1px `#3C3A39`, padding 6px 10px, JetBrains Mono 10.5px uppercase tracking 0.06em, text `#636366`.
- **Body (`WireCardBody`):** padding 8px 12px, JetBrains Mono 11px.
- **Foot (`WireCardFoot`):** bg `#0E0E0F`, border-top 1px `#3C3A39`, padding 6px 10px, hosts ghost action buttons.
- **Inline variant (`inline`):** single-line strip, padding 6px 10px, gap 8px — used for receipt confirmations.

#### Typing Dots

Three-dot typing indicator paired with `<peer> is typing…` copy.

- 3× 4×4 dots, gap 2px, radius 50%, bg `#636366`.
- Animation: `typing-bounce` 1.2s infinite ease-in-out, with 0s / 0.15s / 0.3s stagger.
- Container copy: JetBrains Mono 11px `#636366`.

#### Chat Input

- **Container:** bg `#1E1C1B`, radius 12px, padding 12px 16px, border 1px `#3C3A39`.
- **Focused:** border 1px `#E8572A`.
- **Placeholder:** Inter 14px Regular, `#636366`.
- **Send button:** 36px circle, bg `#E8572A`, white send icon.

### Status Indicators

Inline dot + label patterns.

| Status               | Dot Color | Label           | Usage                                          |
| -------------------- | --------- | --------------- | ---------------------------------------------- |
| Connected / Online   | `#30D158` | "Connected"     | System footer, hero signal                     |
| Disconnected / Error | `#FF453A` | "Disconnected"  | System footer                                  |
| Degraded             | `#FFD60A` | "Degraded"      | Agent status                                   |
| Dream Status         | `#30D158` | "Dream: 3h ago" | Knowledge page header                          |
| Running              | `#E8572A` | "RUNNING"       | Tool call, active session                      |
| Pulse (live)         | `#30D158` | "Shipped today" | InstallSection — `animate-ping` on a 1.5px dot |

**Dot size:** 8px circle (6px for site pulse). Wrap in a larger clickable area when interactive.

### Sidebar (Operator UI)

#### Structure

- **Workspace icon rail:** 44px wide, left edge. 28px circle avatars.
  - App logo: `#E8572A` bg, white letter.
  - Active workspace: `#E8572A` border.
  - Inactive: `#2E2C2B` bg, `#8E8E93` letter.
  - Hover: `#353332` bg.
  - New: `#2E2C2B` bg, dashed border, `+` icon.
- **Sidebar panel:** bg `#0E0E0F` (canvas-deep), width 240px, full height.

#### Section Header (`SidebarSectionLabel`)

JetBrains Mono 9px weight 600 uppercase, tracking 0.14em, `#98989D` (`--color-text-label`). Padding 12px 12px 6px 12px. Same primitive used for `AGENTS`, `WORKSPACE`, `STARRED`, `CHANNELS`, `DIRECT MESSAGES`, and panel-internal subheaders.

#### Nav Row (top-level + channel rows)

Flat row, no border or card chrome.

- **Row:** padding 6px 8px, radius 6px, gap 8px (top-level) / gap 2.5 (channel rows).
- **Icon:** 13–14px, `#636366` default → `#E5E5E7` when active.
- **Label:** Inter 13px (top-level) / JetBrains Mono 12px (channel rows). Default text `#8E8E93`, active `#E5E5E7` weight 500. Unread channel rows render the label `#E5E5E7` weight 600.
- **Hover:** bg `#353332` (`--color-hover`).
- **Active:** bg `#1E1C1B` (`--color-surface`) **plus** a 2px-wide accent left bar (`#E8572A`) anchored against the panel edge (`-left-2` in a `px-2` nav container, `-left-1.5` in a `mx-1.5` row).
- **Unread badge:** `MonoBadge` `tone="solid-accent"` with the count.

#### Session Row (collapsible child)

Indented child row beneath each agent collapsible.

- **Row:** padding 4px 8px, radius 5px, font-size 12px, color `#8E8E93`.
- **Indent:** 18–22px from the agent label, with a 1px `#3C3A39` left rule between the indent and the row.
- **Active treatment:** same flat-row + 2px left accent bar pattern as Nav Row, anchored at `-left-3` to clear the indent line.

#### System Status Footer

Dot + label + version (`#636366`, right-aligned) + settings nav row.

### Site Header (Marketing + Docs)

- **Shell:** sticky top, bg `rgba(20, 19, 18, 0.92)` + `backdrop-blur-xl`, border-bottom 1px `#3C3A39`.
- **Wordmark:** NuixyberNext "agh" + `ALPHA` chip (mono 10px, muted border).
- **Nav pills:** round-full, hover + active tint `rgba(232,87,42,0.12)`.
- **Search trigger:** round-full pill, 36px, mono `⌘K` hint.
- **GH button:** round-full ghost with GitHub logo.

### Docs Masthead

- **Eyebrow:** JetBrains Mono 12px, 600, uppercase, tracking 0.16em, `#636366`.
- **Title:** Inter 600, clamp ramp (see Docs H1).
- **Sub-lead:** Inter 18px Regular, `#8E8E93`, max-width 58ch.

### Empty State

- **Icon:** 48px (rare exception to inline icon sizing), `#636366`, centered.
- **Title:** Inter 15px Medium, `#8E8E93`.
- **Description:** Inter 13px Regular, `#636366`.

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

AGH uses a **flat depth model** — depth comes from the warm surface ramp + 1 px translucent hairlines. Two shadow tokens are whitelisted (ADR-003); every other surface stays flat.

### Surface ramp

```
--rail (#0c0b0b) → --canvas (#131211) → --canvas-soft (#1a1918)
                → --canvas-tint (#1c1b1a) → --elevated (#232220)
```

Each step is a small, deliberate lightness increase. Hairlines (`--line` / `--line-soft` / `--line-strong`) carry the rest of the separation.

### Whitelisted shadows (ADR-003)

| Token              | Value                                                                         | Allowed on                                                                             |
| ------------------ | ----------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `--shadow-overlay` | `0 24px 48px -12px rgba(0, 0, 0, 0.65), 0 0 0 1px rgba(255, 255, 255, 0.045)` | `dialog`, `confirm-dialog`, `sheet`                                                    |
| `--highlight`      | `inset 0 1px 0 rgba(255, 255, 255, 0.035)`                                    | `button --primary`, active `pill-group` segment, active filter `pill`, rail logo plate |

Every other surface stays flat. Popovers, dropdowns, tooltips, and command menus use `box-shadow: 0 0 0 1px var(--line-soft)` plus `bg: var(--canvas-soft)` — the 1 px ring carries the separation.

### Depth Patterns

| Pattern                     | How it works                                                                                  |
| --------------------------- | --------------------------------------------------------------------------------------------- |
| **Card on canvas**          | `--canvas-soft` on `--canvas` — 1 px `--line` ring carries the edge                           |
| **Nested card / icon well** | `--elevated` inside `--canvas-soft` — e.g. search input, icon well                            |
| **Selected list item**      | `--elevated` + 2 px white indicator rail (selection) or 2 px accent rail (unread); never both |
| **Hover state**             | `--hover` replaces the current surface fill                                                   |
| **Divider**                 | 1 px solid `--line` between rows; `--line-soft` for softer subgroup splits                    |
| **Focus ring**              | `box-shadow: 0 0 0 1px var(--line-strong)` — white, never accent                              |
| **Floating overlay**        | `box-shadow: var(--shadow-overlay)` on dialog / sheet only                                    |
| **Active rim**              | `box-shadow: var(--highlight)` on primary button, active pill segment, rail logo plate        |

No ambient shadows, no `shadow-md` / `shadow-lg`, no glows. The styles regression test rejects every Tailwind shadow utility outside the two whitelisted resolutions.

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

`tokens.css` carries a universal-selector reduced-motion guard that zeros every animation and transition — not only those that read `--dur*`. This is required because `tw-animate-css` utilities and the `shimmer` / `typing-bounce` keyframes hardcode their durations and would otherwise keep playing.

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```

The `web/src/__tests__/styles.test.ts` regression test asserts the guard exists, applies to `*, *::before, *::after`, and zeros each of the four declarations.

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
- **Don't write "we" / "our" in marketing body.** Product is the subject.
- **Don't render on a white background.** Dark mode only.

## 11. Responsive Behavior

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

## 12. Agent Prompt Guide

### Quick Color Reference

```
Rail:           #0c0b0b
Canvas:         #131211
Canvas Soft:    #1a1918
Canvas Tint:    #1c1b1a
Sidebar:        #1a1918  (semantic alias of canvas-soft)
Elevated:       #232220
Hover:          #1f1e1d
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
Accent Tint:    rgba(232,87,42,0.10)
Accent Tint S.: rgba(232,87,42,0.16)
Accent Glow:    rgba(232,87,42,0.05)

Success:        #5fbf85   tint rgba(95,191,133,0.08)
Warning:        #d6a647   tint rgba(214,166,71,0.08)
Danger:         #e0635a   tint rgba(224,99,90,0.09)
Info:           #8e8eb5   tint rgba(142,142,181,0.07)
Neutral:        #7a7a80   tint rgba(122,122,128,0.06)

Scrim:          rgba(0,0,0,0.5)         (--overlay-scrim)
Ghost Hover:    rgba(255,255,255,0.06)  (--overlay-ghost-hover)
Selection:      rgba(232,87,42,0.28)    (--overlay-selection)
Tint Formula:   <signal-color> at 6–10% alpha for bg, full color for text
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

> Container: bg `#0E0E0F` (canvas-deep), radius 12px, padding 20px. Font: JetBrains Mono 14px, line-height 1.6. Prompt `$ ` in `#E8572A`, command in `#E5E5E7`. Copy button: absolute top-right, ghost, tertiary icon → accent on hover, checkmark swap for 1.5s on copy success. Optional language eyebrow top-left.

**"Create a filter toolbar"**

> Horizontal flex, 8px gap. Active pill: bg `#E8572A`, white text, radius 20px, height 32px, padding 6px 14px. Inactive pills: border 1px `#3C3A39`, text `#8E8E93`. Dropdown filters: pill, border `#3C3A39`, chevron. Active dropdown: bg `#E8572A1F`, border `#E8572A`, text `#E8572A`. Search input: bg `#2E2C2B`, radius 8px, height 36px, search icon `#636366`. Secondary button: border 1px `#3C3A39`, Inter 14px Medium, icon + text.

**"Create a chat conversation"**

> User message: right-aligned, bg `#2E2C2B`, radius 12px, padding 16px 20px, Inter 14px `#E5E5E7`. "YOU" label above in mono 11px uppercase `#636366`. Agent message: left-aligned, no bubble. Agent name in JetBrains Mono 11px uppercase with status dot. Body in Inter 14px `#8E8E93`. Tool calls: bg `#1E1C1B` cards, border `#3C3A39`, terminal icon, tool name, file path, status badge right-aligned.

### Implementation Checklist

1. **Fonts loaded (runtime):** Inter Variable (400 + 510 + 600), JetBrains Mono (400 + 500 + 600). Site adds Playfair Display + NuixyberNext via its own Next.js font stack.
2. **Canvas background:** `--canvas` (`#131211`) on `<body>`. `color-scheme: dark` hardcoded. `.dark` forced on `RootProvider` with `enabled: false`.
3. **Flat depth:** only `--shadow-overlay` (modals / sheets) and `--highlight` (active rim). Every other surface stays flat with a 1 px ring on `--line-soft`.
4. **Buttons:** `--radius-md` (8 px), heights 22 / 26 / 30 (Inter 12 px / 510 / -0.005em). Never pill.
5. **Badges:** `--radius-mono-badge` (6 px), 22 px height, signal tint bg, JetBrains Mono 10.5 px / 500 / 0 (Pill `--mono`). Status pill default uses Inter 11 px / 510 sentence case.
6. **Inputs:** `--radius-md` (8 px), bg `--elevated`, border 1 px `--line`, focus ring `0 0 0 1px var(--line-strong)`.
7. **Cards:** `--radius-lg` (10 px), bg `--canvas-soft`, 1 px `--line` ring. No shadows.
8. **Filter pills (UI):** `--radius-pill`, accent fill (active) or 1 px `--line` (inactive).
9. **Dividers:** 1 px solid `--line`; group bottoms use `--line-soft`. Focus ring uses `--line-strong` (white). No accent focus ring.
10. **Signal colors:** `--accent` `#e8572a`, `--success` `#5fbf85`, `--warning` `#d6a647`, `--danger` `#e0635a`, `--info` `#8e8eb5`, `--neutral` `#7a7a80`. Tinted chips only; never solid banners.
11. **Voice:** operator-first, dry, no emoji, no "we". Sentence case copy by default; UPPERCASE only on sidebar labels, table heads, run-cells, and `Eyebrow case="upper"`.
12. **Icons:** Lucide, stroke 2, 12–20 px inline, 48 px for empty-state only. Accent inside icon wells, `currentColor` elsewhere.
13. **Reduced motion** — the universal-selector `@media (prefers-reduced-motion: reduce)` block in `tokens.css` zeros animation/transition/scroll-behavior on `*, *::before, *::after`. Never bypass with `!important` or per-component opt-in.
