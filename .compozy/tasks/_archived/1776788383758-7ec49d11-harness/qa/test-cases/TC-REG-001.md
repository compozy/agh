## TC-REG-001: Restart and recovery preserve detached-runtime state without duplicate synthetic wakeups

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 5 and Workstream 6
**Traceability:** `task_06.md`, `task_07.md`, `task_08.md`, ADR-003

---

### Objective

Validate that a restart or recovery boundary preserves detached harness
metadata, reuses task-runtime recovery semantics, and does not emit duplicate
synthetic wakeups for already-observed completions.

---

### Preconditions

- [ ] Detached harness work can be submitted and completed against an isolated AGH home/workspace
- [ ] The daemon/runtime lane supports a stop-and-restart cycle during the scenario
- [ ] Query surfaces can inspect task runs, session events, and harness summaries before and after restart

---

### Test Steps

1. **Create detached harness work and drive it to a state that should be recoverable or already observable**
   - **Expected:** The task/task-run metadata and any emitted completion evidence are persisted durably before restart.

2. **Capture the pre-restart runtime state**
   - **Expected:** The current task-run status, relevant `harness.*` summaries, and any emitted synthetic event ids are recorded for later comparison.

3. **Stop the daemon and restart it through the normal boot path**
   - **Expected:** The daemon boots successfully and reuses the task-runtime recovery rules instead of a harness-only recovery path.

4. **Inspect post-restart task-runtime and observe surfaces**
   - **Expected:** Detached metadata is preserved, recovery state is consistent with the pre-restart record, and previously emitted completions are still visible.

5. **Verify duplicate-protection after restart**
   - **Expected:** The same `task_run` does not emit an additional synthetic wakeup, and the post-restart transcript/event history remains consistent.

6. **Compare HTTP and UDS read-side visibility after restart**
   - **Expected:** Both transports continue to expose the same harness-visible recovery outcome.

---

### Evidence to Capture

- Pre-restart and post-restart task-run status snapshots
- Pre-restart and post-restart `harness.*` summary excerpts
- Any synthetic event ids used for dedupe comparison
- HTTP/UDS read-side output proving parity after restart
- Final note confirming whether duplicate wake protection held

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Recoverable detached run | run exists at restart boundary | task-runtime recovery semantics apply |
| Previously emitted wake | synthetic wake already persisted before restart | no duplicate wake emitted after restart |
| Silent/drop outcome | completion was intentionally non-waking | drop remains observable after restart |
| Mixed transport inspection | post-restart state read over HTTP and UDS | parity preserved |

---

### Related Test Cases

- `TC-INT-005`: Detached harness work persists on the task runtime with stable metadata
- `TC-INT-006`: Detached completion wakes the owning session or records an explicit drop
- `TC-INT-007`: Harness lifecycle observability stays queryable and transport-parity safe

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/daemon/daemon_integration_test.go`
- `internal/daemon/task_runtime_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`
