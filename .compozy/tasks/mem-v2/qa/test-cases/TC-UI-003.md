# TC-UI-003: Web Session Inspector Memory Ledger Surface

**Priority:** P1
**Type:** UI
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify the session inspector memory panel is a read-only forensic ledger surface backed by `GET /api/memory/sessions/{session_id}/ledger`.

## Preconditions

- [ ] At least one active session and one stopped session exist.
- [ ] Stopped session has a materialized ledger or intentionally unavailable ledger.
- [ ] Web dev server is connected to isolated daemon.

## Test Steps

1. **Run focused tests**
   - Input: `cd web && bunx vitest run src/routes/_app/-agents.\\$name.sessions.\\$id.test.tsx src/systems/session/components/session-inspector.test.tsx src/systems/session/adapters/session-api.test.ts`
   - **Expected:** Route query gating, adapter status mapping, and inspector rendering tests pass.

2. **Open active session inspector**
   - Input: browser route for active/running session.
   - **Expected:** Ledger query is disabled until `session.state === "stopped"`; UI does not cache a pre-stop 404.

3. **Open stopped session inspector**
   - Input: browser route for stopped session.
   - **Expected:** Ledger query runs and panel shows workspace, root/parent session, spawn depth, path, checksum, version, created/stopped timestamps, and ledger event metadata.

4. **404 unavailable state**
   - Input: stopped session without materialized ledger.
   - **Expected:** UI renders truthful unavailable empty state, not a hard error.

5. **Non-404 error state**
   - Input: force server error.
   - **Expected:** UI renders forensic error state and does not invent repair controls.

6. **Read-only assertion**
   - Input: inspect controls.
   - **Expected:** No editor, promote, replay, arbitrary payload JSON, or unsupported observability widget appears.

## Evidence To Capture

- Focused web test log.
- Browser screenshots for active, stopped, unavailable, and error states.
- API payloads or mocked equivalent from isolated daemon.

