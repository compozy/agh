# TC-FUNC-030 — Hook payloads use canonical `tool_id` (no `tool_name`/`tool_namespace` aliases)

- **Priority:** P1
- **Type:** Functional / hooks
- **Trace:** Task 04, ADR-007, TechSpec Delete Targets

## Objective

Prove registry-owned hook payloads expose `tool_id` and no `tool_name` / `tool_namespace` field. ACP permission-family matchers may still retain `tool_name` because that is the permission-request vocabulary.

## Test Steps

1. Trigger `tool.pre_call`, `tool.post_call`, `tool.post_error` hooks.
   - **Expected:** Each payload contains `tool_id`, `display_title`, `source`, `risk`, etc.; no top-level `tool_name`/`tool_namespace` for registry-owned tool calls.
2. Confirm matcher syntax in hook config uses `tool_id` patterns (canonical IDs and namespace wildcards).
3. Verify ACP permission-family matchers still accept `tool_name` for ACP permission-request payloads.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/hooks -run TestHookPayloadCanonicalToolID`
