# TC-FUNC-008: Workspace-scope activation resolves workspace and scopes resources

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — `resolveWorkspace()`, scope mapping

## Objective

Validate that workspace-scoped activations correctly resolve the workspace reference, generate workspace-specific stable IDs, and propagate workspace scope to all resources.

## Preconditions

- Workspace resolver configured
- Known workspace exists (e.g., "/home/user/project")
- Extension with bundle containing jobs, triggers, and bridges

## Test Steps

1. Activate with `Scope: "workspace"`, `Workspace: "/home/user/project"`
   **Expected:** Workspace resolved to ID, activation created with WorkspaceID populated

2. Verify stable ID includes workspace ID
   **Expected:** Different ID than global activation for same ext/bundle/profile

3. Verify materialized jobs have `Scope: AutomationScopeWorkspace` and correct WorkspaceID
   **Expected:** Workspace scope and ID propagated

4. Verify materialized bridges have `Scope: ScopeWorkspace` and correct WorkspaceID
   **Expected:** Workspace scope and ID propagated

5. Activate same bundle in same workspace again (idempotent)
   **Expected:** Same activation ID returned

6. Activate same bundle in different workspace
   **Expected:** Different activation ID (different workspace in hash)

## Edge Cases

- Workspace scope with empty workspace ref → error: "workspace reference is required"
- Workspace scope with non-existent workspace path → workspace resolver creates via ResolveOrRegister
- Workspace ref with "~" prefix → expanded to absolute path
- Workspace ref with relative path "./project" → resolved via isPathLikeWorkspaceRef
- Workspace ref that is a UUID (non-path) → resolved via Resolve (not ResolveOrRegister)
- Workspace resolver is nil → error: "workspace resolver is required"
