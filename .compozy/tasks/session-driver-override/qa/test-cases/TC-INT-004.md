## TC-INT-004: Persisted Provider Wins on Resume After Agent Default Changes

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Session persistence and resume determinism
**Traceability:** Task 02 persistence requirements; ADR-003; TechSpec "Session resume flow" steps 3-4

---

### Objective

Verify that a session created with an explicit provider resumes with that persisted provider even after the agent's current default provider changes.

---

### Preconditions

- [ ] A workspace fixture exists where one agent can resolve to provider `B` initially and later defaults to a different provider.
- [ ] Session create, stop, and resume surfaces are available.
- [ ] `session.json`, global DB, and one read surface can be inspected before and after the config change.

---

### Test Steps

1. Create a session with explicit `provider=B`.
   **Expected:** Create succeeds and the session persists `provider=B`.

2. Stop the session and change the agent default provider in the workspace fixture to a different provider.
   **Expected:** Workspace config now resolves a different default for new no-override sessions.

3. Resume the original session.
   **Expected:** Resume succeeds with `provider=B`; AGH does not switch the session to the new agent default.

4. Inspect persistence and read surfaces after resume.
   **Expected:** `session.json`, SQLite, and read payloads still show `provider=B`.

---

### Evidence to Capture

- Initial create evidence showing `provider=B`.
- Config-change evidence showing the agent default changed afterward.
- Resume response/logs proving `provider=B` was reused.
- Post-resume `session.json` and SQLite row proving the provider stayed stable.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Agent default changes to another valid provider | `B -> A` | Persisted session still resumes on `B`. |
| List/detail queried before resume | Stopped session | Provider remains `B` on read surfaces even before resume. |
| Multiple resumes | Same session resumed twice | Provider stays stable across repeated resumes. |

---

### Related Test Cases

- `TC-FUNC-001` for no-override baseline
- `TC-INT-005` for removed-provider resume failure

---

### Notes

This case proves the central deterministic-runtime promise from ADR-003. If it fails, provider override cannot be considered reliable.
