# AGH Design System

A design system for **AGH** — a local-first agent runtime and open coordination protocol, built by Compozy. This system captures the visual language of the AGH marketing site and documentation (the `packages/site/` surface) so new artifacts — slides, mocks, prototypes, production UI — feel unmistakably AGH.

> _"A durable runtime and open coordination layer for real agent work."_ — layout metadata

---

## About the product

AGH is marketed as an agent operating system: a single local binary that runs real agent CLIs (Claude Code, Codex, Gemini CLI, OpenCode, Copilot CLI, Cursor, Kiro, Pi) as durable, resumable, auditable sessions — with a built-in network protocol (`agh-network/v0`) for peer discovery and delegation across machines.

Two product surfaces are represented here:

1. **AGH Runtime** — the local daemon. Sessions, memory, skills, workspaces, automation, bridges, observability, hooks.
2. **AGH Network** — `agh-network/v0`, the open agent network protocol with seven message kinds (`greet`, `whois`, `say`, `direct`, `capability`, `receipt`, `trace`) running over NATS + JSON.

The one website (`packages/site/`) serves both: a marketing landing page plus a docs surface (Fumadocs MDX) with two trees — `/runtime/*` and `/protocol/*`.

## Sources

All visual material was extracted from:

- **Repository:** [`compozy/agh`](https://github.com/compozy/agh) @ `main`
- **Primary subtree:** `packages/site/` — Next.js App Router, Tailwind v4, Fumadocs UI
- **Shared UI package:** `packages/ui/` — shadcn/base-ui primitives + `tokens.css` (the source of truth for colors + radii)

Key files pulled from the repo:

- `packages/ui/src/tokens.css` — all color + radius tokens
- `packages/site/app/global.css` — font wiring, doc body styles, Fumadocs overrides
- `packages/site/app/layout.tsx` — Inter + JetBrains Mono + Playfair Display via `next/font/google`; force dark theme
- `packages/site/components/landing/*` — Hero, FeaturesSection, NetworkSection, InstallSection, Comparison, FinalCta, SupportedAgents, primitives (CtaButton, CodeBlock, FeatureCard, SectionHeader, MonoBadge, KindChip)
- `packages/site/components/site/home-header.tsx` + `components/logo.tsx` — header shell and wordmark
- `packages/site/components/docs/doc-page-masthead.tsx` — docs H1 treatment
- `packages/ui/src/components/*` — Button, Badge, Card, Input, Alert, Kbd (shadcn-derived)

> Reader may not have access to the private `compozy/agh` repo. All necessary tokens, component code, and copy samples are mirrored into this design system so it stands alone.

---

## Content fundamentals

AGH copy has a very specific voice — **operator-first, engineer-to-engineer, confident, slightly dry**. It assumes the reader knows what an agent runtime is and does not explain itself twice.

**Tone and vocabulary**

- Direct, no hype. Nouns over adjectives. "Everything logged, everything replayable."
- Writes to an **operator** (the person running the system), not to a decision-maker. Examples: "one operator surface", "replayable history", "the operator ends up stitching together scripts".
- "You" is used sparingly; most copy is imperative or third-person about the product. "Install the runtime." "Bring the CLI you already use." "The runtime lives on your machine."
- Never "we" or "our" in marketing body. Product is the subject: "AGH does X", "AGH Network gives agents Y".
- Dry confidence over hustle. "Shipped today." "Real commands, not docs-ware." "No Docker. No Postgres."

**Casing**

- Body: sentence case. `"Resume any agent run"`, `"Context that survives restarts"`.
- Eyebrows: **UPPERCASE**, letter-spaced ~0.06em, always mono. `SESSIONS`, `MEMORY`, `WHAT YOU GET`, `GETTING STARTED`.
- Product names: `AGH`, `AGH Runtime`, `AGH Network` — title case. Protocol name is lowercase mono: `agh-network/v0`.
- Brand wordmark: `agh` (all lowercase, NuixyberNext).

**Structural patterns**

- **Eyebrow + big title + short lead + visual** is the canonical section shell. See `SectionHeader` primitive.
- **Three-word card titles** are common: "Resume any agent run", "Reusable playbooks", "Per-project everything". Snappy, verb-forward.
- **Feature cards pair an eyebrow (concept) with a verb-forward title (benefit) and a 1-sentence mechanism (proof).**
- **Honest constraints**, shown not hidden: "macOS and Linux today", "Alpha" chip in the logo, "8 ACP CLIs".

**Emoji / symbols**

- **No emoji.** Anywhere. Not in UI, not in docs, not in marketing.
- Dashes are em-dash `—` (copy) or `· ` (meta separators like "macOS · recommended").
- Arrows: Lucide `ArrowUpRight` as "source link" / "continue reading" indicator.
- `$ ` prompt in shell code blocks, rendered in accent orange.

**Example copy to match** (locked positioning as of 2026-05-01)

- Hero: _"An open workplace for AI agents."_
- Sub: _"AGH runs the agent CLIs you already use as durable sessions — with memory, autonomy, tools, and automation — connected on agh-network/v0 channels where they find each other, share capabilities, and close work with receipts."_
- CTA: _"Install the runtime"_, _"Read the agh-network/v0 spec"_, _"Star on GitHub"_.
- Comparison: _"Other tools stop at the runtime boundary."_

---

## Visual foundations

### Theme

**Dark mode only.** `color-scheme: dark` is hardcoded, `RootProvider` forces `.dark` with `enabled: false`. There is no light mode. Design assets should never render on a white background.

### Color

- **Canvas `#141312`** — warm near-black, slightly tinted away from pure black. Body background.
- **Canvas deep `#0E0E0F`** — code blocks, deeper panels on the landing page (`background="deep"` on `SectionFrame`).
- **Surface `#1E1C1B`** — card background, sidebar.
- **Surface elevated `#2E2C2B`** — popovers, icon wells inside cards, hover targets.
- **Divider `#3C3A39`** — 1px borders everywhere. Rarely stronger than 1px.
- **Accent `#E8572A`** — a single warm orange. The whole system is monochromatic-plus-accent. Use sparingly: CTAs, active states, eyebrows on key marketing surfaces, code prompts, mono badges, sidebar active indicator.
- **Accent tint `#E8572A26`** (15% alpha) — active pill backgrounds, kind-chip backgrounds.
- **Text primary `#E5E5E7`** / secondary `#8E8E93` / tertiary `#636366` / label `#98989D` — Apple-derived neutral scale. Generous spacing between roles keeps hierarchy readable on the dark canvas.
- **Semantic** — `success #30D158`, `danger #FF453A`, `warning #FFD60A`, `info #BF5AF2`. Only ever shown as tinted chips (15% alpha bg + full-color text), never as solid banners.

### Type

- **Body:** `Inter Variable` — the workhorse sans.
- **Display (home only):** `Playfair Display` — weight 400/500. Applied via `.site-home h1, .site-home h2` override. Gives the landing a slight editorial / tasteful magazine feel that docs does not have.
- **Mono:** `JetBrains Mono` — all labels, eyebrows, code, badges, chips, metadata. Often at `10–12px`, uppercase, `tracking: 0.06em`.
- **Wordmark:** `NuixyberNext-Regular.ttf`. Used only for the literal string "agh" in the header Logo component.
- **Docs h1** is a heavier sans treatment (`font-weight: 600`, `letter-spacing: -0.05em`) — distinct from marketing hero which uses Playfair 400.

### Spacing + scale

No formal scale token — the codebase uses Tailwind's default (`gap-4`, `p-6`, etc.) plus a few custom `clamp()` type ramps on headings. Treat `4 / 8 / 12 / 16 / 24 / 32 / 48 / 64` as the working grid. Section vertical padding is set by a `SectionFrame` `padY` prop (`md` / `lg` / `xl`).

### Background treatments

- **Default:** flat canvas. No gradients on content.
- **Hero:** a single faded mesh PNG at 20% opacity with `mix-blend-screen` — `bg-size-[100%_auto]` on `/hero-bg.png`. Provides texture without competing with type. **Not a purple gradient — a warm near-black mesh.**
- **Images are not used elsewhere in the marketing flow.** The design leans on type + diagrams instead of photography.
- Landing sections alternate `canvas` / `surface` / `deep` backgrounds for rhythm via `SectionFrame background=`.

### Borders

- **1px solid `--color-divider`** is the universal separator. Cards, section dividers, table rows, tabs, inputs, buttons-outline.
- On hover, feature cards shift border to `color-mix(in srgb, var(--color-accent) 40%, var(--color-divider))` — a subtle warm-up.
- Highlighted rows in the comparison table get `border-l-4 border-l-accent` and a tinted background. This is the **only** "colored left border" pattern in the system — do not proliferate it.

### Shadows

Minimal. `ring-1 ring-foreground/10` on cards (very subtle inner line), and `shadow-xs` on buttons/inputs via shadcn defaults. **No heavy drop shadows.** The depth story is layered surfaces (`canvas → surface → surface-elevated`), not blur.

### Corner radii

- `6px` — small chips (kind chips, mono badges).
- `8px` — buttons, inputs, small UI (`--radius`).
- `10–12px` — cards, icon wells, code blocks (`--radius-diagram`).
- `9999px` (`rounded-full`) — header nav pills, `slots.searchTrigger`, GH button.
- No extreme rounding. No pill buttons on CTAs — CTAs are `lg` rounded-lg.

### Animation

- **Transitions:** `transition-colors` only — ~150ms. Background, border, text color change on hover. No transform transitions on hover.
- **Active press:** `active:translate-y-px` on buttons (1px nudge). That's the entire press animation vocabulary.
- **Pulse:** `animate-ping` on a 1.5px success dot (the "online" indicator in InstallSection). Used sparingly to signal live state.
- **Shimmer:** defined as `@utility animate-shimmer` (200% bg shift, 2s infinite) — used for skeleton loaders in the UI package.
- **Reduced motion is respected globally** via a `@media (prefers-reduced-motion: reduce)` rule that zeroes all durations.

### Hover / active states

- **Text links:** body text → accent color on hover.
- **Buttons (primary):** `bg-primary` → `bg-primary/80` on anchor-buttons; `active:translate-y-px`.
- **Buttons (outline/ghost):** border turns accent, text turns accent (ghost CTA overrides: `hover:border-accent hover:text-accent hover:bg-transparent`).
- **Cards:** border cool → border warm (40% accent mix). No lift, no scale.
- **Nav pills:** active state gets `bg-[rgba(232,87,42,0.12)]` + accent text.
- **Code copy button:** tertiary → accent on hover; checkmark swap on copy-success (1.5s).

### Transparency + blur

- Sticky header uses `bg-[rgba(18,18,18,0.92)] backdrop-blur-xl` — pinned, translucent, blurred.
- Hero signal cards use `border-white/10 backdrop-blur-sm` on top of the mesh bg.
- Otherwise, transparency is avoided — surfaces are solid.

### Imagery vibe

No photography. The landing uses diagrams (`NetworkProtocolVisual`, `RuntimeMicroDiagram`, `ArchitectureDiagram`) made of SVG lines, chips, and monospace labels. Imagery, when added, should be **warm, desaturated, grainy, dark-ground** — to match the `#141312` canvas. Never bright/bluish/cool.

### Card anatomy

- `rounded-[12px]` (`--radius-diagram`).
- `border border-divider`, solid.
- `bg-surface` (`#1E1C1B`).
- Padding: `p-6` (24px).
- Inside: icon well (40×40, `rounded-[10px]`, `bg-surface-elevated`, accent-colored lucide icon) → mono eyebrow → sans title → secondary-text description → optional mono source cite link.
- Hover: border warms toward accent.

### Layout rules

- **Max width `--site-layout-width: 1200px`** for landing content. Centered, generous gutters.
- **Docs max width is wider** (`96rem`) with a fixed left sidebar (`16rem`) and right TOC (`14rem`).
- Hero is `grid-cols-[minmax(0,1fr)_minmax(0,540px)]` — copy-left / visual-right, reversed on mobile.
- Sections stack vertically; no horizontal scroll anywhere except code blocks.

---

## Iconography

**Primary system: Lucide React.** Every icon in the landing components comes from `lucide-react` — `Check`, `Minus`, `ArrowUpRight`, `Activity`, `Boxes`, `Database`, `FileCode2`, `Network`, `Plug`, `Sparkles`, `Timer`, `Star`, `Copy`, `Check`. Standard weight (2), size `h-4 w-4` (16px) inside 40×40 icon wells; `h-3 w-3` for inline decoration.

**Stroke weight:** default 2. **Never** switch to filled or duotone sets — Lucide strokes match the 1px divider language.

**Color:** accent (`#E8572A`) when inside a card icon well; otherwise `currentColor` inheriting from the text role (secondary/tertiary). Never multi-color.

**Sizes used**

- `size-3` / `h-3 w-3` (12px) — inline in mono chips, copy buttons, source-cite arrows.
- `h-3.5 w-3.5` (14px) — check / minus cells in comparison table.
- `h-4 w-4` (16px) — default, inside 40×40 wells.
- `h-5 w-5` (20px) — rarely; hero dismiss / nav.

**Logos**: custom SVG-in-TSX files under `packages/ui/src/logos/` (imported via `@agh/ui/logos`) for partner agents (Claude, OpenAI, Gemini, GitHub, Cursor, OpenCode, Kiro, Pi, Slack, Telegram, Discord, Linear, Microsoft Teams, WhatsApp, Google Chat). For this design system we ship a representative set as inline SVG placeholders — see `assets/logos/`. **Do not generate logo SVGs from scratch**; request the real SVGs if missing.

**Brand wordmark**: the literal string `agh` in `NuixyberNext` font, followed by an `ALPHA` chip (mono, uppercase, tracking-widest, muted border). See `preview/brand-wordmark.html`.

**Unicode / emoji**: not used. `$ ` is the only glyph marker used in code blocks. `—` em-dash is the only typographic flourish.

**Font substitution note:** we were unable to access the real `NuixyberNext-Regular.ttf` binary from the repo. The `@font-face` declaration in `colors_and_type.css` references `./fonts/NuixyberNext-Regular.ttf` and falls back to sans. The wordmark card currently renders in Inter. **Please drop the real `NuixyberNext-Regular.ttf` into `fonts/` to restore the brand wordmark.**

---

## Index

Top-level files in this design system:

| Path                  | Purpose                                                                |
| --------------------- | ---------------------------------------------------------------------- |
| `README.md`           | This file — product context, content + visual foundations, iconography |
| `.agents/skills/agh-design/SKILL.md` | Agent skill descriptor (user-invokable); lives next to other repo skills |
| `colors_and_type.css` | All CSS variables, `@font-face`, base + semantic type classes          |
| `fonts/`              | Web fonts (NuixyberNext placeholder — awaiting real file)              |
| `assets/logos/`       | Inline SVG copies of partner + brand logos                             |
| `preview/`            | 700×(≤400)px cards registered in the Design System tab                 |
| `ui_kits/marketing/`  | React recreation of the landing page sections                          |
| `ui_kits/docs/`       | React recreation of a `/runtime/core/*` doc page                       |

### UI kits

- **`ui_kits/marketing/`** — `index.html` + JSX components. Renders a Hero, Features grid, Supported Agents strip, Install tabs, Comparison table, FinalCta, and Home header. Use for marketing pages, blog posts, launch announcements.
- **`ui_kits/docs/`** — `index.html` + JSX components. Renders a docs shell: sticky home header, left sidebar tree, DocPageMasthead + long-form body, right TOC. Use for documentation pages and long-form reference.

### Preview cards (design system tab)

See `preview/` for each card. Grouped into **Type**, **Colors**, **Spacing**, **Components**, **Brand**.

---

## Caveats & flags

- **NuixyberNext** font file: not shipped. Wordmark renders in Inter fallback until you drop the real `.ttf` into `fonts/`.
- **Logos for partner agents** (Claude, OpenAI, Gemini, etc.) are stored as simplified placeholders — real assets live in `packages/ui/src/logos/*.tsx` (exported via `@agh/ui/logos`). Copy those TSX files and strip the React wrapper to get the production SVGs.
- **`hero-bg.png` mesh** is referenced by the hero but not shipped here (lives in `packages/site/public/`). The marketing UI kit renders the hero without it; a CSS noise fallback is used.
- **Diagrams** (`NetworkProtocolVisual`, `RuntimeMicroDiagram`, `ArchitectureDiagram`) are complex SVG compositions — not recreated in the UI kit. The hero visual is replaced with a static placeholder labeled "Network protocol visual".
