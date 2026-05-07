# TC-UI-002: Settings > Providers - Manual Entry + Curated Edit

**Priority:** P1
**Type:** UI
**Surface:** `web/src/routes/_app/settings/providers.tsx`
**Requirement:** TechSpec SI-6, Task 09.
**Status:** Not Run

## Objective

Verify the new settings form edits `models.default` and `models.curated`, allows manual model IDs (curated is not an allowlist), and emits payloads matching the new nested contract (no `default_model`, `supported_models`, or `supports_reasoning_effort`).

## Preconditions

- [ ] Daemon and web app running.

## Test Steps

1. **Add curated model with reasoning efforts.**
   - **Expected:** Form accepts metadata; submits payload using `models.default`/`models.curated`; daemon persists; CLI/HTTP/UDS reflect change.
2. **Set default to a model NOT in curated list.**
   - **Expected:** Form accepts; payload shows `models.default = "manual-id"`; manual model becomes selectable in session create dialog.
3. **Reject duplicate curated id.**
   - **Expected:** Form shows inline validation error.
4. **Reject blank reasoning effort.**
   - **Expected:** Inline validation error referencing the empty entry.
5. **Inspect payload network request.**
   - **Expected:** No legacy keys; matches generated TS contract `web/src/generated/agh-openapi.d.ts`.

## Audit Coverage

- C5, C8.
- SI-6.

## Pass Criteria

- Form contract matches generated types.
- Manual entry accepted.

## Failure Criteria

- Legacy fields appear in payload.
- Manual default rejected.
