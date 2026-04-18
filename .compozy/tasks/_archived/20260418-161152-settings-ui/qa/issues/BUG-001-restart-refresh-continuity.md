# BUG-001: Restart Polling Lost Continuity After Full Page Refresh

## Summary

- Status: `Fixed`
- Severity: `High`
- Origin case: `TC-FUNC-002`
- Surface: `/settings/general`

## Symptom

When the operator saved a restart-required settings change and refreshed the page while the daemon replacement was in progress, the restart banner disappeared and the UI stopped polling for the active operation.

## Root Cause

`use-settings-restart-store` kept the active operation id and last known restart status only in in-memory Zustand state. A full page reload during daemon handoff cleared that state before the replacement daemon reported a terminal status.

## Fix

- Persisted the minimal restart operation state in `sessionStorage` with Zustand `persist`.
- Added store reset support for test isolation.
- Added hook regression coverage that rehydrates a pending restart and proves polling survives page refresh.
- Kept the daemon-served browser test on a real full-document refresh path.

## Regression Coverage

- `web/src/systems/settings/hooks/use-settings-restart.test.tsx`
- `web/e2e/settings.spec.ts`
- `make test-e2e-web`
- `make verify`

## Evidence

- `qa/screenshots/TC-FUNC-002-general-restart-polling.png`
- `qa/screenshots/TC-INT-016-general-restart-ready.png`
