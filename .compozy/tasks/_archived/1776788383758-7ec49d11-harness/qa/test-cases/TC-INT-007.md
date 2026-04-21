## TC-INT-007: Harness lifecycle observability stays queryable and transport-parity safe

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 6
**Traceability:** `task_08.md`, ADR-002, ADR-003, ADR-004

---

### Objective

Validate that harness lifecycle decisions remain visible through the existing
`event_summaries` and observe/query surfaces, and that HTTP and UDS transports
expose the same ordered harness evidence for the same runtime flow.

---

### Preconditions

- [ ] One end-to-end harness runtime flow is available that exercises startup, augmentation, detached completion, and reentry
- [ ] HTTP and UDS observe/query surfaces are available against the same runtime state
- [ ] The execution lane can inspect ordered harness lifecycle summaries

---

### Test Steps

1. **Execute one harness flow that exercises startup selection, augmentation, detached completion, and synthetic reentry**
   - **Expected:** The flow completes far enough to leave visible lifecycle evidence for all major harness seams.

2. **Query harness lifecycle evidence through the HTTP observe surface**
   - **Expected:** The response includes ordered `harness.*` summaries such as `harness.context_resolved`, `harness.section_selected`, `harness.augmenter_applied` or `harness.augmenter_failed`, `harness.detached_run_completed`, and `harness.synthetic_reentry_emitted` or `harness.synthetic_reentry_dropped`.

3. **Query the equivalent evidence through the UDS surface**
   - **Expected:** The UDS response shows the same ordered harness lifecycle sequence and the same session/task identifiers for shared fields.

4. **Inspect a stream or follow-style read surface after the flow**
   - **Expected:** The same harness lifecycle payloads remain visible through the read-side transport helpers and are not lost after the initial query.

5. **Verify startup summary association for the created session**
   - **Expected:** Startup summaries remain attached to the correct session even when selection happened before the session row was fully available.

---

### Evidence to Capture

- Ordered HTTP harness summary response
- Ordered UDS harness summary response
- A direct comparison showing parity (or the exact divergence if failed)
- Session ids and any task/task-run ids present in the summary payloads
- One stream/read-side excerpt confirming harness payload visibility

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Successful full flow | startup + augmentation + wake path | emitted summaries visible in order |
| Drop path | completion does not wake target session | `harness.synthetic_reentry_dropped` visible |
| Failure path | warning/failure augmentation scenario | `harness.augmenter_failed` visible without transport drift |
| Startup-before-session-row | startup summaries queued until session creation | final summaries still attach to correct session |
| HTTP vs UDS | same runtime flow inspected over both transports | parity preserved |

---

### Related Test Cases

- `TC-INT-001`: Startup section selection follows resolved harness policy
- `TC-INT-002`: Ordered augmentation preserves canonical stored input
- `TC-INT-006`: Detached completion wakes the owning session or records an explicit drop

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/observe/observer_test.go`
- `internal/api/httpapi/stream_helpers_test.go`
- `internal/api/udsapi/stream_helpers_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`
