# Shared Logo Primitive Plan

## Summary

- Create a shared `Logo` primitive in `@agh/ui` with `logo`, `symbol`, and `lettering` variants.
- Use the `symbol` variant in `web/`, the full `logo` variant in `packages/site`, and the `symbol` variant in the Remotion hero rail.
- Remove the old local text-logo component from `packages/site` instead of keeping a compatibility shim.

## Public API

- Add `Logo`, `LogoProps`, and `LogoVariant` exports from `packages/ui/src/index.ts`.
- `LogoVariant = "logo" | "symbol" | "lettering"`.
- `LogoProps` extends SVG props and adds:
  - `variant?: LogoVariant`, default `"logo"`.
  - `label?: string`, default `"AGH"`.
  - `decorative?: boolean`, default `false`.
- A non-decorative logo renders as an image with an accessible label.
- A decorative logo renders with `aria-hidden="true"` so the parent link/button supplies the accessible name.

## Implementation

- Create `packages/ui/src/components/logo.tsx`.
- Render one `<svg data-slot="logo" data-variant={variant}>`.
- Use the supplied SVG artwork with these source viewBoxes:
  - `logo`: `0 0 972 386`.
  - `symbol`: `0 0 355 355`.
  - `lettering`: `0 0 543 362`.
- Compose the full logo internally from the symbol artwork translated down by `30.6388` and the lettering artwork translated right by `429`.
- Keep the brand SVG fills from the supplied asset (`#E8572B`, `#231F20`, `white`) intact.
- Add `packages/ui/src/components/logo.test.tsx`.
- Add `packages/ui/src/components/stories/logo.stories.tsx`.
- Update `packages/ui/README.md`.
- Replace `packages/site/components/logo.tsx` usages with `@agh/ui` imports and delete the local file.
- Replace the ad-hoc `web/src/components/app-sidebar.tsx` app-logo letter with `<Logo variant="symbol" decorative />`.
- Replace the generic Remotion rail icon in `packages/site/remotion/hero/components/sidebar-rail.tsx` with `<Logo variant="symbol" decorative />`.

## Tests

- `packages/ui` logo tests cover default, `symbol`, `lettering`, accessibility modes, `className`, `style`, and `viewBox`.
- `web/src/components/app-sidebar.test.tsx` verifies the shared logo with `data-variant="symbol"` while preserving the link behavior.
- `packages/site/components/site/home-header.test.tsx` verifies the full shared logo in the site header link.

## Verification

- `bun run --cwd packages/ui test -- src/components/logo.test.tsx`
- `bun run --cwd web test:raw -- src/components/app-sidebar.test.tsx`
- `bun run --cwd packages/site test -- components/site/home-header.test.tsx`
- `make web-typecheck`
- `make web-lint`
- `make verify`

## Assumptions

- The SVG `viewBox` values are the source of truth where they differ from dimensions described in prose.
- The `Alpha` chip is removed from the header logo because the requested site header treatment is the complete logo artwork.
- No dependency changes are needed.
- Unrelated worktree changes remain untouched.
