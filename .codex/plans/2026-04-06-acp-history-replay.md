# ACP Historical Replay Fix

## Summary

- Replace the `web/` historical replay contract with a backend canonical transcript API.
- Normalize legacy raw ACP payloads in the backend read path.
- Persist future `user_message` events and canonical envelopes for new events.
- Remove silent ACP resume fallback that breaks continuity.

## Key Changes

- Add `GET /api/sessions/:id/transcript` on HTTP and UDS.
- Add backend transcript assembly that supports both legacy raw ACP rows and new canonical rows.
- Persist `user_message` before calling ACP in `Manager.Prompt`.
- Persist new event payloads in a canonical envelope with nested `raw` diagnostics.
- Update the web session route and session system to hydrate from transcript instead of raw history.

## Test Plan

- Go tests for transcript normalization from legacy raw ACP rows.
- Go tests for `user_message` persistence and strict resume behavior.
- Go handler tests for transcript endpoints.
- Frontend tests for transcript fetching, route hydration, and rendering of transcript messages.
- Final verification with `make web-lint`, `make web-typecheck`, and `make verify`.

## Defaults

- No migration/backfill of old SQLite data.
- No synthetic `user_message` for legacy sessions.
- Legacy compatibility stays concentrated in backend transcript assembly.
