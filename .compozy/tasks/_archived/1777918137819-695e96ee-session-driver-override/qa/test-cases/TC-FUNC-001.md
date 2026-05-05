## TC-FUNC-001: No-Override Session Create Uses the Resolved Agent Default Provider

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Config and session lifecycle baseline
**Traceability:** Task 01 no-override requirement; Task 02 persistence/read-model requirement; TechSpec "Resolution helper semantics" and "Session create flow"

---

### Objective

Verify that creating a session without an explicit provider uses the agent's resolved default provider, persists that provider, and exposes it coherently on read surfaces.

---

### Preconditions

- [ ] Workspace fixture `WS-PROVIDER-MATRIX` exists with one agent whose default provider is known.
- [ ] The workspace exposes at least one alternate provider, but this test will not select it.
- [ ] Backend logs and SQLite inspection are available.
- [ ] No pre-existing session ID is reused for this run.

---

### Test Steps

1. Create a new session through one explicit surface without sending `provider`.
   **Expected:** Session creation succeeds and the returned session payload/output shows the default provider from the resolved agent.

2. Inspect the resulting on-disk `session.json`.
   **Expected:** The metadata contains the same effective provider that was returned at create time.

3. Inspect the global `sessions` row and list/detail/status surfaces for the created session.
   **Expected:** Every read surface reports the same provider; there is no blank or omitted provider.

4. Stop and resume the same session without changing workspace config.
   **Expected:** Resume succeeds with the same persisted provider and does not recompute a different runtime identity.

---

### Evidence to Capture

- Create request and response or CLI output showing no explicit provider request and the resolved provider in the result.
- `session.json` snippet with `provider`.
- SQLite row from `sessions` showing `provider`.
- One list/detail/status sample showing the same provider.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Request omits `provider` entirely | No `provider` field | Default provider is resolved and persisted. |
| Request sends empty provider | `provider: ""` | Behavior matches omission and still resolves the default provider. |
| Workspace has only one visible provider | Default only | Provider still persists explicitly in metadata and read surfaces. |

---

### Related Test Cases

- `TC-FUNC-002` for explicit override behavior
- `TC-INT-004` for persisted resume after default drift

---

### Notes

Use this case as the baseline comparator for every explicit override case. If this baseline fails, task 08 should stop before deeper provider-override validation.
