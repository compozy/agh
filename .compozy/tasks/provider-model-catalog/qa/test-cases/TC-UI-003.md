# TC-UI-003: New Session Dialog - Catalog vs ACP Config Options

**Priority:** P1
**Type:** UI
**Surface:** `web/src/systems/session/components/session-create-dialog.tsx`, `web/src/systems/session/hooks/use-session-create-dialog.ts`, `web/src/systems/model-catalog/lib/derive-active-session-options.ts`.
**Requirement:** TechSpec SI-7, Task 09.
**Status:** Not Run

## Objective

Verify the new session dialog uses the daemon catalog (not legacy `supported_models`) for pre-session model selection, supports manual entry, surfaces stale/error/empty states, and switches to ACP `configOptions` once the session is active.

## Preconditions

- [ ] Daemon with seeded catalog.
- [ ] Web app under bootstrap manifest proxy target.

## Test Steps

1. **Open new session dialog with seeded catalog.**
   - **Expected:** Model picker lists rows from `useProviderModels`; sources/availability badges visible; manual entry input present.
2. **Catalog stale.**
   - Force stale flag in seed.
   - **Expected:** Stale models render with stale label; selection still allowed.
3. **Catalog empty.**
   - Seed with zero rows.
   - **Expected:** Empty state shown; manual entry remains valid; submitting manual model creates session successfully.
4. **Refresh in dialog.**
   - **Expected:** `useRefreshProviderModels` triggers; loading state visible; rows update on completion.
5. **Switch to active session.**
   - After session creates, open active session settings panel.
   - **Expected:** Controls switch to ACP `configOptions` (model + reasoning) via `deriveActiveSessionOptions`; catalog metadata never overrides session option current value (SI-7).
6. **No legacy field reads.**
   - Inspect React Query cache + network responses.
   - **Expected:** No `supported_models` / `default_model` / `supports_reasoning_effort` references.

## Audit Coverage

- C5, C7, C11.
- SI-7.

## Pass Criteria

- Catalog drives picker; ACP overrides post-creation.
- Manual entry valid in all states.

## Failure Criteria

- Picker reads legacy field.
- ACP override missing.
- Manual entry blocked.
