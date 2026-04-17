# TC-SEC-003: Workspace-Scoped Extension Cannot Publish Global Resources

**Priority:** P0
**Type:** Security
**Package:** internal/resources
**Related Tasks:** 01, 04

## Objective

Validate that scope enforcement prevents an extension declared with `MaxScope=workspace` from publishing resources at the global scope. The runtime must reject such attempts with 403 before any persistence occurs.

## Preconditions

- Extension `ext-limited` is registered with `MaxScope=workspace` in its manifest/configuration.
- Extension `ext-global` is registered with `MaxScope=global` (for comparison).
- Both extensions have active sessions with valid nonces.
- A workspace context is established for the current session.

## Test Steps

1. As `ext-limited`, submit a snapshot containing a record with `scope=global`.
   **Expected:** The snapshot is rejected with 403 Forbidden. The error message indicates a scope violation -- the extension's max scope does not permit global publication.

2. Verify no records from step 1 were persisted by querying the global resource store.
   **Expected:** No records from `ext-limited` exist at global scope. The rejection occurred before any write.

3. As `ext-limited`, submit a snapshot containing a record with `scope=workspace`.
   **Expected:** The snapshot succeeds. The extension can publish within its authorized scope.

4. As `ext-global`, submit a snapshot containing a record with `scope=global`.
   **Expected:** The snapshot succeeds. An extension with sufficient MaxScope can publish globally.

5. As `ext-limited`, submit a snapshot containing a mix of workspace-scoped and global-scoped records.
   **Expected:** The entire snapshot is rejected with 403. No partial application occurs -- the workspace-scoped records are also not persisted.

## Edge Cases

- Extension modifies its own manifest to upgrade `MaxScope` at runtime (manifest tampering).
- Extension omits the `scope` field entirely, relying on a default that might be `global`.
- Extension sets `scope` to an unrecognized string value (e.g., `"universal"`) to probe for permissive fallback behavior.
- Extension with `MaxScope=session` attempts `scope=workspace` publication (stricter scope boundary).
- Nested scope escalation: extension publishes at workspace, then attempts to "promote" the record to global via a subsequent snapshot.

## Threat Model

This test prevents **vertical privilege escalation through scope inflation**. A workspace-scoped extension is intentionally restricted to avoid affecting resources beyond its workspace boundary. If an extension could bypass this restriction, it could inject global tool definitions or hook bindings that affect all workspaces and all sessions, amplifying the blast radius of a compromised or malicious extension from a single workspace to the entire daemon.
