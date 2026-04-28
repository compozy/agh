# Marketing UI Kit

Recreation of the AGH landing page (`packages/site/app/(home)/page.tsx`) as composable React components. Dark-only. Copy + layout + token values pulled from `compozy/agh @ main`.

## Components

- `HomeHeader` — sticky translucent header with `agh` wordmark + Alpha chip, nav pills, pill-search, GH icon button.
- `Hero` — eyebrow lockup, Playfair hero h1, lead, two CTAs, placeholder network visual, 4 signal cards.
- `Features` — 4×2 feature-card grid with Lucide-equivalent inline icons.
- `SupportedAgents` — 8-tile agent CLI strip.
- `InstallSection` — tabbed install command (Homebrew / go / binary) + 3 numbered step cards with embedded code blocks.
- `Comparison` — 4-row positioning table with one accent-highlighted row (AGH).
- `FinalCta` — bordered card, ship-it eyebrow, final CTA duo, GitHub star link.

## Primitives

- `Eyebrow`, `MonoBadge`, `CtaButton`, `FeatureCard`, `SectionHeader`, `CodeBlock`.

## Known gaps vs production

- Hero visual is a placeholder (the real landing uses a complex `NetworkProtocolVisual` SVG composition).
- Agent logos are solid placeholders — real SVGs live in `packages/ui/src/logos/*.tsx` (imported via `@agh/ui/logos`).
- The `hero-bg.png` mesh is faked with a subtle radial-gradient overlay.
- No search palette backend; `⌘K` is decorative.
