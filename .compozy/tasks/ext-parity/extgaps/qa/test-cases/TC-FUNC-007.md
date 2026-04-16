# TC-FUNC-007: Global-scope activation propagates scope to all resources

**Priority:** P1 (High)
**Type:** Functional
**Component:** `internal/bundles/service.go` — scope mapping functions

## Objective

Validate that global-scope activations correctly propagate scope to all materialized automation jobs, triggers, and bridge instances.

## Preconditions

- Extension with bundle containing jobs, triggers, and bridges
- Bundle service initialized with workspace resolver

## Test Steps

1. Activate bundle with `Scope: "global"`, `Workspace: ""`
   **Expected:** Activation created with Scope="global", WorkspaceID=""

2. Verify materialized jobs have `Scope: AutomationScopeGlobal`
   **Expected:** All jobs use global automation scope

3. Verify materialized triggers have `Scope: AutomationScopeGlobal`
   **Expected:** All triggers use global automation scope

4. Verify materialized bridges have `Scope: ScopeGlobal`
   **Expected:** All bridge instances use global bridge scope

5. Verify all resources have empty WorkspaceID
   **Expected:** WorkspaceID="" for all materialized resources

## Edge Cases

- Global scope with non-empty workspace ref → validation error: "global activation cannot include workspace id"
- Scope string "GLOBAL" (uppercase) → normalized to "global"
- Empty scope defaults to "global"
