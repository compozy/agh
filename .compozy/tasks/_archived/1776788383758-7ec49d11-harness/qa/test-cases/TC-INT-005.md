## TC-INT-005: Detached harness work persists on the task runtime with stable metadata and idempotency

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 5
**Traceability:** `task_06.md`, ADR-003

---

### Objective

Validate that detached harness work reuses the existing `task` / `task_run`
substrate with explicit harness metadata, stable query visibility, and
idempotent submission semantics across supported scopes.

---

### Preconditions

- [ ] The daemon/runtime lane can submit detached harness work through the daemon-owned bridge
- [ ] Normal task and task-run query surfaces are available
- [ ] The execution lane can inspect persisted metadata for tasks and runs

---

### Test Steps

1. **Submit one detached harness work item targeting the global scope**
   - **Expected:** A `task` and `task_run` are created through the normal task-runtime substrate with harness-specific origin and metadata.

2. **Inspect the persisted task and task-run records**
   - **Expected:** Owner-session linkage, wake targeting, workspace/global scope, schema metadata, and origin fields are all present and readable through normal query surfaces.

3. **Repeat the submission with the same idempotent identity**
   - **Expected:** The runtime reuses the existing durable record or rejects the duplicate cleanly according to task-runtime idempotency rules; it does not create a second detached run silently.

4. **Submit one detached harness work item targeting a workspace scope**
   - **Expected:** Workspace-bound metadata persists correctly and remains distinguishable from the global-scope run.

5. **Attempt a mismatched or invalid detached submission**
   - **Expected:** Validation fails cleanly and no partial detached record is persisted.

---

### Evidence to Capture

- Task id and task-run id for global and workspace submissions
- Metadata JSON excerpts proving owner-session and wake-target persistence
- Query output proving the same surfaces used for ordinary task/runtime inspection work here
- Duplicate-submission evidence showing idempotent behavior

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Global detached run | no workspace binding | persisted as global with harness metadata |
| Workspace detached run | workspace id provided | persisted with workspace binding |
| Idempotent resubmission | same request identity repeated | no duplicate durable run |
| Invalid target session | missing or inconsistent owner-session fields | validation failure |
| Metadata mismatch | existing detached record conflicts with new request | rejection or conflict, not silent reuse |

---

### Related Test Cases

- `TC-INT-006`: Detached completion wakes or drops through the reentry bridge
- `TC-REG-001`: Recovery and duplicate-protection stay correct across restart

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/daemon/task_runtime_test.go`
- `internal/daemon/daemon_integration_test.go`
