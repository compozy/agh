## TC-INT-006: CLI `agh task create --scope global --title "Test"` creates task via daemon API

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that the `agh task create` CLI command sends a well-formed CreateTaskRequest to the daemon UDS API, receives the created task, and renders the output correctly in both human-readable and JSON formats.

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] UDS server listening on `/tmp/.agh/daemon.sock`
- [ ] `agh` binary available in PATH
- [ ] No pre-existing tasks in the store

---

### Test Steps

1. **Create a global task with minimal flags**
   - Input:
     ```bash
     agh task create --scope global --title "CLI Test Task"
     ```
   - **Expected:** Exit code 0
   - **Expected:** Human-readable output contains:
     - "Task" section header
     - ID: a non-empty UUID
     - Scope: global
     - Title: CLI Test Task
     - Status: pending

2. **Create a global task with JSON output**
   - Input:
     ```bash
     agh task create --scope global --title "CLI JSON Test" --output json
     ```
   - **Expected:** Exit code 0
   - **Expected:** Valid JSON output parseable as TaskPayload
   - **Expected:** `id` is non-empty, `scope` is "global", `title` is "CLI JSON Test", `status` is "pending"
   - **Expected:** `created_by.kind` is "human", `origin.kind` is "uds" (CLI uses UDS transport)

3. **Create a workspace-scoped task with all flags**
   - Input:
     ```bash
     agh task create \
       --scope workspace \
       --workspace <valid-workspace-ref> \
       --title "Full CLI Task" \
       --description "Created via CLI with all options" \
       --identifier "cli-001" \
       --owner-kind human \
       --owner-ref alice \
       --metadata '{"source":"cli-test"}'
     ```
   - **Expected:** Exit code 0
   - **Expected:** Output shows all provided fields correctly
   - **Expected:** `scope` is "workspace", `workspace_id` is resolved, `identifier` is "cli-001"
   - **Expected:** `owner` is "human:alice", `metadata` contains `{"source":"cli-test"}`

4. **Verify created task exists via HTTP API**
   - Input: `GET http://localhost:2123/api/tasks` (or parse task ID from step 1 output)
   - **Expected:** The CLI-created tasks appear in the list
   - **Expected:** All field values match what the CLI reported

5. **Create with explicit ID**
   - Input:
     ```bash
     agh task create --scope global --title "Explicit ID Task" --id "custom-test-id-001"
     ```
   - **Expected:** Exit code 0
   - **Expected:** `id` in output equals "custom-test-id-001"

---

### Data Validation
| Field | CLI Output | API Verification | Status |
|-------|-----------|-----------------|--------|
| task.id | Non-empty UUID | Matches GET response | [ ] |
| task.scope | "global" | "global" | [ ] |
| task.title | "CLI Test Task" | "CLI Test Task" | [ ] |
| task.status | "pending" | "pending" | [ ] |
| task.created_by.kind | "human" | "human" | [ ] |
| task.origin.kind | "uds" | "uds" | [ ] |
| task.workspace_id (ws-scope) | Resolved ID | Matches | [ ] |
| task.owner (when set) | "human:alice" | Matches | [ ] |

---

### Error Scenarios
- [ ] Missing `--scope` flag exits with non-zero code and error message "scope is required"
- [ ] Missing `--title` flag exits with non-zero code and error message "title is required"
- [ ] `--scope workspace` without `--workspace` exits with error "workspace is required when scope is workspace"
- [ ] `--scope global` with `--workspace` exits with error "workspace must be empty when scope is global"
- [ ] Invalid `--scope bogus` exits with validation error
- [ ] Invalid `--metadata` (not valid JSON) exits with error about invalid JSON

---

### Related Test Cases
- TC-INT-001: HTTP POST /api/tasks validates the same contract the CLI calls
- TC-INT-005: UDS parity (CLI uses UDS transport)
- TC-INT-007: CLI list command verifies created tasks appear
