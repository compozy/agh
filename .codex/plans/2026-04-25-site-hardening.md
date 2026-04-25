# packages/site Quality, Performance & Security Hardening

## Summary

- Keep the site as a static export with `output: "export"` and canonical domain `https://agh.network`.
- Harden `packages/site` across bundle size, raw/unoptimized images, metadata/SEO, static-host security headers, error states, package verification, and budget checks.
- Preserve user-owned dirty work in `packages/site/app/protocol/layout.tsx`, `packages/site/app/runtime/layout.tsx`, `packages/site/components/site/docs-header.tsx`, and `packages/site/public/images/runtime/runtime-overview-storyboard-v1.png`.

## Key Changes

- Split marketing home from Fumadocs runtime so docs-only providers/search do not inflate the homepage bundle.
- Replace above-fold Remotion playback with a static first-paint visual and only lazy-load interactivity if it stays within budget.
- Add static-export-safe image handling, public asset hygiene, and bundle/asset budget enforcement.
- Add canonical site metadata, sitemap, robots, OG image, branded not-found/error states, and static deployment security headers.
- Add package-local `verify` that runs format, lint, typecheck, tests, build, and budget checks.

## Acceptance Criteria

- `cd packages/site && bun run verify` passes.
- `cd packages/site && bunx next experimental-analyze --output` produces diagnostics within budgets.
- `make verify` passes before completion unless blocked by unrelated user-owned work.
- No destructive git commands are used and unrelated dirty files remain untouched.
