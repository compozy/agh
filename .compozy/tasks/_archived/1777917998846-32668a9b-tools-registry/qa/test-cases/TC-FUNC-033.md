# TC-FUNC-033 — Telemetry events include redacted required fields

- **Priority:** P1
- **Type:** Functional / observability
- **Trace:** Task 04, TechSpec Monitoring and Observability

## Objective

Prove every emitted event (`tool.registered`, `tool.updated`, `tool.removed`, `tool.conflicted`, `tool.availability_changed`, `tool.policy_evaluated`, `tool.call_started`, `tool.call_completed`, `tool.call_failed`, `tool.call_denied`, `tool.result_truncated`) carries the required fields and redacts secrets.

## Test Steps

1. Invoke a tool successfully.
   - **Expected:** `tool.call_started` and `tool.call_completed` recorded with `tool_id`, `display_title`, `source_kind`, `source_owner`, `workspace_id`, `session_id`, `parent_session_id`, `root_session_id`, `agent_name`, `risk`, `read_only`, `destructive`, `open_world`, `approval_mode`, `decision`, `reason_codes`, `duration_ms`, `result_bytes`, `truncated`, `correlation_id`.
2. Trigger conflict registration → `tool.conflicted` event with provenance.
3. Trigger denial → `tool.call_denied` with denying layer reason.
4. Trigger truncation → `tool.result_truncated`.
5. Sentinel scan across event payloads.
   - **Expected:** No raw tokens, no raw inputs marked sensitive.

## Automation

- **Target:** Integration
- **Status:** Existing partial
- **Command/Spec:** `go test ./internal/observe -run TestToolEvents`
