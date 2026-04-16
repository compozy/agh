# SMOKE-006: Hook Binding Fires Through Resource-Backed Dispatch

**Priority:** P0
**Type:** Smoke
**Package:** internal/hooks
**Related Tasks:** 07

## Objective

Validate that persisting a hook.binding resource record and triggering reconciliation causes the hook dispatcher to recognize and fire the binding when the matching event occurs. This confirms the migration from the old hook catalog to resource-backed hook dispatch is functionally complete.

## Preconditions

- Resource store initialized with the hook.binding kind codec registered
- Reconcile driver configured with the hooks projector
- Hook dispatcher wired to read bindings from the resource store
- A test hook handler registered for the target event (e.g., "session.started")

## Test Steps

1. **Persist a hook.binding resource** with kind="hook.binding", spec containing event="session.started", handler reference pointing to the test handler, and scope="workspace".
   **Expected:** Record is created with version=1. No errors.

2. **Trigger reconciliation** so the hooks projector processes the new binding.
   **Expected:** Reconciliation completes without error. The hooks projector registers the binding in its internal dispatch table.

3. **Emit a "session.started" event** through the hook dispatcher.
   **Expected:** The test handler is invoked exactly once. The handler receives the correct event payload.

4. **Verify handler execution** by checking the test handler's call log or channel.
   **Expected:** Handler was called with event type="session.started" and the expected context (session ID, workspace, etc.).

5. **Delete the hook.binding resource** and trigger reconciliation again.
   **Expected:** Reconciliation completes. The binding is removed from the dispatch table.

6. **Emit the same "session.started" event** again.
   **Expected:** The test handler is NOT invoked. The event is unhandled or falls through to the default no-op path.

## Edge Cases

- A hook.binding with an invalid or unresolvable handler reference fails reconciliation with a clear error, not a silent skip
- Two hook.bindings for the same event both fire in deterministic order (registration order or priority field)
- A hook.binding with a scope that does not match the event context (e.g., binding scoped to workspace A, event from workspace B) does not fire
- Updating a hook.binding spec (e.g., changing the handler) and reconciling replaces the old binding, not duplicates it
- A handler that returns an error does not prevent other bindings for the same event from firing
- A hook.binding with an empty event field fails validation at persist time, not at dispatch time
- Reconciliation is idempotent: running it twice with no changes produces the same dispatch table
