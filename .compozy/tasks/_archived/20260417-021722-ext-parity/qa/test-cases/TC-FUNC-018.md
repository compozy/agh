# TC-FUNC-018: tool.* and permission.* hooks wired end to end

**Priority:** P0
**Type:** Functional
**Package:** internal/hooks
**Related Tasks:** 07

## Objective

Validate that resource-backed hook bindings for tool.pre_call and permission.request events fire correctly when the corresponding domain events occur. This proves the end-to-end wiring from hook.binding resource records through the projector, into the live dispatch table, and finally into the runtime hook invocation path. Both hook families (tool.* and permission.*) must be exercised.

## Preconditions

- Daemon is running with the resource runtime and hook projector active.
- A test tool resource (e.g., `my-tool`) is registered and callable.
- No pre-existing hook.binding records exist for `tool.pre_call` or `permission.request` (clean state).
- A hook handler implementation is available that records invocations (e.g., appends to an in-memory log or writes a marker to the store).

## Test Steps

1. Register a hook.binding resource for event `tool.pre_call` targeting tool `my-tool`, with the handler set to record invocation metadata (tool name, arguments, timestamp).
   **Expected:** hook.binding resource created successfully. Projector Build + Apply cycle completes.

2. Register a hook.binding resource for event `permission.request` with the handler set to record invocation metadata (permission kind, requester, timestamp).
   **Expected:** hook.binding resource created successfully. Projector Build + Apply cycle completes.

3. Invoke tool `my-tool` with sample arguments via an agent session or direct API call.
   **Expected:** The `tool.pre_call` hook fires before the tool executes. The handler's invocation log contains an entry with the correct tool name and arguments.

4. Verify the tool call itself completed successfully after the hook.
   **Expected:** Tool execution returns a valid result. The hook did not block or abort the tool call (assuming the hook handler does not reject).

5. Trigger a permission.request event (e.g., an agent requests file write permission).
   **Expected:** The `permission.request` hook fires. The handler's invocation log contains an entry with the correct permission kind and requester identity.

6. Remove the hook.binding resource for `tool.pre_call`. Run projector Build + Apply.
   **Expected:** The binding is removed from the live dispatch table.

7. Invoke tool `my-tool` again.
   **Expected:** No `tool.pre_call` hook fires. The invocation log has no new entries for this event. The tool still executes normally.

8. Verify the `permission.request` binding is still active by triggering another permission event.
   **Expected:** The `permission.request` hook still fires. Removing one binding did not affect the other.

## Edge Cases

- Hook handler that returns an error: the runtime must propagate or log the error without crashing. Tool call behavior depends on hook contract (fail-open vs fail-closed — verify which is specified).
- Multiple hook.binding records for the same event: all handlers fire (fan-out), not just the first.
- Hook binding referencing a non-existent tool: binding is created but the hook never fires (no phantom invocations).
- Hook handler with high latency: tool call is delayed but not deadlocked (timeout applies if configured).
- Re-creating a deleted hook.binding with the same name: new binding takes effect after Build + Apply.
