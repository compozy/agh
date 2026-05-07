# TC-UI-001: Settings > Providers - Source Status + Refresh

**Priority:** P1
**Type:** UI
**Surface:** `web/src/routes/_app/settings/providers.tsx`, `web/src/systems/model-catalog/`
**Requirement:** TechSpec Web, Task 09.
**Status:** Not Run

## Objective

Verify each provider card surfaces source status (id, kind, last refresh, next refresh, redacted last error, stale flag), exposes a refresh control, and reflects daemon-served catalog state including curated metadata snapshot preservation.

## Preconditions

- [ ] Daemon running with seeded catalog state.
- [ ] Web app served under `AGH_WEB_API_PROXY_TARGET` from bootstrap manifest.
- [ ] Browser via `browser-use:browser` or `agent-browser` fallback.

## Test Steps

1. **Open Settings > Providers.**
   - **Expected:** Each provider card lists every catalog source with status; loading skeleton replaced by data; no console errors.
2. **Trigger refresh for one provider.**
   - **Expected:** Refresh button enters pending state; on completion the card updates source rows, `last_refresh_at`, and `refresh_state`; no other provider is impacted.
3. **Force a source error and refresh.**
   - **Expected:** Card shows redacted `last_error`; stale flag visible; manual entry control still available.
4. **Curated metadata snapshot preserved on save.**
   - Edit curated entry; save settings.
   - **Expected:** Catalog adapters use snapshot-preserved metadata so unrelated rows are not mutated; daemon `config` source rows reflect only the edited fields.
5. **Visual conformance.**
   - **Expected:** Card uses `DESIGN.md` tokens (no shadows, warm-dark palette, signal palette for refresh state colors). Default state matches `Paper` artboards in `DESIGN.md`. No invented metrics shown.

## Visual Specifications

- Background: `oklch` warm-dark token from `DESIGN.md`.
- Refresh button states: idle (neutral), running (warning `#FFD60A`), success (`#30D158`), failure (`#FF453A`).
- Stale label uses warning palette.

## Responsive Checks

- Desktop 1280px, Tablet 768px, Mobile 375px - layout legible at each breakpoint.

## Audit Coverage

- C5, C8, C11.

## Pass Criteria

- Source status renders correctly with redacted errors.
- Refresh updates only target provider.
- Curated edit preserves snapshot.

## Failure Criteria

- Stale flag missing.
- Console error during refresh.
- Curated edit corrupts other models.
