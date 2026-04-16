# TC-FUNC-017: Hook projector Build does not mutate live dispatch table

**Priority:** P0
**Type:** Functional
**Package:** internal/hooks
**Related Tasks:** 07

## Objective

Validate that the hook projector follows the Build/Apply two-phase commit pattern. Calling Build must compute a new dispatch table from hook.binding resource records but must not replace or mutate the live dispatch table that is actively serving hook invocations. The live table must remain unchanged until Apply is explicitly called. This guarantees zero-downtime reconfiguration and prevents partial states from leaking into production dispatch.

## Preconditions

- Hook projector is instantiated with an initial (possibly empty) dispatch table.
- A set of hook.binding resource records is prepared in the resource store (e.g., bindings for `tool.pre_call`, `session.on_start`).
- The current live dispatch table has a known, deterministic state (e.g., contains one existing binding for `session.on_start`).

## Test Steps

1. Snapshot the current live dispatch table by querying the projector's active bindings.
   **Expected:** Returns the known initial state (e.g., one binding for `session.on_start`).

2. Insert two new hook.binding resource records into the store: one for `tool.pre_call` and one for `permission.request`.
   **Expected:** Records are persisted in the resource store.

3. Call the projector's Build method.
   **Expected:** Build returns a plan/delta object (or success status) without error. The method does not block indefinitely.

4. Immediately query the live dispatch table again.
   **Expected:** The live dispatch table is identical to the snapshot from step 1. The two new bindings are NOT present in live dispatch. No in-flight hook invocations are affected.

5. Trigger a hook event that would match `tool.pre_call` (e.g., simulate a tool call).
   **Expected:** No handler fires for `tool.pre_call` because the live table has not been updated.

6. Call the projector's Apply method with the result from Build.
   **Expected:** Apply completes without error. The live dispatch table is atomically swapped.

7. Query the live dispatch table after Apply.
   **Expected:** The dispatch table now contains all three bindings: the original `session.on_start` plus the two new ones (`tool.pre_call`, `permission.request`).

8. Trigger a hook event matching `tool.pre_call`.
   **Expected:** The newly registered handler fires as expected.

## Edge Cases

- Build called twice without Apply: second Build supersedes the first without corrupting state.
- Apply called without a preceding Build: returns an error or no-op (does not panic).
- Build with an empty resource set: produces a dispatch table with zero bindings. Apply removes all live bindings.
- Concurrent Build calls: only one should succeed or they should be serialized; no data races.
- Build detects invalid hook.binding spec (e.g., unknown event type): returns a build error and does not produce a partial plan.
