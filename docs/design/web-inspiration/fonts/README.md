# Fonts

## Shipped via Google Fonts (loaded by `colors_and_type.css` consumers)
- **Inter Variable** — UI sans
- **Playfair Display** — marketing display serif, weights 400/500
- **JetBrains Mono** — code + eyebrows, weights 500/600

## Missing — please provide
- **`NuixyberNext-Regular.ttf`** — the brand wordmark font. Only used for the string `agh` in the header logo lockup. We were unable to extract the real binary from the private `compozy/agh` repo (it lives in `packages/site/public/fonts/NuixyberNext-Regular.ttf`).

Drop the file here as `NuixyberNext-Regular.ttf` and the `@font-face` rule in `colors_and_type.css` will pick it up automatically.

Until then, the wordmark falls back to Inter sans.
