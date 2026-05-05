# TC-FUNC-006: Automation Tool Family CRUD, Trigger, And Run Inspection

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the `agh__automation_*` family routes through the existing `automation.Manager` and validators, that CRUD/trigger/run-inspection paths match the existing CLI/HTTP/UDS behavior, and that approval/source/scope policy denials surface deterministic reason codes.

## Traceability

- Task: task_07 (Automation Tool Family).
- TechSpec: "Mutable Surface Policy → agh__automation", "Agent Manageability Plan", "Implementation Steps".
- ADR: ADR-006.
- Surfaces: `internal/tools/builtin/automation.go`, `internal/automation/{manager.go,validate.go,persistence.go}`, `internal/api/core/automation.go`, `internal/cli/automation.go`, `internal/daemon/automation_resources.go`.

## Preconditions

- Isolated `AGH_HOME`.
- `globaldb` provisioned with at least one fixture job and one fixture trigger from a deterministic seed.
- Approval channel auto-approves mutating calls in this lab.

## Test Steps

1. **Read parity:**
   ```bash
   agh automation jobs list -o json | tee qa/logs/TC-FUNC-006/cli-jobs-list.json
   agh tool invoke agh__automation_jobs_list -o json | tee qa/logs/TC-FUNC-006/tool-jobs-list.json
   diff <(jq -S . qa/logs/TC-FUNC-006/cli-jobs-list.json) \
        <(jq -S . qa/logs/TC-FUNC-006/tool-jobs-list.json)

   agh automation triggers list -o json | tee qa/logs/TC-FUNC-006/cli-triggers-list.json
   agh tool invoke agh__automation_triggers_list -o json | tee qa/logs/TC-FUNC-006/tool-triggers-list.json
   ```
   Repeat with `*_get` tools for the seeded job/trigger IDs.
   - **Expected:** Identical output across surfaces.

2. **Job lifecycle:**
   ```bash
   agh tool invoke agh__automation_jobs_create --input '{
     "name":"qa-job",
     "schedule":"@every 5m",
     "action":{"type":"noop"}
   }' -o json | tee qa/logs/TC-FUNC-006/tool-jobs-create.json

   agh tool invoke agh__automation_jobs_update --input '{
     "id":"qa-job",
     "schedule":"@every 10m"
   }' -o json | tee qa/logs/TC-FUNC-006/tool-jobs-update.json

   agh tool invoke agh__automation_jobs_trigger --input '{"id":"qa-job"}' -o json \
     | tee qa/logs/TC-FUNC-006/tool-jobs-trigger.json

   agh tool invoke agh__automation_jobs_history --input '{"id":"qa-job"}' -o json \
     | tee qa/logs/TC-FUNC-006/tool-jobs-history.json

   agh tool invoke agh__automation_jobs_delete --input '{"id":"qa-job"}' -o json \
     | tee qa/logs/TC-FUNC-006/tool-jobs-delete.json
   ```
   - **Expected:** Each step persists to `automation` tables and matches the CLI `agh automation jobs create|update|delete|trigger|history` behavior. Manual trigger writes a run row.

3. **Trigger lifecycle:**
   ```bash
   agh tool invoke agh__automation_triggers_create --input '{
     "name":"qa-trigger",
     "type":"webhook",
     "ref":{"path":"/webhook/qa"}
   }' -o json | tee qa/logs/TC-FUNC-006/tool-triggers-create.json

   agh tool invoke agh__automation_triggers_history --input '{"id":"qa-trigger"}' -o json \
     | tee qa/logs/TC-FUNC-006/tool-triggers-history.json

   agh tool invoke agh__automation_triggers_delete --input '{"id":"qa-trigger"}' -o json \
     | tee qa/logs/TC-FUNC-006/tool-triggers-delete.json
   ```

4. **Run inspection:**
   ```bash
   agh tool invoke agh__automation_runs_list -o json | tee qa/logs/TC-FUNC-006/tool-runs-list.json
   agh automation runs list -o json | tee qa/logs/TC-FUNC-006/cli-runs-list.json
   diff <(jq -S . qa/logs/TC-FUNC-006/tool-runs-list.json) \
        <(jq -S . qa/logs/TC-FUNC-006/cli-runs-list.json)
   ```

5. **Denial taxonomy:**
   - Submit a job with secret-bearing webhook material:
     ```bash
     agh tool invoke agh__automation_jobs_create --input '{
       "name":"qa-secret",
       "schedule":"@every 5m",
       "action":{"type":"webhook","secret":"hunter2"}
     }' -o json | tee qa/logs/TC-FUNC-006/tool-jobs-create-secret.json
     ```
     - **Expected:** `AUTOMATION_SECRET_INPUT_FORBIDDEN`.
   - Submit a job with invalid schedule:
     - **Expected:** `AUTOMATION_VALIDATION_FAILED`.
   - With approval disabled, retry Step 2's `create`:
     - **Expected:** `AUTOMATION_APPROVAL_REQUIRED` (or `approval_unreachable`).
   - Force scope denial (bootstrap-time scheduler trust-root mutation):
     - **Expected:** `AUTOMATION_SCOPE_FORBIDDEN`.

6. **HTTP/UDS parity:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     -d '{"name":"qa-uds","schedule":"@every 5m","action":{"type":"noop"}}' \
     "http://localhost/api/tools/agh__automation_jobs_create/invoke" | tee qa/logs/TC-FUNC-006/uds-jobs-create.json
   ```

7. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestAutomation" -count=1 | tee qa/logs/TC-FUNC-006/builtin-tests.log
   go test ./internal/automation -count=1 | tee qa/logs/TC-FUNC-006/automation-tests.log
   ```

## Evidence To Capture

- All `qa/logs/TC-FUNC-006/cli-*` and `tool-*` JSON.
- DB rows (sqlite query logs) showing the new job/trigger/run rows after each lifecycle step.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Trigger a job that does not exist | `agh__automation_jobs_trigger {"id":"missing"}` | Deterministic `not_found` error |
| Update a config-backed job vs package-backed job | both kinds | Both manageable when current overlay is the owner; package-backed denies if owner is immutable |
| Run history pagination | `--limit 10 --offset 10` | Tool returns paginated slice matching CLI |
| Concurrent tool + CLI create | sequential per CLAUDE.md | Single-success semantics; no parallel execution |

## Channels Exercised

- Tool invoke, CLI, HTTP/UDS.
- Persisted automation tables.

## Related Test Cases

- TC-INT-002 (transport parity).
- TC-UI-001 (web automation panel after DTO changes).
