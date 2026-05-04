# TC-INT-006 — Hooks deny / patch / redact cross-surface

- **Priority:** P2
- **Type:** Integration / hooks
- **Trace:** Task 04

## Test Steps

1. Configure a hook that denies a specific `tool_id`.
2. Trigger via CLI / HTTP / UDS / hosted MCP.
   - **Expected:** All four surfaces return `hook_denied` with same reason code structure.
3. Configure a hook that patches input.
   - **Expected:** Provider receives patched input regardless of surface.
4. Configure a hook that redacts result.
   - **Expected:** All four surfaces return redacted result envelope.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/hooks -run TestHooksCrossSurface`
