## TC-INT-007: CLI `agh task list --status ready --scope workspace` returns filtered results

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that the `agh task list` CLI command correctly translates filter flags into TaskListQuery parameters, sends them to the daemon UDS API, and renders the filtered task list in both human-readable table and JSON formats.

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] UDS server listening on `/tmp/.agh/daemon.sock`
- [ ] `agh` binary available in PATH
- [ ] At least one workspace registered
- [ ] Seed data: create the following tasks before running tests:
  - Task A: scope=global, status=pending, title="Global Pending"
  - Task B: scope=workspace, workspace=<ws>, title="WS Pending" (status=pending after creation)
  - Task C: scope=workspace, workspace=<ws>, title="WS Ready" (advance to ready status by satisfying dependencies or direct state)
  - Task D: scope=global, status=pending, owner={kind:"human", ref:"alice"}, title="Owned Global"
  - Task E: scope=workspace, workspace=<ws>, network_channel="ch-qa", title="WS Channeled"

---

### Test Steps

1. **List all tasks with no filters**
   - Input: `agh task list`
   - **Expected:** Exit code 0
   - **Expected:** Human-readable table output with columns: ID, Identifier, Scope, Workspace, Parent, Status, Owner, Channel, Title
   - **Expected:** All 5 seeded tasks appear in the output

2. **Filter by scope=workspace**
   - Input: `agh task list --scope workspace`
   - **Expected:** Exit code 0
   - **Expected:** Only tasks B, C, E appear (workspace-scoped tasks)
   - **Expected:** Tasks A, D excluded (global-scoped)

3. **Filter by status=ready**
   - Input: `agh task list --status ready`
   - **Expected:** Exit code 0
   - **Expected:** Only Task C appears (the one with ready status)

4. **Combined filter: scope=workspace AND status=ready**
   - Input: `agh task list --scope workspace --status ready`
   - **Expected:** Exit code 0
   - **Expected:** Only Task C appears

5. **Filter by workspace reference**
   - Input: `agh task list --scope workspace --workspace <ws-ref>`
   - **Expected:** Exit code 0
   - **Expected:** Tasks B, C, E appear (all workspace-scoped tasks in that workspace)

6. **Filter by owner**
   - Input: `agh task list --owner-kind human --owner-ref alice`
   - **Expected:** Exit code 0
   - **Expected:** Only Task D appears

7. **Filter by channel**
   - Input: `agh task list --channel ch-qa`
   - **Expected:** Exit code 0
   - **Expected:** Only Task E appears

8. **Apply --last limit**
   - Input: `agh task list --last 2`
   - **Expected:** Exit code 0
   - **Expected:** Exactly 2 tasks returned

9. **JSON output with filters**
   - Input: `agh task list --scope global --output json`
   - **Expected:** Exit code 0
   - **Expected:** Valid JSON array of TaskSummaryPayload objects
   - **Expected:** All entries have `scope = "global"`

10. **Empty result set**
    - Input: `agh task list --status completed`
    - **Expected:** Exit code 0
    - **Expected:** Output indicates no tasks found (empty table or empty JSON array)

---

### Data Validation
| Field | Filter Applied | Expected Count | Status |
|-------|---------------|----------------|--------|
| No filter | none | 5 | [ ] |
| scope=workspace | --scope workspace | 3 | [ ] |
| status=ready | --status ready | 1 | [ ] |
| scope+status | --scope workspace --status ready | 1 | [ ] |
| workspace ref | --scope workspace --workspace <ref> | 3 | [ ] |
| owner | --owner-kind human --owner-ref alice | 1 | [ ] |
| channel | --channel ch-qa | 1 | [ ] |
| limit | --last 2 | 2 | [ ] |
| no matches | --status completed | 0 | [ ] |

---

### Error Scenarios
- [ ] Invalid `--status bogus` exits with non-zero code and validation error
- [ ] Invalid `--scope bogus` exits with non-zero code and validation error
- [ ] `--owner-kind` without `--owner-ref` exits with error (must be provided together)
- [ ] `--owner-ref` without `--owner-kind` exits with error (must be provided together)
- [ ] Negative `--last -1` exits with error
- [ ] `--scope global --workspace some-ws` exits with error (workspace must be empty for global scope)

---

### Related Test Cases
- TC-INT-002: HTTP GET /api/tasks with query filters (same filtering logic)
- TC-INT-005: UDS endpoint parity
- TC-INT-006: CLI create command used to seed test data
