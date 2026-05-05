# TC-FUNC-054 — `agh tool info <id>` deterministic error structures

- **Priority:** P1
- **Type:** Functional / CLI
- **Trace:** Task 12

## Test Steps

1. `agh tool info agh__skill_view -o json`.
   - **Expected:** Descriptor + availability + source provenance.
2. `agh tool info bad-id -o json`.
   - **Expected:** Exit non-zero; structured error includes `code`, `message`, `reason_codes`.
3. `agh tool info <unavailable-mcp-tool> -o json`.
   - **Expected:** Operator view with `mcp_auth_required`/`expired`/etc. truthful reasons; redacted auth_status.
4. `agh tool info <conflicted-id>`.
   - **Expected:** Includes `conflicted_id` reason and provenance for both providers.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/cli -run TestToolInfoCommand`
