# TC-FUNC-019: Manifest tool declarations normalize to canonical record shape

**Priority:** P0
**Type:** Functional
**Package:** internal/tools
**Related Tasks:** 08

## Objective

Validate that tool declarations from two distinct origins — static manifest declarations (defined in agent TOML config) and dynamic snapshot tools (discovered at runtime from extensions or MCP servers) — are both normalized into identical canonical resource record shapes. This ensures that downstream consumers (hook dispatch, permission checks, UDS API responses) can treat all tools uniformly regardless of provenance.

## Preconditions

- A static tool is declared in the agent manifest (TOML config) with fields: name, description, input_schema, and any manifest-specific metadata.
- An extension or MCP server is configured to advertise a dynamic tool with the same logical fields: name, description, input_schema.
- The resource store is accessible for inspection.
- The tool resource codec/normalizer is active.

## Test Steps

1. Load the agent configuration that declares a static manifest tool `calc-sum` with name, description, and input_schema.
   **Expected:** Configuration loads without error. The manifest tool is recognized.

2. Register the manifest tool through the tool resource pipeline.
   **Expected:** A tool resource record is created in the store for `calc-sum`. No validation errors.

3. Start an extension/MCP server that advertises a dynamic tool `calc-product` with equivalent field structure (name, description, input_schema).
   **Expected:** The dynamic tool is discovered and ingested by the tool snapshot pipeline.

4. Register the dynamic tool through the tool resource pipeline.
   **Expected:** A tool resource record is created in the store for `calc-product`. No validation errors.

5. Retrieve both resource records from the store and compare their shapes.
   **Expected:** Both records have identical top-level structure: same field names, same nesting, same type representations. Fields present: kind, name, version, spec (containing description, input_schema), metadata (containing provenance). The only differences are in the values (different names, descriptions, schemas) and provenance metadata (manifest vs extension).

6. Verify the spec sub-structure is identical in shape for both records.
   **Expected:** spec.description is a string in both. spec.input_schema is a JSON Schema object in both. No manifest-only or extension-only fields leak into the canonical spec.

7. Pass both records through the tool projector Build.
   **Expected:** Both are accepted without kind-specific branching. The projector treats them uniformly.

## Edge Cases

- Manifest tool with extra non-standard fields: extra fields are either dropped during normalization or placed in a metadata sidecar — they do not appear in spec.
- Dynamic tool with missing optional fields (e.g., no description): the canonical record has the field present with a zero value (empty string), not absent.
- Tool name collision between manifest and dynamic: the resource store enforces uniqueness — second write either fails with conflict or overwrites with higher version.
- input_schema with complex nested types (objects, arrays, enums): normalized identically from both origins without schema rewriting.
- Manifest tool with no input_schema: canonical record has input_schema as empty object or null — matches the same representation a dynamic tool with no input_schema would produce.
