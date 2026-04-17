# TC-FUNC-021: Agent codec rejects invalid specs

**Priority:** P1
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 09

## Objective

Validate that the agent resource codec performs strict validation on agent specs before they are persisted to the resource store. An agent resource with missing required fields, invalid field types, or structurally malformed specs must be rejected at the codec layer with a clear error message. This prevents invalid agent definitions from reaching downstream consumers (session manager, projectors).

## Preconditions

- Resource runtime is active with the agent kind registered.
- The agent codec is wired into the resource write path.
- The resource store is empty (no pre-existing agent records).

## Test Steps

1. Attempt to create an agent resource with a completely empty spec (no fields).
   **Expected:** Codec returns a validation error listing the missing required fields. No record is written to the store.

2. Attempt to create an agent resource with spec missing the `command` (or equivalent required execution field).
   **Expected:** Codec returns a validation error specifically identifying the missing command field. No record is written.

3. Attempt to create an agent resource with spec missing the `name` field.
   **Expected:** Codec returns a validation error identifying the missing name. No record is written.

4. Attempt to create an agent resource with an invalid field type (e.g., `max_turns` set to a string instead of integer).
   **Expected:** Codec returns a type mismatch validation error. No record is written.

5. Create an agent resource with all required fields present and valid.
   **Expected:** Codec validates successfully. Record is persisted in the store with correct version.

6. Retrieve the persisted record and verify its spec matches what was submitted.
   **Expected:** All fields round-trip correctly. No data loss or silent field dropping.

7. Attempt to update the valid agent resource with a spec that removes a required field.
   **Expected:** Codec rejects the update with a validation error. The existing record remains unchanged at its current version.

## Edge Cases

- Agent spec with unknown/extra fields: codec either ignores extra fields silently or rejects them (verify which behavior is specified by the codec contract).
- Agent spec with empty string for required string fields (e.g., name=""): codec rejects as invalid (empty is not the same as present).
- Agent spec with valid required fields but deeply nested invalid sub-structure (e.g., malformed env vars): codec validates nested structures, not just top-level fields.
- Concurrent writes of invalid and valid agent specs: invalid write fails without affecting the valid write.
- Agent spec with extremely long field values (e.g., 1MB description): codec enforces size limits or the store layer does — either way, a clear error is returned.
