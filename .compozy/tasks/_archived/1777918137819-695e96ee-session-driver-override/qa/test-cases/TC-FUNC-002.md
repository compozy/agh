## TC-FUNC-002: Explicit Provider Override Re-Resolves Provider-Owned Runtime Fields

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Config resolution and session startup
**Traceability:** Task 01 requirements on override semantics for `provider`, `command`, `default_model`, and provider-owned MCP layers; ADR-001; TechSpec "Core Interfaces" and "Testing Approach"

---

### Objective

Verify that an explicit session provider override changes only the provider-owned runtime state and persists the selected provider as the session's runtime identity.

---

### Preconditions

- [ ] Workspace fixture `WS-PROVIDER-MATRIX` exists with one agent whose default provider is `A` and alternate provider is `B`.
- [ ] Provider `A` and provider `B` expose visibly different command, default model, or MCP-layer markers.
- [ ] Backend log capture or driver/runtime inspection exists so the resolved provider-owned fields are observable.
- [ ] The agent identity remains constant across the test.

---

### Test Steps

1. Create a session for the chosen agent with explicit `provider=B`.
   **Expected:** Session creation succeeds and the returned payload/output shows the same `agent_name` with effective `provider=B`.

2. Inspect startup evidence for resolved runtime fields.
   **Expected:** Provider-owned command/default model/MCP markers come from provider `B`, while the session still uses the original agent identity.

3. Inspect `session.json` and the global session row.
   **Expected:** Both persistence layers store `provider=B`.

4. Read the session through a second surface such as list/detail/status.
   **Expected:** The session still reports `provider=B`; no surface reports the agent default provider instead.

---

### Evidence to Capture

- Create request and response showing `provider=B`.
- Backend log or startup/runtime evidence demonstrating provider `B` owns the resolved command/model/MCP state.
- `session.json` and SQLite row storing `provider=B`.
- One read surface proving the same provider round-trips.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Agent has explicit command/model in config | Override to provider `B` | Explicit agent command/model influence is cleared and provider `B` owns runtime fields. |
| Provider swap changes MCP layer | Default provider `A`, override `B` | Provider-owned MCP layer reflects `B`; global and agent-local layers remain intact. |
| Same agent across all steps | Constant `agent_name` | Agent identity does not change when provider changes. |

---

### Related Test Cases

- `TC-FUNC-001` for no-override baseline
- `TC-FUNC-003` for invalid override failure
- `TC-INT-008` for transport parity on the persisted provider

---

### Notes

This is the highest-risk semantic case. Use fixtures that make provider-owned runtime changes obvious enough that task 08 can produce hard evidence rather than inference.
