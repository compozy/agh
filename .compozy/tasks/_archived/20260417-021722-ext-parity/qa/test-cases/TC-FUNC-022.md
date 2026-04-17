# TC-FUNC-022: Skill publication preserves provenance and sidecar MCP

**Priority:** P1
**Type:** Functional
**Package:** internal/skills
**Related Tasks:** 09

## Objective

Validate that when a skill is published as a resource record, its provenance metadata (origin, author, source path, bundled vs user-defined) and any sidecar-derived MCP server attachments are preserved in the resource record without the resource runtime needing to parse skill definition files. The skill publisher must embed this information at write time so that downstream consumers can access provenance and MCP configuration purely from the resource record.

## Preconditions

- Resource runtime is active with the skill kind registered.
- A bundled skill definition exists with known provenance metadata (e.g., origin="bundled", source path points to internal/skills/bundled/).
- A user-defined skill exists with a sidecar MCP server configuration (e.g., the skill references an MCP server in its definition).
- The skill publisher/loader is initialized.

## Test Steps

1. Publish the bundled skill through the skill resource pipeline.
   **Expected:** Skill resource record is created in the store. No errors.

2. Retrieve the bundled skill resource record from the store.
   **Expected:** The record's metadata contains provenance fields: origin is "bundled", source path is present and correct, author/package information is preserved.

3. Verify provenance is accessible without re-reading the skill definition file.
   **Expected:** All provenance data is embedded in the resource record's metadata. No file I/O is required to determine the skill's origin.

4. Publish the user-defined skill that has a sidecar MCP server attachment.
   **Expected:** Skill resource record is created in the store. No errors.

5. Retrieve the user-defined skill resource record from the store.
   **Expected:** The record's metadata contains provenance fields: origin is "user" (or equivalent), source path points to the user's skill directory.

6. Inspect the sidecar MCP attachment in the user-defined skill's resource record.
   **Expected:** The MCP server configuration (server name, command, args, env) is embedded in the resource record's spec or metadata. The resource runtime did not parse the MCP config — it was provided by the skill publisher at write time.

7. Update the user-defined skill's definition to change the MCP server args. Re-publish.
   **Expected:** The resource record is updated (version incremented). The MCP attachment reflects the new args. Provenance origin remains "user".

## Edge Cases

- Skill with no MCP sidecar: resource record has MCP attachment field as null/empty, not absent. No error during publication.
- Skill provenance with special characters in source path (spaces, unicode): round-trips correctly.
- Republishing a bundled skill with identical content: version does not increment if the resource store supports idempotent writes, or increments if all writes bump version — verify which behavior is specified.
- Skill with multiple MCP sidecar servers: all are preserved in the resource record, not just the first.
- Deleting and re-creating a skill resource: provenance is fresh from the new publication, not carried over from the deleted record.
