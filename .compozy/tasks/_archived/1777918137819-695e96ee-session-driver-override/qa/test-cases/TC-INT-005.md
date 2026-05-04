## TC-INT-005: Resume Fails Explicitly When the Persisted Provider Is Unavailable

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 18 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Resume failure behavior
**Traceability:** Task 02 unavailable-provider resume requirement; ADR-003; TechSpec "Session resume flow" step 5 and "Monitoring and Observability"

---

### Objective

Verify that AGH returns an explicit, actionable failure when a persisted provider is no longer available and never falls back silently to the current agent default.

---

### Preconditions

- [ ] A session already exists with a persisted provider that can later be removed from workspace config.
- [ ] The workspace fixture `WS-PROVIDER-REMOVED` is prepared to hide or delete the persisted provider after session creation.
- [ ] Resume can be attempted through at least one explicit backend-facing surface.
- [ ] Error payloads and backend logs are being captured.

---

### Test Steps

1. Create and stop a session with explicit `provider=B`.
   **Expected:** The session persists `provider=B`.

2. Remove provider `B` from the relevant workspace-visible config.
   **Expected:** New resolution for `B` fails in the workspace.

3. Attempt to resume the persisted session.
   **Expected:** Resume fails explicitly and names the session id and missing provider; AGH does not fall back to a different provider.

4. Inspect persistence after the failed resume.
   **Expected:** The session still stores `provider=B`; the failed resume does not rewrite the session to a fallback provider.

---

### Evidence to Capture

- Resume error payload or CLI stderr naming the session id and missing provider.
- Backend log lines showing `phase=resume` and the missing provider.
- Post-failure `session.json` and SQLite evidence showing `provider=B` is still persisted.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Provider removed entirely | `B` absent from workspace config | Explicit unavailable-provider failure. |
| Provider renamed | `B` replaced with `B2` | Resume still fails; no silent mapping or fallback. |
| Agent default remains valid | Another provider still available | Resume still fails instead of switching to the valid default. |

---

### Related Test Cases

- `TC-INT-004` for successful persisted-provider resume
- `TC-UI-011` for the browser-visible manifestation of this same failure

---

### Notes

Task 08 should reuse the same persisted session in this case and in `TC-UI-011` so backend and browser evidence stay aligned.
