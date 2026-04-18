## TC-INT-003: Prompt Recall Augments Driver Dispatch Without Mutating Stored User Events

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Session Prompt Path
**Requirement:** REQ-MEM-005

---

### Objective

Verify that bounded durable-memory recall is injected only into the driver-dispatch message and does not alter the persisted raw user message in the session event log.

---

### Preconditions

- [ ] A workspace session can be created against a controllable ACP/fake driver.
- [ ] The tester can inspect both dispatched prompt content and stored session events.
- [ ] The workspace corpus contains at least one relevant memory.

---

### Test Steps

1. Seed a workspace memory that should match the user message.
   - **Expected:** The memory is searchable before prompting.

2. Create a session in that workspace and send a prompt such as `remember me`.
   - **Expected:** The prompt is accepted and the session records a user message event.

3. Inspect the driver-dispatch payload.
   - **Expected:** The dispatched message begins with a recall block and then includes the original user message.

4. Inspect the stored session event for the same turn.
   - **Expected:** The stored event contains only the original raw user message and does not include the recall block.

5. Validate recall bounds.
   - **Expected:** At most 3 recalled items are injected and the block stays within the documented character budget.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| No relevant matches | unrelated user message | No recall block is injected |
| Stale memory | old matched memory | Recall block includes freshness warning when applicable |
| High match volume | many relevant docs | Injection is truncated to bounded results/characters |

---

### Related Test Cases

- `TC-FUNC-002`
- `TC-REG-003`

---

### Notes

This is one of the highest-risk regressions because it affects transcript correctness and agent prompt fidelity at the same time.
