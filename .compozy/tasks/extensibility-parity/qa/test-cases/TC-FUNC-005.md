# TC-FUNC-005: Owner/source fields stamped from actor

**Priority:** P1
**Type:** Functional
**Package:** internal/resources
**Related Tasks:** 01

## Objective

Validate that the resource store stamps `owner_kind`, `owner_id`, `source_kind`, and `source_id` from the `MutationActor` context, not from user-supplied payload fields. Even if the caller includes these fields in the raw payload or record metadata, the store must override them with the values from the authenticated actor. This prevents privilege escalation through field spoofing.

## Preconditions

- A fresh resource store is initialized with schema applied.
- A `MutationActor` is configured with `OwnerKind="user"`, `OwnerID="user-42"`, `SourceKind="cli"`, `SourceID="cli-session-7"`.
- At least one resource kind (e.g., `tool`) is registered.

## Test Steps

1. Call `PutRaw` with `Kind="tool"`, `ID="stamped-tool"`, `ExpectedVersion=0`, a valid payload, and explicitly set `OwnerKind="admin"`, `OwnerID="admin-1"`, `SourceKind="daemon"`, `SourceID="daemon-main"` in the request metadata.
   **Expected:** The call succeeds. The returned record has `OwnerKind="user"`, `OwnerID="user-42"`, `SourceKind="cli"`, `SourceID="cli-session-7"` -- all from the actor, not from the user-supplied values.

2. Call `Get` for `Kind="tool"`, `ID="stamped-tool"`.
   **Expected:** The persisted record confirms `OwnerKind="user"`, `OwnerID="user-42"`, `SourceKind="cli"`, `SourceID="cli-session-7"`.

3. Switch to a different `MutationActor` with `OwnerKind="agent"`, `OwnerID="agent-99"`, `SourceKind="extension"`, `SourceID="ext-abc"`. Call `PutRaw` with `Kind="tool"`, `ID="agent-tool"`, `ExpectedVersion=0`.
   **Expected:** The returned record has `OwnerKind="agent"`, `OwnerID="agent-99"`, `SourceKind="extension"`, `SourceID="ext-abc"`.

4. Using the original actor (`user-42`), attempt to update the record created by `agent-99` (if cross-source updates are allowed by policy).
   **Expected:** If allowed, the `owner_kind`/`owner_id` on the record reflect the original creator or the policy-defined behavior. The `source_kind`/`source_id` are updated to the current actor's source, not the original source.

## Edge Cases

- `MutationActor` with empty `OwnerID` or `SourceID`: the store rejects the operation at validation, not allowing blank ownership stamps.
- Payload JSON that contains top-level fields named `owner_kind`, `source_id`, etc.: these fields in the payload body are preserved as payload data but do not influence the record-level metadata fields.
- `ApplySourceSnapshotRaw` also stamps all records in the batch from the actor, not from individual record payloads within the snapshot.
- Actor fields containing special characters or very long strings are validated and bounded before stamping.
