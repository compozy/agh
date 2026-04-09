# Design System: AGH

**Agent Operating System** — command surfaces for live agent orchestration.

## 1. Visual Theme & Atmosphere

AGH is a control surface for orchestrating AI agent sessions: spawning processes, watching event streams, managing permissions, reading transcripts. The interface is built for operators who need information density without visual noise.

The aesthetic is **dark operator** — an iOS-native dark palette (#121212 canvas) with a single warm accent (#E8572A) that cuts through the neutral grays. Surfaces are flat, separated by background lightness rather than shadows. Typography does the heavy lifting: Inter for all readable content, JetBrains Mono for structural metadata. Color is signal, never decoration — accent means _act_, green means _stable_, red means _stop_, yellow means _caution_.

This is a flat depth model. No gradients, no glassmorphism, no decorative texture. Depth comes from three background levels (canvas → surface → elevated) and hairline dividers. The design language borrows from iOS system conventions — the gray scale, the border radii, the status dot patterns — adapted for a dense, data-rich desktop context.

**Key qualities:**

- **Dense, not cramped** — compact mono labels, tight vertical rhythm, generous section separation
- **Flat, not lifeless** — depth through lightness stepping, not shadow stacking
- **Warm signal on cold ground** — #E8572A accent breaks the neutral gray palette with urgency
- **Operational, not decorative** — every color carries meaning; no ornament without function

## 2. Color Palette & Roles

All colors are hex values from the iOS-inspired dark palette. No OKLCH, no color-mix — clean hex tokens.

### Backgrounds

| Token                                   | Value     | Role                                        |
| --------------------------------------- | --------- | ------------------------------------------- |
| **Canvas** `--color-canvas`             | `#121212` | Primary app background — near-black neutral |
| **Surface** `--color-surface`           | `#1C1C1E` | Cards, panels, sidebar, modal backgrounds   |
| **Elevated** `--color-surface-elevated` | `#2C2C2E` | Popovers, elevated cards, search inputs     |
| **Divider** `--color-divider`           | `#3A3A3C` | Borders, separators, input outlines         |
| **Hover** `--color-hover`               | `#333336` | Hover state for interactive surfaces        |
| **Disabled** `--color-disabled`         | `#48484A` | Disabled backgrounds and elements           |

### Text

| Token                                  | Value     | Role                                      |
| -------------------------------------- | --------- | ----------------------------------------- |
| **Primary** `--color-text-primary`     | `#E5E5E7` | Headings, titles, high-emphasis content   |
| **Secondary** `--color-text-secondary` | `#8E8E93` | Body text, descriptions, helper text      |
| **Tertiary** `--color-text-tertiary`   | `#636366` | Placeholders, disabled text, low-emphasis |
| **Label** `--color-text-label`         | `#98989D` | Meta labels, section headers, timestamps  |

### Accent & Semantic

Color is signal in AGH. Each accent has exactly one meaning:

| Token                         | Value     | Signal                  | Usage                                                         |
| ----------------------------- | --------- | ----------------------- | ------------------------------------------------------------- |
| **Accent** `--color-accent`   | `#E8572A` | **Action / Primary**    | CTAs, primary buttons, active filters, focus rings, links     |
| **Accent Hover**              | `#D14E25` | **Action Pressed**      | Hover/pressed state for accent elements                       |
| **Success** `--color-success` | `#30D158` | **Stable / Live**       | Connected status, enabled items, positive P&L, running agents |
| **Danger** `--color-danger`   | `#FF453A` | **Error / Destructive** | Errors, disconnected, rejected, kill switch, negative values  |
| **Warning** `--color-warning` | `#FFD60A` | **Caution / Pending**   | Pending states, partial fills, degraded status                |
| **Info** `--color-info`       | `#BF5AF2` | **Informational**       | Informational badges, secondary categorization                |

### Badge Tint Formula

Status badges use 15% opacity of the semantic color as background:

| Semantic Color    | Background         | Text Color |
| ----------------- | ------------------ | ---------- |
| Accent (#E8572A)  | `#E8572A26` (~15%) | `#E8572A`  |
| Success (#30D158) | `#30D15826` (~15%) | `#30D158`  |
| Danger (#FF453A)  | `#FF453A26` (~15%) | `#FF453A`  |
| Warning (#FFD60A) | `#FFD60A26` (~15%) | `#FFD60A`  |
| Info (#BF5AF2)    | `#BF5AF226` (~15%) | `#BF5AF2`  |
| Neutral (#636366) | `#63636626` (~15%) | `#636366`  |

### Overlay

| Token           | Value                   | Role                          |
| --------------- | ----------------------- | ----------------------------- |
| **Scrim**       | `rgba(0,0,0,0.5)`       | Modal/dialog backdrop overlay |
| **Ghost Hover** | `#FFFFFF0F` (~6% white) | Ghost button hover state      |

## 3. Typography Rules

### Font Families

| Role                 | Typeface           | Fallback                                      | Usage                                                 |
| -------------------- | ------------------ | --------------------------------------------- | ----------------------------------------------------- |
| **Primary (Sans)**   | **Inter**          | -apple-system, BlinkMacSystemFont, sans-serif | Headings, body, prices, all readable content          |
| **Secondary (Mono)** | **JetBrains Mono** | 'Courier New', monospace                      | Labels, badges, counters, metadata — always uppercase |

### Type Scale

| Role             | Font           | Size | Weight      | Line Height | Letter Spacing | Notes                                        |
| ---------------- | -------------- | ---- | ----------- | ----------- | -------------- | -------------------------------------------- |
| **Page Title**   | Inter          | 20px | Bold 700    | 28px        | -0.01em        | Top-level page headings                      |
| **Card Title**   | Inter          | 16px | Semi 600    | 24px        | -0.01em        | Panel and card headings                      |
| **Item Title**   | Inter          | 15px | Medium 500  | 22px        | —              | List item titles, session names              |
| **Body**         | Inter          | 14px | Regular 400 | 20px        | —              | Default reading text, descriptions           |
| **Small Body**   | Inter          | 13px | Regular 400 | 18px        | —              | Helper text, subtext, captions               |
| **Price Large**  | Inter          | 18px | Bold 700    | 26px        | -0.01em        | Metric card hero values                      |
| **Metric Value** | Inter          | 24px | Bold 700    | 30px        | -0.02em        | Large dashboard numbers                      |
| **Price / P&L**  | Inter          | 15px | Semi 600    | 22px        | —              | Inline monetary values, colored by direction |
| **Meta Label**   | JetBrains Mono | 11px | Medium 500  | 16px        | 0.06em         | Section headers, metadata — uppercase        |
| **Badge Text**   | JetBrains Mono | 10px | Semi 600    | 12px        | 0.08em         | Status badges, tags — uppercase              |
| **Button Text**  | Inter          | 14px | Medium 500  | 18px        | —              | Primary/secondary button labels              |
| **Ghost Button** | Inter          | 13px | Medium 500  | 16px        | —              | Ghost/compact button labels                  |
| **Filter Pill**  | Inter          | 14px | Regular 400 | 16px        | —              | Pill filter text in toolbars                 |

### Typography Principles

- **Inter for everything readable** — headings, body, values, buttons. No display font.
- **JetBrains Mono for structural metadata** — labels, badges, counters. Always `uppercase` with wide letter-spacing.
- **Negative tracking on headings** (-0.01em to -0.02em) — tighter setting at larger sizes.
- **No bold body text** — max weight for body is Medium 500. Bold 700 is reserved for page titles and metric values.
- **P&L coloring** — positive values in `#30D158`, negative in `#FF453A`, neutral in `#E5E5E7`.

## 4. Component Stylings

### Buttons

#### Primary

Solid accent fill, white text. The main call-to-action.

| State     | Background | Text      | Border Radius | Height | Padding           |
| --------- | ---------- | --------- | ------------- | ------ | ----------------- |
| Default   | `#E8572A`  | `#FFFFFF` | 8px           | 36px   | 8px 20px          |
| Hover     | `#D14E25`  | `#FFFFFF` | 8px           | 36px   | 8px 20px          |
| Disabled  | `#48484A`  | `#636366` | 8px           | 36px   | 8px 20px          |
| With Icon | `#E8572A`  | `#FFFFFF` | 8px           | 36px   | 8px 20px, gap 6px |
| Compact   | `#E8572A`  | `#FFFFFF` | 8px           | 28px   | 6px 12px          |

#### Danger

Same shape as primary, red fill for destructive actions.

| State    | Background  | Text      |
| -------- | ----------- | --------- |
| Default  | `#FF453A`   | `#FFFFFF` |
| Hover    | Lighter red | `#FFFFFF` |
| Disabled | `#48484A`   | `#636366` |

#### Secondary

Border-only button for secondary actions.

| State    | Background  | Border              | Text                |
| -------- | ----------- | ------------------- | ------------------- |
| Default  | transparent | 1px solid `#3A3A3C` | `#E5E5E7`           |
| Hover    | `#333336`   | 1px solid `#3A3A3C` | `#E5E5E7`           |
| Disabled | transparent | 1px solid `#3A3A3C` | `#636366`           |
| Dropdown | transparent | 1px solid `#3A3A3C` | `#E5E5E7` + chevron |

#### Ghost

Text-only button, no border. Hover reveals subtle background.

| State     | Background              | Text      | Height | Padding      |
| --------- | ----------------------- | --------- | ------ | ------------ |
| Default   | transparent             | `#8E8E93` | 28px   | 6px 12px     |
| Hover     | `#FFFFFF0F` (~6% white) | `#E5E5E7` | 28px   | 6px 12px     |
| Icon Only | transparent             | `#8E8E93` | 28px   | 6px (square) |

#### Pill Toggle / Filter Tabs

Used for filter groups and segmented controls.

| State            | Background  | Border              | Text      | Radius |
| ---------------- | ----------- | ------------------- | --------- | ------ |
| Active           | `#E8572A`   | none                | `#FFFFFF` | 20px   |
| Inactive         | transparent | 1px solid `#3A3A3C` | `#8E8E93` | 20px   |
| Segmented Active | `#E8572A`   | none                | `#FFFFFF` | 20px   |

**Filter pill dimensions:** height 32px, padding 6px 14px, gap 6px.

### Badges & Tags

Compact monospace labels for status, categorization, and metadata. Always JetBrains Mono, uppercase, 10px, 600 weight, 0.08em tracking.

**Dimensions:** height 22px, padding 3px 8px, border-radius 6px.

**Color by semantic role:**

- **Accent badges**: bg `#E8572A26`, text `#E8572A` — submitted, running, active
- **Success badges**: bg `#30D15826`, text `#30D158` — filled, approved, running, healthy
- **Danger badges**: bg `#FF453A26`, text `#FF453A` — rejected, error, critical, halted
- **Warning badges**: bg `#FFD60A26`, text `#FFD60A` — partial, pending, degraded
- **Info badges**: bg `#BF5AF226`, text `#BF5AF2` — info, guardian type
- **Neutral badges**: bg `#63636626`, text `#636366` — cancelled, idle, inactive

### Metric Cards

Three variants, all using surface background with 12px radius.

#### Simple

Label + large value. Compact.

- **Container:** bg `#1C1C1E`, radius 12px, padding 16px 20px, gap 8px
- **Label:** JetBrains Mono, 11px, Medium 500, uppercase, 0.06em tracking, `#636366`
- **Value:** Inter, 24px, Bold 700, -0.02em tracking, `#E5E5E7`
- **Colored values:** positive `#30D158`, negative `#FF453A`, warning `#FFD60A`

#### With Subtext

Adds a secondary line below the value.

- Same as Simple, plus **Subtext:** Inter, 13px, Regular 400, `#8E8E93`

#### With Sparkline

Adds a small inline chart next to the value.

- Same as Simple, value row becomes flex with sparkline SVG aligned right

### Inputs

#### Search Input

- **Container:** bg `#2C2C2E`, radius 8px, height 36px, padding 0 12px, gap 8px
- **Border:** 1px solid `#3A3A3C` (default), 1.5px solid `#E8572A` (focused)
- **Placeholder:** Inter 14px Regular, `#636366`
- **Text:** Inter 14px Regular, `#E5E5E7`
- **Icon:** search icon, `#636366`
- **Disabled:** bg `#1C1C1E`, border `#2C2C2E`, text `#48484A`

#### Dropdown Filter

Pill-shaped dropdown for filter controls.

- **Container:** radius 20px, height 32px, padding 6px 14px, gap 6px
- **Border:** 1px solid `#3A3A3C` (default), 1px solid `#E8572A` (active)
- **Active background:** `#E8572A1F` (~12% accent tint)
- **Text:** Inter 14px, `#8E8E93` (default), `#E8572A` (active)

### Cards & Containers

#### Content Preview Card

- **Container:** bg `#1C1C1E`, radius 12px, padding 16px 20px
- **Title:** Inter 15px Medium 500, `#E5E5E7`
- **Description:** Inter 14px Regular, `#8E8E93`
- **Link:** Inter 14px, `#E8572A`, with arrow suffix

#### Metadata Table

Striped key-value rows.

- **Odd rows:** bg transparent
- **Even rows:** bg `#1C1C1E`
- **Key:** Inter 13px Regular, `#636366`
- **Value:** Inter 14px Medium, `#E5E5E7`
- **Badge values:** rendered as status badges inline

### Chat Components

#### User Message

Right-aligned bubble.

- **Bubble:** bg `#2C2C2E`, radius 12px, padding 16px 20px
- **Text:** Inter 14px Regular, `#E5E5E7`
- **Meta:** "YOU" + timestamp, Inter 13px, `#636366`, right-aligned above bubble

#### Agent Message

Left-aligned, no bubble.

- **Agent label:** dot (semantic color) + agent name (JetBrains Mono 11px uppercase `#98989D`) + timestamp
- **Text:** Inter 14px Regular, `#8E8E93`
- **Numbered lists:** Inter 14px Regular, `#8E8E93`

#### Tool Call Card

Inline card showing tool execution.

- **Container:** bg `#1C1C1E`, radius 8px, padding 10px 16px, border 1px solid `#3A3A3C`
- **Icon:** terminal icon `>_`, `#636366`
- **Tool name:** Inter 14px Medium, `#E5E5E7`
- **File path:** Inter 13px Regular, `#636366`
- **Status badge:** right-aligned, colored dot + label (DONE/RUNNING/ERROR)

#### Chat Input

- **Container:** bg `#1C1C1E`, radius 12px, padding 12px 16px, border 1px solid `#3A3A3C`
- **Focused:** border 1px solid `#E8572A`
- **Placeholder:** Inter 14px Regular, `#636366`
- **Send button:** 36px circle, bg `#E8572A`, white send icon

### Status Indicators

Inline dot + label patterns.

| Status                 | Dot Color | Label           | Usage                       |
| ---------------------- | --------- | --------------- | --------------------------- |
| Connected / Online     | `#30D158` | "Connected"     | System footer, active items |
| Disconnected / Offline | `#FF453A` | "Disconnected"  | System footer, error states |
| Enabled / Active       | `#30D158` | —               | Skill list, session tabs    |
| Degraded               | `#FF453A` | "Degraded"      | Agent status                |
| Dream Status           | `#30D158` | "Dream: 3h ago" | Knowledge page header       |
| Running                | `#E8572A` | "RUNNING"       | Tool call status            |

**Dot size:** 8px circle (width/height), with 2px ring spacing.

### Sidebar

#### Structure

- **Workspace icon rail:** 40px wide, left edge. Circle avatars (32px) with single letter.
  - App logo: `#E8572A` background, white letter
  - Active workspace: `#E8572A` ring border
  - Inactive: `#2C2C2E` background, `#8E8E93` letter
  - Hover: `#333336` background
  - New: `#2C2C2E` background, "+" icon
- **Sidebar panel:** bg `#1C1C1E`, width ~220px, full height

#### Section Header

JetBrains Mono 11px Medium uppercase, `#636366`, with optional count right-aligned.

#### Agent List Item

- **Row:** height ~36px, padding 8px 12px
- **Avatar:** 24px circle with letter, colored per agent type
- **Name:** Inter 14px Medium, `#E5E5E7`
- **Count + chevron:** `#636366`, right-aligned
- **Active indicator:** 3px left accent bar `#E8572A`

#### Nav Item

- **Row:** padding 8px 12px
- **Icon:** 16px, `#636366` (default), `#E5E5E7` (active)
- **Label:** Inter 14px Regular, `#8E8E93` (default), `#E5E5E7` (active)
- **Active indicator:** 3px left accent bar `#E8572A`

#### System Status Footer

- **Dot + label:** Connected/Disconnected status
- **Version:** `#636366`, right-aligned
- **Settings:** gear icon, `#636366`

### Page Layout

#### Page Header Bar

- **Icon + title:** 16px icon + Inter 16px Semi 600
- **Count:** bg `#2C2C2E`, radius 6px, Inter 13px
- **Tab pills:** pill toggle pattern (INSTALLED/MARKETPLACE or ALL/GLOBAL/WORKSPACE)
- **Right status:** dot + label (e.g., "Dream: 3h ago")

#### Breadcrumb / Session Header

- **Path:** JetBrains Mono 11px Medium uppercase, `#98989D`
- **Session selector:** dot + agent name + session dropdown + "+" add button

#### Detail Panel Header

- **Title:** Inter 16px Semi 600, `#E5E5E7`
- **Version:** Inter 13px Regular, `#636366`
- **Source badge:** (e.g., BUNDLED) in success green badge
- **Status line:** dot + "Enabled" + path in `#636366`

#### Empty State

Centered illustration + text.

- **Icon:** 48px, `#636366`, centered
- **Title:** Inter 15px Medium, `#8E8E93`
- **Description:** Inter 13px Regular, `#636366`

### List Items

#### Skill List Item

- **Row:** padding 8px 0, border-bottom 1px solid `#2C2C2E`
- **Status dot:** 8px, semantic color
- **Name:** Inter 14px Medium, `#E5E5E7`
- **Version:** Inter 13px Regular, `#636366`, right-aligned
- **Selected:** bg `#2C2C2E` with left accent bar

#### Knowledge List Item

- **Row:** padding 12px 0
- **Title:** Inter 15px Medium, `#E5E5E7`
- **Description:** Inter 13px Regular, `#8E8E93`
- **Date:** Inter 13px Regular, `#636366`, right-aligned
- **Tags:** semantic badges below description
- **Selected:** bg `#2C2C2E` with left accent bar

#### Marketplace Row

- **Row:** bg `#1C1C1E`, radius 8px, padding 12px 16px
- **Name:** Inter 15px Medium, `#E5E5E7`
- **Author + version:** Inter 13px Regular, `#636366`
- **Tags:** neutral border badges (radius 6px, border `#3A3A3C`)
- **Downloads:** Inter 13px Regular, `#636366` + download icon
- **Action:** INSTALL (accent pill) or INSTALLED (neutral pill)

## 5. Layout Principles

### Spacing Scale (Base: 4px)

| Token       | Value | Usage                                           |
| ----------- | ----- | ----------------------------------------------- |
| **space-1** | 4px   | Icon-label gap, tight inner padding             |
| **space-2** | 8px   | Tag padding, inline element gap, badge internal |
| **space-3** | 12px  | List row padding, stacked element gap           |
| **space-4** | 16px  | Standard card padding, section gap              |
| **space-5** | 20px  | List item gap, card inline padding              |
| **space-6** | 24px  | Card outer padding, major section gap           |
| **space-8** | 32px  | Section-level spacing                           |

### Border Radius Scale

| Token            | Value | Usage                                                       |
| ---------------- | ----- | ----------------------------------------------------------- |
| `--radius-sm`    | 6px   | Tags, badges, calendar cells, counts                        |
| `--radius-md`    | 8px   | Buttons, inputs, avatars, tool call cards, marketplace rows |
| `--radius-lg`    | 12px  | Cards, modals, containers, metric cards, chat bubbles       |
| `--radius-pill`  | 20px  | Toggle pills, capsule filter shapes, dropdown filters       |
| `--radius-round` | 50%   | Status dots, avatar circles                                 |

### Elevation (Flat Depth Model)

No box-shadows. Depth is communicated purely through background lightness.

| Level | Name             | Background        | Shadow | Usage                                    |
| ----- | ---------------- | ----------------- | ------ | ---------------------------------------- |
| 0     | **Canvas**       | `#121212`         | none   | Page background                          |
| 1     | **Card / Modal** | `#1C1C1E`         | none   | Cards, sidebar, panels, modals           |
| 2     | **Elevated**     | `#2C2C2E`         | subtle | Popovers, search inputs, elevated panels |
| —     | **Scrim**        | `rgba(0,0,0,0.5)` | —      | Modal/dialog backdrop                    |

### Grid & Layout

- **Sidebar:** workspace rail (40px) + panel (~220px) = ~260px total
- **Content area:** flex-1, left-aligned
- **Three-panel layout:** sidebar + list panel + detail panel (Skills, Knowledge pages)
- **Metric grids:** 4-column row of equal-width cards with 16px gap
- **List layouts:** full-width rows with consistent padding, border-bottom separators

### Whitespace Philosophy

- **Flat hierarchy, clear grouping** — sections separated by 24-32px gaps, items within by 8-12px
- **Left-aligned, not centered** — sidebar and content align to left edge
- **Dense but not crowded** — 36px interactive heights, 22px badge heights, 8px inline gaps

## 6. Depth & Elevation

AGH uses a **flat depth model** — no traditional shadow system. The three background levels create all necessary visual hierarchy.

```
Canvas (#121212) → Surface (#1C1C1E) → Elevated (#2C2C2E)
```

Each step is a clear lightness increase in the gray scale. Borders (`#3A3A3C`) provide additional separation when background contrast alone isn't sufficient.

### Depth Patterns

| Pattern                | How it works                                                                       |
| ---------------------- | ---------------------------------------------------------------------------------- |
| **Card on canvas**     | Surface bg (#1C1C1E) on canvas (#121212) — enough contrast, no border needed       |
| **Nested card**        | Elevated bg (#2C2C2E) inside surface (#1C1C1E) — e.g., search input inside sidebar |
| **Selected list item** | Elevated bg (#2C2C2E) + left accent bar (#E8572A)                                  |
| **Hover state**        | Hover bg (#333336) replaces current surface                                        |
| **Divider**            | 1px solid #3A3A3C — between list items, sections, sidebar regions                  |
| **Border emphasis**    | 1px solid #E8572A — focused inputs, active dropdown filters                        |

## 7. Do's and Don'ts

### Do

- **Use the flat depth model** — bg lightness (#121212 → #1C1C1E → #2C2C2E) for hierarchy. No gradients, no stacked shadows.
- **Use JetBrains Mono for all metadata** — section headers, badge text, timestamps. Always uppercase with letter-spacing (0.06em-0.08em).
- **Use semantic color consistently** — accent = action, green = stable/positive, red = error/negative, yellow = warning/pending.
- **Use the badge tint formula** — 15% opacity of semantic color as background, full color as text.
- **Keep buttons at 36px height** (default) or 28px (compact/ghost) — no other heights.
- **Use 1px solid #3A3A3C borders** — the universal divider. 1.5px solid #E8572A for focus/active.
- **Left-align content** — page titles, list items, descriptions all start from the same left edge.
- **Use colored P&L values** — positive green, negative red, neutral white. The color IS the information.

### Don't

- **Don't use shadows** — this is a flat depth model. No box-shadow, no drop-shadow, no elevation shadows.
- **Don't use gradients** — no background gradients, no gradient text, no gradient borders.
- **Don't use blur/glass effects** — no backdrop-blur, no frosted glass, no glassmorphism.
- **Don't use decorative texture** — no hatching, no grain, no noise overlays.
- **Don't use a display/heading font** — Inter handles all sizes from 13px to 24px. No Bricolage, no Geist, no separate display face.
- **Don't add new semantic colors** — the five-signal system (accent, success, danger, warning, info) is complete.
- **Don't use large decorative icons** — icons are 14-16px inline. No 48px+ hero icons except empty states.
- **Don't make badges with borders** — badges use tinted backgrounds only, no border. Neutral tags use borders.
- **Don't use pure white (#FFFFFF) for backgrounds** — reserved only for button text on accent fill.
- **Don't use rounded-full (999px) for buttons** — buttons use 8px radius. Only filter pills use 20px radius.

## 8. Responsive Behavior

### Breakpoints

Desktop-first layout, optimized for 1440px wide screens.

- **Sidebar:** fixed width, does not collapse on desktop. On narrow viewports, consider icon-rail-only mode (40px).
- **Three-panel:** list panel and detail panel share the content area. On narrow viewports, stack as full-width views with back navigation.
- **Metric cards:** 4-column grid. On narrow viewports, collapse to 2-column or stack.
- **Filter groups:** horizontal flex-wrap. Pills and dropdowns wrap to second line when space is constrained.

### Touch Targets

- **Minimum interactive height:** 28px (ghost/compact buttons)
- **Standard interactive height:** 36px (buttons, inputs, list items)
- **Filter pills:** 32px height for comfortable interaction
- **Sidebar nav items:** full-width hit area with 8px 12px padding
- **Status dots:** 8px visual but wrapped in larger clickable area when interactive

### Sidebar Modes

- **Expanded:** icon rail (40px) + panel (~220px). Agent list, nav items, system footer visible.
- **Collapsed:** icon rail only (40px). Workspace circles visible, panel content hidden.

## 9. Motion & Animation

### Principles

- **Minimal, purposeful motion** — transitions serve state changes, not decoration
- **Fast interactions** — 150ms for hover/focus transitions, 200ms for panel transitions
- **No entrance animations** — content appears immediately, no staggered reveals
- **No bounce/spring easing** — use ease-out or ease-in-out only

### Transitions

| Element          | Property         | Duration | Easing      |
| ---------------- | ---------------- | -------- | ----------- |
| Button hover     | background-color | 150ms    | ease-out    |
| Border focus     | border-color     | 150ms    | ease-out    |
| List item hover  | background-color | 150ms    | ease-out    |
| Sidebar collapse | width            | 200ms    | ease-in-out |
| Modal enter      | opacity          | 200ms    | ease-out    |
| Tooltip          | opacity          | 100ms    | ease-out    |

### Loading States

- **Shimmer:** gradient sweep animation on skeleton elements (2s, ease-in-out, infinite)
- **Spinner:** rotating icon for inline loading indicators
- **Status dot pulse:** subtle opacity pulse for "reconnecting" or "running" states

## 10. Agent Prompt Guide

### Quick Color Reference

```
Canvas:         #121212
Surface:        #1C1C1E
Elevated:       #2C2C2E
Divider:        #3A3A3C
Hover:          #333336
Disabled:       #48484A

Text Primary:   #E5E5E7
Text Secondary: #8E8E93
Text Tertiary:  #636366
Text Label:     #98989D

Accent:         #E8572A
Accent Hover:   #D14E25
Success:        #30D158
Danger:         #FF453A
Warning:        #FFD60A
Info:           #BF5AF2

Scrim:          rgba(0,0,0,0.5)
Ghost Hover:    rgba(255,255,255,0.06)
Badge Tint:     <semantic-color> at 15% opacity
```

### Example Component Prompts

**"Create a session list panel"**

> Surface bg (#1C1C1E). Section header in JetBrains Mono 11px Medium uppercase #636366 with count right-aligned. List items: 8px status dot + Inter 14px Medium name + Inter 13px version right-aligned. Selected item: bg #2C2C2E with 3px left accent bar #E8572A. Border-bottom 1px #3A3A3C between items.

**"Create a metric dashboard row"**

> 4-column grid, 16px gap. Each card: bg #1C1C1E, radius 12px, padding 16px 20px. Label: JetBrains Mono 11px Medium uppercase #636366. Value: Inter 24px Bold 700 #E5E5E7. Color positive values #30D158, negative #FF453A. Optional subtext: Inter 13px Regular #8E8E93.

**"Create a filter toolbar"**

> Horizontal flex row, 8px gap. "All" pill: bg #E8572A, text white, radius 20px, height 32px, padding 6px 14px. Inactive pills: border 1px #3A3A3C, text #8E8E93. Dropdown filters: pill shape, border #3A3A3C, chevron icon. Active dropdown: bg #E8572A1F, border #E8572A, text #E8572A. Search input: bg #2C2C2E, radius 8px, height 36px, search icon #636366. Secondary button: border 1px #3A3A3C, Inter 14px Medium, icon + text.

**"Create a chat conversation"**

> User message: right-aligned, bg #2C2C2E, radius 12px, padding 16px 20px, Inter 14px #E5E5E7. "YOU" label above in #636366. Agent message: left-aligned, no bubble. Agent name in JetBrains Mono 11px uppercase with status dot. Body in Inter 14px #8E8E93. Tool calls: bg #1C1C1E cards with border #3A3A3C, terminal icon, tool name, file path, status badge right-aligned.

### Implementation Checklist

1. **Fonts loaded:** Inter (400, 500, 600, 700) + JetBrains Mono (500, 600)
2. **Canvas background:** `#121212` on `<body>`
3. **Flat depth:** no box-shadows anywhere, depth via bg color only
4. **Buttons:** 8px radius, 36px default / 28px compact
5. **Badges:** 6px radius, 22px height, tinted bg at 15%, JetBrains Mono 10px uppercase
6. **Inputs:** 8px radius, 36px height, bg #2C2C2E, border #3A3A3C, focus border #E8572A
7. **Cards:** 12px radius, bg #1C1C1E, padding 16px 20px
8. **Filter pills:** 20px radius, 32px height, accent fill (active) or border-only (inactive)
9. **Dividers:** 1px solid #3A3A3C everywhere
10. **Semantic colors:** accent=#E8572A, success=#30D158, danger=#FF453A, warning=#FFD60A, info=#BF5AF2
