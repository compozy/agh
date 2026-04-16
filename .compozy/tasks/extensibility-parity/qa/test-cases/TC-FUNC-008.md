# TC-FUNC-008: Typed projector adapter decodes once per pass

**Priority:** P1
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 02

## Objective

Validate that the `TypedProjector` adapter delivers pre-decoded `Record[T]` values to the projector callback, not raw `json.RawMessage` bytes. Each reconciliation pass should decode each record exactly once through the registered codec, and the projector function signature must accept the typed record directly. This eliminates redundant decoding in every projector and ensures type safety at compile time.

## Preconditions

- A fresh resource store is initialized with schema applied.
- A codec for the `tool` kind is registered mapping to `ToolSpec` struct.
- A `TypedProjector[ToolSpec]` is registered for the `tool` kind.
- The projector callback captures the received record types for assertion.
- Several `tool` records exist in the store.

## Test Steps

1. Register a `TypedProjector[ToolSpec]` that records the Go type of each record it receives into a test slice.
   **Expected:** The projector compiles without type errors. The callback signature accepts `[]Record[ToolSpec]`, not `[]Record[json.RawMessage]` or `[]RawRecord`.

2. Trigger a reconciliation pass for the `tool` kind (e.g., by committing a new tool record).
   **Expected:** The projector callback is invoked. Each element in the received slice has `.Spec` as a `ToolSpec` value (not `json.RawMessage`). Accessing `.Spec.Name` and `.Spec.Description` works directly without any additional JSON unmarshalling.

3. Count the number of codec decode invocations during the reconciliation pass using a decode counter on the codec.
   **Expected:** The decode count equals the number of tool records processed in the pass. Each record is decoded exactly once, not zero times (lazy) or multiple times (redundant).

4. Introduce a record with a payload that fails codec decoding. Trigger reconciliation.
   **Expected:** The reconciliation pass either skips the malformed record (logging an error) or fails the entire pass, depending on the error policy. The projector never receives a partially decoded or zero-value `ToolSpec`.

5. Register a second `TypedProjector[SkillSpec]` for the `skill` kind. Trigger reconciliation for both kinds.
   **Expected:** Each projector receives only records of its own kind with the correct typed spec. The `ToolSpec` projector never receives `SkillSpec` records and vice versa.

## Edge Cases

- A projector registered for a kind with no codec falls back to raw mode or is rejected at registration time.
- Zero records exist for a kind: the projector is either not called or called with an empty slice, but never called with nil.
- The adapter does not hold decoded records in memory longer than the projector callback's lifetime, preventing memory leaks on large record sets.
- If the codec is updated (e.g., new version of the struct), existing records that fail the new codec are handled gracefully during the next reconciliation pass.
