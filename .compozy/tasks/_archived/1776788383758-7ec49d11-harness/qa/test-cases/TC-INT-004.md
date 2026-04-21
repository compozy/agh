## TC-INT-004: Synthetic turns preserve transcript, hook, and extension trust boundaries

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 4
**Traceability:** `task_05.md`, ADR-002, ADR-003

---

### Objective

Validate that synthetic turns are rendered and classified as daemon-originated
runtime input across transcript assembly, hook input classification, and
extension-host replay, without being mistaken for user input.

---

### Preconditions

- [ ] A session history exists with both human-origin and synthetic turns
- [ ] Transcript APIs are available over the supported runtime surfaces
- [ ] The execution lane can inspect hook input classification and extension-host prompt replay behavior

---

### Test Steps

1. **Create or load a session containing at least one user turn and one synthetic turn**
   - **Expected:** The session history contains both types of input and remains queryable through the normal transcript/read surfaces.

2. **Fetch the transcript through the supported session transcript API**
   - **Expected:** Synthetic input renders as daemon/system-originated content rather than a user role/message, and the transcript order remains stable.

3. **Inspect hook classification for the synthetic turn**
   - **Expected:** The hook path emits a dedicated synthetic input class such as `synthetic_reentry`, not `user_message`.

4. **Exercise extension-host prompt replay or stored-event turn discovery**
   - **Expected:** Extension-host turn discovery succeeds even when the turn boundary starts from `synthetic_reentry` rather than `user_message`.

5. **Compare transcript output across HTTP and UDS consumers**
   - **Expected:** Both transports return the same ordering and role semantics for the mixed-turn history.

---

### Evidence to Capture

- Transcript excerpt showing the synthetic turn rendered as system/daemon content
- Hook classification output or logs for the synthetic input
- Extension-host or replay evidence showing correct turn-id discovery
- HTTP vs UDS transcript excerpt comparison

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Mixed history | user turn followed by synthetic turn | transcript order remains stable |
| Synthetic-first boundary | turn begins with `synthetic_reentry` | extension replay still finds the correct turn id |
| Tool activity in mixed window | synthetic turn appears near tool calls/results | tool pairing remains correct |
| Cross-transport fetch | same transcript fetched over HTTP and UDS | identical ordering and role semantics |

---

### Related Test Cases

- `TC-INT-003`: Synthetic prompt submission persists dedicated events and queues FIFO
- `TC-INT-007`: Harness observability and HTTP/UDS parity

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/transcript/transcript_test.go`
- `internal/session/transcript_test.go`
- `internal/session/manager_hooks_test.go`
- `internal/extension/host_api_test.go`
