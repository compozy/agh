# TC-FUNC-002: Scope validation rejects invalid combinations

**Priority:** P0
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that the resource store enforces strict scope validation rules on every mutation. The three scope-related invariants are: (1) a `global` scope must have an empty `scope_id`, (2) a `workspace` scope must have a non-empty `scope_id`, and (3) omitting the scope entirely is rejected. This prevents records from being stored with ambiguous or inconsistent scoping.

## Preconditions

- A fresh resource store is initialized with schema applied.
- A valid `MutationActor` is configured.
- At least one resource kind is registered.

## Test Steps

1. Call `PutRaw` with `Scope="global"` and `ScopeID=""` (empty string).
   **Expected:** The call succeeds. The returned record has `Scope="global"` and `ScopeID=""`.

2. Call `PutRaw` with `Scope="global"` and `ScopeID="ws-123"` (non-empty).
   **Expected:** The call returns a validation error before any write occurs. The error clearly indicates that `global` scope must not have a `scope_id`.

3. Call `PutRaw` with `Scope="workspace"` and `ScopeID="ws-123"` (non-empty).
   **Expected:** The call succeeds. The returned record has `Scope="workspace"` and `ScopeID="ws-123"`.

4. Call `PutRaw` with `Scope="workspace"` and `ScopeID=""` (empty string).
   **Expected:** The call returns a validation error before any write occurs. The error clearly indicates that `workspace` scope requires a non-empty `scope_id`.

5. Call `PutRaw` with `Scope=""` (empty/omitted) and any `ScopeID`.
   **Expected:** The call returns a validation error. The error indicates that scope must be explicitly specified.

6. Call `PutRaw` with `Scope="session"` and `ScopeID="sess-abc"`.
   **Expected:** The call succeeds if `session` is a valid scope, or returns a validation error if only `global` and `workspace` are allowed. The behavior matches the defined scope enum.

## Edge Cases

- Scope value with leading/trailing whitespace (e.g., `" global "`) is either trimmed and accepted or rejected as invalid, but never stored with whitespace.
- Scope value in wrong case (e.g., `"Global"`, `"WORKSPACE"`) is rejected since scope values should be case-sensitive lowercase enums.
- A valid create with `Scope="workspace"` followed by an update attempt that changes the scope to `global` (or vice versa) is rejected, since scope is immutable after creation.
- `ScopeID` containing only whitespace (e.g., `"   "`) for a `workspace` scope is rejected as effectively empty.
