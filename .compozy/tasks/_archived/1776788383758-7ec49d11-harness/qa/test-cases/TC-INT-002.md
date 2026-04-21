## TC-INT-002: Ordered augmentation preserves canonical stored input and emits explicit observability

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 3
**Traceability:** `task_03.md`, ADR-002

---

### Objective

Validate that the daemon-composed prompt augmentation pipeline runs in
deterministic order, preserves the canonical stored user/network input, and
emits explicit observability for both successful augmentation and failure
paths.

---

### Preconditions

- [ ] A session can be started whose resolved policy enables prompt augmentation
- [ ] The execution lane can inspect both persisted session events and the message dispatched to the driver/runtime
- [ ] Observe/query surfaces can return harness lifecycle summaries for the session

---

### Test Steps

1. **Submit a prompt through a session whose resolved policy enables augmentation**
   - **Expected:** The prompt is accepted and the runtime attempts augmentation before driver dispatch.

2. **Inspect the persisted input event and the driver-dispatched prompt**
   - **Expected:** The stored event preserves the original user or network input exactly, while the dispatched prompt contains the augmenter-contributed content.

3. **Inspect observability for the successful path**
   - **Expected:** Harness summaries include `harness.context_resolved` and `harness.augmenter_applied` with the relevant session id and augmenter metadata.

4. **Exercise a warning-only augmentation failure through the supported regression harness**
   - **Expected:** Dispatch continues with the last valid prompt state, and the failure is recorded explicitly rather than silently skipped.

5. **Exercise a fail-fast augmentation failure through the supported regression harness**
   - **Expected:** Dispatch aborts before driver submission, the failure is visible to the caller/runtime, and no partial success is reported.

---

### Evidence to Capture

- Stored input excerpt and dispatched prompt excerpt for the same turn
- Ordered harness summaries for successful and failure scenarios
- Visible failure output for warning-only and fail-fast paths
- Any augmenter names or budget metadata exposed by the runtime

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Baseline successful augmentation | normal prompt with enabled augmenter(s) | stored input unchanged; dispatched input augmented |
| Network-origin prompt | prompt enters through network-aware path | same stored-input invariant holds |
| Blank augmenter output | augmenter returns empty/whitespace output | valid prompt is not clobbered |
| Warning-only failure | noncritical augmenter fails | warning path recorded; dispatch continues |
| Fail-fast failure | critical augmenter fails | dispatch aborted before driver submission |

---

### Related Test Cases

- `TC-INT-001`: Startup section selection follows resolved policy
- `TC-INT-007`: Harness observability and HTTP/UDS parity

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/session/manager_test.go`
- `internal/daemon/harness_observability_test.go`
- `internal/daemon/harness_context_test.go`
