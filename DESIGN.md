# Design System: AGH

**Agent Operating System** — one visual language across the runtime daemon, the shared UI kit, and the marketing + docs site.

AGH ships as three surfaces that must feel like one product:

1. **AGH Runtime** — the local daemon, operator UI (`web/`) and CLI. Sessions, memory, skills, workspaces, automation, bridges, observability.
2. **AGH Network** — `agh-network/v0`, the seven-kind wire protocol (`greet`, `whois`, `say`, `direct`, `recipe`, `receipt`, `trace`) over NATS + JSON.
3. **packages/site** — the marketing landing + Fumadocs MDX docs at `agh.network` with two trees (`/runtime/*`, `/protocol/*`).

The canonical token source is [`packages/ui/src/tokens.css`](packages/ui/src/tokens.css). The canonical reference extraction and UI kits live in [`docs/design/design-system/`](docs/design/design-system/).

## 1. Visual Theme & Atmosphere

AGH is a control surface for running real agent work. The aesthetic is **warm dark operator** — a near-black canvas (#141312) with a slight warm tint away from pure black, broken by a single operator-orange accent (#E8572A). Depth is layered surfaces, not shadows. Type does the heavy lifting: Inter for UI, Playfair Display as an editorial display serif for the marketing hero, JetBrains Mono for all metadata, NuixyberNext for the brand wordmark.

This is a **flat depth model**. No gradients on content, no glassmorphism outside the sticky header, no decorative texture. Depth comes from three background steps (canvas → surface → elevated) and 1px hairline dividers. Color is signal, never decoration — accent means _act_, green means _stable_, red means _stop_, yellow means _caution_, purple means _informational_.

**Key qualities:**

- **Warm, not neutral** — the whole gray ramp is tinted warm (`#141312`, `#1E1C1B`, `#2E2C2B`), not pure iOS gray. Never cool or bluish.
- **Dark mode only** — `color-scheme: dark` is hardcoded. There is no light mode. Assets must never render on a white background.
- **Monochromatic + accent** — a single warm orange (#E8572A) is the only color that breaks the neutral ramp. Semantic colors only appear as 15%-tinted chips, never as solid banners.
- **Editorial calm, operator density** — marketing surfaces use generous Playfair Display headings; operator UI stays dense with Inter + mono metadata. Same tokens, two rhythms.
- **Operational, not decorative** — every color, every chip, every icon carries meaning. No ornament without function.

## 2. Color Palette & Roles

All colors are hex. No OKLCH, no color-mix at the token level (only in component hover states).

### Backgrounds

| Token                                           | Value     | Role                                                            |
| ----------------------------------------------- | --------- | --------------------------------------------------------------- |
| **Canvas** `--color-canvas`                     | `#141312` | Primary app + site background — warm near-black                 |
| **Canvas Deep** `--color-canvas-deep`           | `#0E0E0F` | Code blocks, deep sections (marketing `background="deep"`)      |
| **Surface** `--color-surface`                   | `#1E1C1B` | Cards, sidebar, panels, modals                                  |
| **Surface Panel** `--color-surface-panel`       | `#181716` | Alternate panel fill (docs sidebar, subtle separators)          |
| **Surface Elevated** `--color-surface-elevated` | `#2E2C2B` | Popovers, icon wells inside cards, search inputs, hover targets |
| **Divider** `--color-divider` / `--color-line`  | `#3C3A39` | 1px borders, separators, input outlines — the universal line    |
| **Hover** `--color-hover`                       | `#353332` | Hover state for neutral interactive surfaces                    |
| **Disabled** `--color-disabled`                 | `#4A4847` | Disabled backgrounds and elements                               |

### Text

| Token                                  | Value     | Role                                      |
| -------------------------------------- | --------- | ----------------------------------------- |
| **Primary** `--color-text-primary`     | `#E5E5E7` | Headings, titles, high-emphasis content   |
| **Secondary** `--color-text-secondary` | `#8E8E93` | Body copy, descriptions, helper text      |
| **Tertiary** `--color-text-tertiary`   | `#636366` | Placeholders, disabled text, low-emphasis |
| **Label** `--color-text-label`         | `#98989D` | Meta labels, mono eyebrows on dark fills  |

### Accent & Semantic

Color is signal. Each accent has exactly one meaning.

| Token                                     | Value       | Signal               | Usage                                                                 |
| ----------------------------------------- | ----------- | -------------------- | --------------------------------------------------------------------- |
| **Accent** `--color-accent`               | `#E8572A`   | **Action / Primary** | CTAs, primary buttons, active pills, focus rings, links, `$ ` prompts |
| **Accent Ink** `--color-accent-ink`       | `#17110F`   | Text on accent fill  | Used when text sits on a solid accent background                      |
| **Accent Hover** `--color-accent-hover`   | `#D14E25`   | Action pressed       | Hover / pressed state for accent fills                                |
| **Accent Strong** `--color-accent-strong` | `#F6874F`   | Highlight accent     | Rare high-emphasis accent, e.g. inline code on dark panels            |
| **Accent Dim** `--color-accent-dim`       | `#E8572A59` | Muted accent outline | ~35% alpha — focus rings, subtle active borders                       |
| **Success** `--color-success`             | `#30D158`   | Stable / Live        | Connected status, running agents, online indicator                    |
| **Danger** `--color-danger`               | `#FF453A`   | Error / Destructive  | Errors, disconnected, destructive buttons, kill switches              |
| **Warning** `--color-warning`             | `#FFD60A`   | Caution / Pending    | Pending states, degraded status                                       |
| **Info** `--color-info`                   | `#BF5AF2`   | Informational        | Info chips, secondary categorization                                  |

### Tint Formula

Status badges and kind chips use 15% opacity of the semantic color as background with the full-color text.

| Semantic Color | Tint Token             | Value       | Text      |
| -------------- | ---------------------- | ----------- | --------- |
| Accent         | `--color-accent-tint`  | `#E8572A26` | `#E8572A` |
| Success        | `--color-success-tint` | `#30D15826` | `#30D158` |
| Danger         | `--color-danger-tint`  | `#FF453A26` | `#FF453A` |
| Warning        | `--color-warning-tint` | `#FFD60A26` | `#FFD60A` |
| Info           | `--color-info-tint`    | `#BF5AF226` | `#BF5AF2` |
| Neutral        | `--color-neutral-tint` | `#63636626` | `#636366` |

### Overlays

| Purpose           | Value                                             | Notes                               |
| ----------------- | ------------------------------------------------- | ----------------------------------- |
| **Modal scrim**   | `rgba(0, 0, 0, 0.5)`                              | Dialog backdrop                     |
| **Ghost hover**   | `rgba(255, 255, 255, 0.06)`                       | Ghost button hover on dark surfaces |
| **Selection**     | `rgba(232, 87, 42, 0.28)`                         | Text selection — warm accent tint   |
| **Sticky header** | `rgba(20, 19, 18, 0.92)` + `backdrop-blur-xl`     | The only place blur is allowed      |
| **Hero mesh**     | `/hero-bg.png` @ 20% opacity + `mix-blend-screen` | Warm mesh, never a gradient         |

## 3. Typography Rules

### Font Families

| Role                | Typeface             | Fallback                                             | Usage                                                                                       |
| ------------------- | -------------------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **Primary (Sans)**  | **Inter Variable**   | Inter, -apple-system, BlinkMacSystemFont, sans-serif | Body, UI, docs headings, buttons, everything readable                                       |
| **Display (Serif)** | **Playfair Display** | Inter Variable, serif                                | **Marketing only** — home `h1`/`h2` (`.site-home` scope)                                    |
| **Mono**            | **JetBrains Mono**   | 'Courier New', monospace                             | Labels, badges, eyebrows, code, counters, protocol strings — always uppercase when non-code |
| **Wordmark**        | **NuixyberNext**     | var(--font-sans)                                     | The literal string `agh` in the header Logo — nothing else                                  |

### Type Scale

Marketing uses a fluid `clamp()` ramp; docs and UI use fixed sizes.

| Role                     | Font             | Size                           | Weight | Line Height | Letter Spacing | Notes                                             |
| ------------------------ | ---------------- | ------------------------------ | ------ | ----------- | -------------- | ------------------------------------------------- |
| **Hero H1 (site-home)**  | Playfair Display | `clamp(2.8rem, 6.5vw, 5.4rem)` | 400    | 0.96        | -0.035em       | Editorial display — marketing only                |
| **Marketing H2**         | Playfair Display | `clamp(2.2rem, 4.6vw, 3.6rem)` | 400    | 1.02        | -0.03em        | Section headers on landing                        |
| **Docs H1**              | Inter            | `clamp(2.55rem, 4.7vw, 4rem)`  | 600    | 0.94        | -0.05em        | Doc masthead — distinct from marketing hero       |
| **Docs H2**              | Inter            | `clamp(1.7rem, 3vw, 2.45rem)`  | 600    | 1.05        | -0.035em       | Has `border-top 1px divider` + `padding-top 1rem` |
| **H3**                   | Inter            | 20px (`1.25rem`)               | 500    | 1.2         | -0.02em        | Card titles, subsection headers                   |
| **Page Title (UI)**      | Inter            | 20px                           | 700    | 28px        | -0.01em        | Operator UI top-level page titles                 |
| **Card Title**           | Inter            | 16px                           | 600    | 24px        | -0.01em        | Panel and card headings in UI                     |
| **Item Title**           | Inter            | 15px                           | 500    | 22px        | —              | List item titles, session names                   |
| **Lead**                 | Inter            | 18px (`1.125rem`)              | 400    | 1.6         | —              | Hero sub-lead, section leads — max-width 58ch     |
| **Body**                 | Inter            | 16px (`1rem`)                  | 400    | 1.5–1.7     | —              | Default reading text                              |
| **Body (Docs)**          | Inter            | 16px                           | 400    | 1.8         | —              | Long-form docs — max-width 72ch                   |
| **Small Body**           | Inter            | 13px (`0.8125rem`)             | 400    | 18px        | —              | Helper text, captions, meta                       |
| **Metric Value**         | Inter            | 24px                           | 700    | 30px        | -0.02em        | Large dashboard numbers                           |
| **Button Text**          | Inter            | 14px                           | 500    | 18px        | —              | Primary/secondary button labels                   |
| **Ghost Button**         | Inter            | 13px                           | 500    | 16px        | —              | Ghost/compact button labels                       |
| **Eyebrow**              | JetBrains Mono   | 11px                           | 600    | 16px        | 0.06em         | Uppercase — section headers, meta labels          |
| **Doc Masthead Eyebrow** | JetBrains Mono   | 12px                           | 600    | 16px        | 0.16em         | Wider tracking for the `/runtime/*` doc masthead  |
| **Badge Text**           | JetBrains Mono   | 10px                           | 600    | 12px        | 0.08em         | Status badges, kind chips — uppercase             |
| **Mono Badge**           | JetBrains Mono   | 11px                           | 500    | 14px        | 0.06em         | Inline mono pills (agent IDs, protocol names)     |
| **Inline Code**          | JetBrains Mono   | 0.9em                          | 400    | inherit     | —              | In-flow code tokens                               |
| **Brand Wordmark**       | NuixyberNext     | ~24px @ header                 | 400    | 1           | —              | Lowercase `agh` + neighboring `ALPHA` chip (mono) |

### Typography Principles

- **Playfair Display is scoped to `.site-home` only.** Docs and operator UI never use the serif.
- **Docs H1 is heavier sans** (600, -0.05em tracking) so it reads as reference material, not marketing.
- **All eyebrows are uppercase JetBrains Mono** with 0.06em tracking (0.16em on the doc masthead). Lowercase mono is reserved for inline code and protocol strings like `agh-network/v0`.
- **Negative tracking on headings** (-0.02em to -0.05em) tightens large type; body stays at default tracking.
- **No bold body text.** Max weight for body is Medium 500.
- **P&L / semantic values** — positive in `#30D158`, negative in `#FF453A`, neutral in `#E5E5E7`. The color IS the information.
- **Selection color** — `rgba(232,87,42,0.28)`. Warm accent, not default blue.

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

| Token                                 | Value  | Usage                                                           |
| ------------------------------------- | ------ | --------------------------------------------------------------- |
| `--radius-chip`                       | 5px    | Kind chips (protocol kinds)                                     |
| `--radius-mono-badge` / `--radius-sm` | 6px    | Mono badges, status badges, tags, counts                        |
| `--radius-md` / `--radius`            | 8px    | Buttons, inputs, avatars, tool call cards, small UI             |
| `--radius-lg`                         | 12px   | Cards, modals, containers (alias: `--radius-diagram`)           |
| `--radius-diagram`                    | 12px   | Cards, code blocks, diagrams — marketing feature card canonical |
| `--radius-xl`                         | 20px   | Filter pills, large capsule filters                             |
| `rounded-full` (9999px)               | 9999px | Header nav pills, search trigger, GH button, status dots        |

No extreme rounding. CTAs are 8px (`rounded-lg`), **never pill**.

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

AGH uses a **flat depth model** — no traditional shadow system. Four background levels create all necessary visual hierarchy.

```
Canvas Deep (#0E0E0F) → Canvas (#141312) → Surface (#1E1C1B) → Elevated (#2E2C2B)
```

Each step is a clear lightness increase in the warm gray scale. Borders (`#3C3A39`) provide additional separation when background contrast alone isn't sufficient.

### Depth Patterns

| Pattern                     | How it works                                                                                |
| --------------------------- | ------------------------------------------------------------------------------------------- |
| **Card on canvas**          | Surface (#1E1C1B) on canvas (#141312) — needs 1px divider border for clear edge             |
| **Nested card / icon well** | Elevated (#2E2C2B) inside surface (#1E1C1B) — e.g. search input, icon well                  |
| **Deep panel on landing**   | Canvas Deep (#0E0E0F) on canvas — used for code blocks and `SectionFrame background="deep"` |
| **Selected list item**      | Elevated (#2E2C2B) + left accent bar (#E8572A)                                              |
| **Hover state**             | Hover (#353332) replaces the current surface                                                |
| **Divider**                 | 1px solid #3C3A39 — between list items, sections, sidebar regions                           |
| **Border emphasis**         | 1px solid #E8572A — focused inputs, active dropdown filters                                 |
| **Warm hover on card**      | Border → `color-mix(in srgb, #E8572A 40%, #3C3A39)` — the only card hover                   |
| **Comparison highlight**    | `border-l-4 #E8572A` + `#E8572A26` bg — ONLY on the comparison table                        |

Ring outlines (`ring-1 ring-foreground/10`) and `shadow-xs` on shadcn inputs are the only places shadows appear. No drop shadows, no layered shadows, no glows.

## 7. Voice & Content

AGH copy has a specific, operator-first voice. Design without the voice is incomplete.

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
- **Fast.** 150ms for hover/focus, 200ms for panel transitions.
- **No entrance animations.** Content appears immediately; no staggered reveals, no scroll-triggered fades.
- **No bounce / spring easing.** Use `ease-out` or `ease-in-out` only.
- **Reduced motion is respected globally** — `@media (prefers-reduced-motion: reduce)` zeroes all durations and animations.

### Transitions

| Element           | Property         | Duration | Easing                                    |
| ----------------- | ---------------- | -------- | ----------------------------------------- |
| Button hover      | background-color | 150ms    | ease-out                                  |
| Link hover        | color            | 150ms    | ease                                      |
| Border focus      | border-color     | 150ms    | ease-out                                  |
| List item hover   | background-color | 150ms    | ease-out                                  |
| Card hover border | border-color     | 150ms    | ease-out                                  |
| Sidebar collapse  | width            | 200ms    | ease-in-out                               |
| Modal enter       | opacity          | 200ms    | ease-out                                  |
| Tooltip           | opacity          | 100ms    | ease-out                                  |
| Button active     | transform        | —        | `translate-y-px` (1px nudge, no duration) |

No `transform` transitions on hover. No scale, no lift, no translate on card/button hover.

### Loading States

- **Shimmer:** `@utility animate-shimmer` — 200% background shift, 2s, infinite. For skeleton loaders.
- **Spinner:** rotating icon for inline loading.
- **Pulse:** `animate-ping` on a 1.5px success dot — signals live state ("Shipped today"). Used sparingly.

## 10. Do's and Don'ts

### Do

- **Use the flat depth model** — bg lightness (`#0E0E0F → #141312 → #1E1C1B → #2E2C2B`) for hierarchy. No gradients on content. No stacked shadows.
- **Use warm values.** All grays in this system are tinted warm. Never substitute cool iOS grays.
- **Use JetBrains Mono for all metadata** — section eyebrows, badge text, timestamps, protocol strings. Always uppercase (except protocol identifiers, which are lowercase mono) with 0.06–0.16em tracking.
- **Use Playfair Display only inside `.site-home`.** Marketing hero and section H2s only.
- **Use Inter everywhere else** — including docs H1/H2. Docs headings use weight 600, tracking -0.05em / -0.035em.
- **Use semantic color consistently** — accent = action, green = stable/positive, red = error/negative, yellow = warning/pending, purple = informational.
- **Use the 15%-tint formula** — 15% opacity of the semantic color as background, full color as text.
- **Keep buttons at 36px height** (default) or 28px (compact/ghost) or 44px (marketing `lg` CTA) — no other heights.
- **Use 1px solid `#3C3A39` borders** as the universal divider. `1.5px solid #E8572A` for focus. `color-mix(#E8572A 40%, #3C3A39)` for card hover.
- **Left-align content** — page titles, list items, descriptions start from the same left edge.
- **Use colored semantic values** — positive green, negative red, neutral white. The color IS the information.
- **Respect reduced motion** — the global CSS already zeros durations; never override.

### Don't

- **Don't use shadows.** Flat depth only. No `box-shadow`, no drop-shadow, no elevation shadows. `ring-1 ring-foreground/10` and `shadow-xs` on shadcn inputs are the only exceptions.
- **Don't use gradients** — no content gradients, no gradient text, no gradient borders. The hero mesh PNG is the single exception.
- **Don't use blur / glass effects** outside the sticky site header and the hero signal cards. No backdrop-blur elsewhere.
- **Don't use decorative texture** — no hatching, no grain, no noise overlays outside the hero mesh.
- **Don't use emoji. Anywhere.** Not in UI, docs, marketing, commits, or copy samples.
- **Don't switch fonts.** Inter for readable content. Playfair Display only in `.site-home`. JetBrains Mono for metadata. NuixyberNext only for the `agh` wordmark. No Bricolage, no Geist, no alternative display face.
- **Don't add new semantic colors.** The five-signal system (accent, success, danger, warning, info) is complete.
- **Don't use large decorative icons** — icons are 12–20px inline. 48px is reserved for empty-state illustrations.
- **Don't make badges with borders** — status badges and kind chips use tinted backgrounds only. Neutral / mono badges use borders.
- **Don't use pure white (`#FFFFFF`) for backgrounds** — reserved only for button text on accent fill.
- **Don't use `rounded-full` for CTAs.** CTAs are 8px. Only header nav pills, search trigger, status dots use `rounded-full`.
- **Don't proliferate the left-colored-border pattern.** It exists once — the highlighted comparison row. Selected list items use a 3px accent bar on a surface background, not a border.
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
Canvas Deep:    #0E0E0F
Canvas:         #141312
Surface:        #1E1C1B
Surface Panel:  #181716
Elevated:       #2E2C2B
Divider:        #3C3A39
Hover:          #353332
Disabled:       #4A4847

Text Primary:   #E5E5E7
Text Secondary: #8E8E93
Text Tertiary:  #636366
Text Label:     #98989D

Accent:         #E8572A
Accent Ink:     #17110F   (text on accent fill)
Accent Hover:   #D14E25
Accent Strong:  #F6874F
Accent Dim:     #E8572A59 (~35% alpha)
Accent Tint:    #E8572A26 (~15% alpha)

Success:        #30D158
Danger:         #FF453A
Warning:        #FFD60A
Info:           #BF5AF2

Scrim:          rgba(0,0,0,0.5)
Ghost Hover:    rgba(255,255,255,0.06)
Selection:      rgba(232,87,42,0.28)
Tint Formula:   <semantic-color> at 15% opacity for bg, full color for text
```

### Quick Type Reference

```
UI / Body:      Inter Variable
Marketing h1/h2: Playfair Display (only inside .site-home)
Docs h1/h2:     Inter, weight 600, tracking -0.05em / -0.035em
Metadata:       JetBrains Mono, uppercase, tracking 0.06em
Wordmark:       NuixyberNext (only for the "agh" lockup)
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

1. **Fonts loaded:** Inter Variable (400–600), Playfair Display (400, 500), JetBrains Mono (400–600), NuixyberNext (400, wordmark only).
2. **Canvas background:** `#141312` on `<body>`. `color-scheme: dark` hardcoded. `.dark` forced on `RootProvider` with `enabled: false`.
3. **Flat depth:** no box-shadows (except `ring-1 ring-foreground/10` and shadcn `shadow-xs`). Depth via bg color + 1px dividers only.
4. **Buttons:** 8px radius, 28px / 36px / 44px heights. Never pill.
5. **Badges:** 6px radius (`--radius-mono-badge`), 22px height, 15%-tinted bg, JetBrains Mono 10px 600 uppercase tracking 0.08em. Kind chips use 5px radius.
6. **Inputs:** 8px radius, 36px height, bg `#2E2C2B`, border `#3C3A39`, focus border `#E8572A`.
7. **Cards:** 12px radius (`--radius-diagram`), bg `#1E1C1B`, border 1px `#3C3A39`, padding 24px (marketing feature card) or 16px 20px (UI metric card).
8. **Filter pills (UI):** 20px radius, 32px height, accent fill (active) or border-only (inactive).
9. **Header nav pills (site):** `rounded-full`, hover/active tint `rgba(232,87,42,0.12)`.
10. **Dividers:** 1px solid `#3C3A39` everywhere. 1.5px solid `#E8572A` for focus.
11. **Semantic colors:** accent `#E8572A`, success `#30D158`, danger `#FF453A`, warning `#FFD60A`, info `#BF5AF2`. Tinted chips only; never solid banners.
12. **Voice:** operator-first, dry, no emoji, no "we". Mono uppercase eyebrows everywhere.
13. **Icons:** Lucide, stroke 2, 12–20px inline, 48px for empty-state only. Accent inside icon wells, `currentColor` elsewhere.
14. **Respect `prefers-reduced-motion`** — do not add animations that bypass the global reset.
