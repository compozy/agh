# TC-INT-006: Extension fixture receives grants and nonce during initialize

**Priority:** P0
**Type:** Integration
**Package:** internal/extension
**Related Tasks:** 04, 05

## Objective

Validate that a real extension subprocess receives the correct grants and session nonce during the `initialize` handshake. The initialize response must contain `session_nonce`, `granted_resource_kinds`, and `granted_resource_scopes` — these are the extension's authorization tokens for subsequent resource operations.

## Preconditions

- A test extension binary or script that implements the extension protocol (JSON-RPC over stdio)
- The extension fixture reads initialize params and echoes them back or writes them to a temp file for verification
- Real SQLite database via `t.TempDir()` with resource tables and source state
- Extension manager/host initialized and configured with grants for the test extension

## Test Steps

1. Configure the extension host with a test extension definition that declares `resource_kinds=["tool", "hook.binding"]` and `resource_scopes=["session", "global"]`.
   **Expected:** Configuration accepted. Extension registered in the host.

2. Launch the extension subprocess via the extension host's start/initialize flow.
   **Expected:** Subprocess spawns successfully. JSON-RPC connection established over stdio.

3. Capture the `initialize` request sent to the extension subprocess.
   **Expected:** Request includes `session_nonce` (non-empty string), `granted_resource_kinds` containing `["tool", "hook.binding"]`, and `granted_resource_scopes` containing `["session", "global"]`.

4. Verify the `session_nonce` is a cryptographically random value (not empty, not zero, sufficient entropy).
   **Expected:** Nonce is at least 16 characters and appears random (no obvious pattern).

5. Verify the `session_nonce` is persisted in `resource_source_state` for this extension's source identifier.
   **Expected:** `resource_source_state` row exists with matching nonce.

6. Shut down the extension subprocess cleanly.
   **Expected:** Process exits with code 0. No orphan processes.

## Edge Cases

- Extension with zero granted kinds — initialize still succeeds but grants are empty arrays
- Extension binary does not exist — clear error returned, no panic
- Extension binary crashes during initialize — error propagated, nonce not persisted
- Initialize timeout — extension that hangs during initialize is killed after deadline
- Nonce uniqueness — two sequential initializations for the same extension produce different nonces
