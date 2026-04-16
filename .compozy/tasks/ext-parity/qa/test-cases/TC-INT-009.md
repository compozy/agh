# TC-INT-009: Resource-backed hook binding fires on tool.pre_call

**Priority:** P0
**Type:** Integration
**Package:** internal/hooks, internal/session
**Related Tasks:** 07

## Objective

Validate that a hook binding persisted as a resource record fires correctly when the `tool.pre_call` event occurs during session execution. This proves the migration from legacy hook storage to resource-backed hook bindings is functional end-to-end.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with hook projector wired
- Session runtime initialized with hook dispatch capability
- A test hook handler that records invocations (e.g., writes to a channel or atomic counter)

## Test Steps

1. Persist a `hook.binding` resource record targeting `event=tool.pre_call` with the test hook handler configuration.
   **Expected:** Record stored in `resource_records` with correct kind, id, and data payload.

2. Trigger reconciliation so the hook projector rebuilds the dispatch table from resource records.
   **Expected:** Projector Apply completes. Dispatch table now contains the `tool.pre_call` binding.

3. Simulate a tool call through the session runtime that emits the `tool.pre_call` event.
   **Expected:** The hook dispatch table matches the event and routes it to the test hook handler.

4. Verify the test hook handler was invoked.
   **Expected:** Handler's invocation counter is 1. The hook received the correct event payload (tool name, arguments, etc.).

5. Simulate a second tool call.
   **Expected:** Handler's invocation counter is 2. Each tool call fires the hook exactly once.

6. Remove the `hook.binding` resource record and trigger reconciliation.
   **Expected:** Dispatch table no longer contains the binding. Subsequent tool calls do not fire the hook.

7. Simulate a third tool call.
   **Expected:** Handler's invocation counter remains 2. Hook did not fire.

## Edge Cases

- Multiple hook bindings for the same event — all fire, none suppressed
- Hook handler returns an error — error is logged/propagated, does not crash the session
- Hook binding with a filter (e.g., only for specific tool names) — fires only for matching tools
- Hook binding added while a tool call is in progress — does not retroactively fire for in-flight call
- Resource record with malformed hook data — projector skips it, logs warning, other bindings unaffected
