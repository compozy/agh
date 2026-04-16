# TC-INT-010: Permission hooks fire end to end

**Priority:** P1
**Type:** Integration
**Package:** internal/hooks, internal/session
**Related Tasks:** 07

## Objective

Validate that resource-backed hook bindings for the permission lifecycle events (`permission.request`, `permission.resolved`, `permission.denied`) all fire correctly through the session runtime. This ensures the full permission hook chain works with the new resource-backed dispatch system.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with hook projector
- Session runtime with permission flow support
- Three test hook handlers, one for each permission event, each recording invocations independently

## Test Steps

1. Persist three `hook.binding` resource records:
   - Binding A: `event=permission.request`
   - Binding B: `event=permission.resolved`
   - Binding C: `event=permission.denied`
   **Expected:** All three records stored. Projector reconciled.

2. Trigger a permission request flow that results in approval (resolved).
   **Expected:** `permission.request` hook fires first, then `permission.resolved` hook fires.

3. Verify handler invocation counts: A=1, B=1, C=0.
   **Expected:** `permission.request` and `permission.resolved` each fired once. `permission.denied` did not fire.

4. Trigger a permission request flow that results in denial.
   **Expected:** `permission.request` hook fires, then `permission.denied` hook fires.

5. Verify handler invocation counts: A=2, B=1, C=1.
   **Expected:** `permission.request` fired again (total 2). `permission.denied` fired once. `permission.resolved` count unchanged.

6. Verify each hook received the correct event payload (permission type, resource identifier, decision, etc.).
   **Expected:** Payloads contain the permission context — what was requested and the outcome.

## Edge Cases

- Permission resolved and denied hooks both registered but only one fires per flow — no double-fire
- Permission request hook modifies the request (if supported) — downstream hooks see the modified version
- Hook handler panics — recovered, other hooks still fire, session not crashed
- All three bindings removed mid-flow — hooks that already matched still complete, removed bindings do not fire for next flow
- High-frequency permission requests — hooks fire for each, no coalescing of permission events
