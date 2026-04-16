# SMOKE-004: Extension Handshake Includes Resource Grants

**Priority:** P0
**Type:** Smoke
**Package:** internal/extension
**Related Tasks:** 04, 05

## Objective

Validate that when an extension session is started, the initialize response includes the correct resource grants (granted_resource_kinds, granted_resource_scopes) and a session_nonce computed from daemon policy. This confirms the extension protocol correctly negotiates capabilities during handshake, preventing unauthorized resource access.

## Preconditions

- Daemon running with a defined extension policy that grants specific resource kinds and scopes
- An extension manifest declaring requested resource kinds (e.g., "hook.binding", "tool.definition") and scopes (e.g., "session", "workspace")
- A test extension client capable of performing the initialize handshake

## Test Steps

1. **Configure daemon policy** to allow resource kinds=["hook.binding", "tool.definition"] and scopes=["session", "workspace"] for the test extension.
   **Expected:** Policy is accepted without validation errors.

2. **Start an extension session** by sending an initialize request from the test extension client.
   **Expected:** The daemon responds with a successful initialize response (no error).

3. **Inspect the initialize response** for the granted_resource_kinds field.
   **Expected:** Contains exactly ["hook.binding", "tool.definition"] (or the subset granted by policy). No extra kinds are granted beyond what the policy allows.

4. **Inspect the initialize response** for the granted_resource_scopes field.
   **Expected:** Contains exactly ["session", "workspace"]. No scopes beyond policy are granted.

5. **Inspect the initialize response** for the session_nonce field.
   **Expected:** session_nonce is a non-empty string, is unique per session (different from a previous session's nonce), and is deterministically derived from the daemon policy and session parameters.

6. **Attempt a resource operation (PutRaw)** using a kind not in granted_resource_kinds (e.g., "automation.job").
   **Expected:** The operation is rejected with a permission/authorization error referencing the missing grant.

7. **Attempt a resource operation (PutRaw)** using a scope not in granted_resource_scopes (e.g., "global").
   **Expected:** The operation is rejected with a permission/authorization error referencing the missing scope grant.

## Edge Cases

- An extension requesting zero resource kinds receives an empty granted_resource_kinds and cannot write any resources
- An extension requesting kinds the policy does not allow receives only the intersection of requested and allowed kinds
- The session_nonce changes across daemon restarts even if the same extension reconnects
- An initialize request with a malformed manifest returns a clear validation error, not a partial grant
- Two concurrent extension sessions receive different session_nonce values
- An extension that does not request any scopes receives no scope grants and is limited to scopeless operations (if any exist)
- Policy changes after session initialization do not retroactively modify the grants for an active session
