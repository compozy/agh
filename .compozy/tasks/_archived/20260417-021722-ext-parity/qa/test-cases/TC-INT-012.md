# TC-INT-012: Static manifest publishes tools into canonical store

**Priority:** P0
**Type:** Integration
**Package:** internal/extension, internal/resources
**Related Tasks:** 08

## Objective

Validate that loading an extension manifest with static tool declarations results in tool records being published into the resource store with correct shapes. This is the primary path for extensions to declare tools without runtime interaction — the manifest is the source of truth.

## Preconditions

- Real SQLite database via `t.TempDir()` with resource tables created
- Resource store initialized with reconcile driver for `kind=tool`
- A test extension manifest (TOML or embedded struct) declaring 3 tools with names, descriptions, and input schemas
- Extension loader/manager capable of parsing manifests and publishing records

## Test Steps

1. Load the test extension manifest containing 3 tool declarations: `tool-alpha`, `tool-beta`, `tool-gamma`.
   **Expected:** Manifest parsed without error. All 3 tool declarations extracted.

2. Trigger the manifest-to-resource publish flow.
   **Expected:** 3 resource records created with `kind=tool` and IDs matching the tool names.

3. Query the resource store for `kind=tool` records from the extension's source.
   **Expected:** Exactly 3 records returned: `tool-alpha`, `tool-beta`, `tool-gamma`.

4. Inspect each record's `data` payload.
   **Expected:** Each record contains the tool's name, description, and input schema as declared in the manifest. No field is missing or truncated.

5. Verify the records have the correct `source` and `owner_kind`/`owner_id` fields.
   **Expected:** `source` matches the extension identifier. Owner fields reflect the extension as the owning entity.

6. Reload the same manifest (idempotent re-publish).
   **Expected:** No duplicate records. `updated_at` may change but record count remains 3.

## Edge Cases

- Manifest with zero tool declarations — no records published, no error
- Manifest with a tool that has no input schema — record created with empty/null schema field
- Manifest with duplicate tool names — last declaration wins or error, no silent duplicates
- Tool name with special characters — stored and retrievable without escaping issues
- Manifest loaded after the extension already published dynamic tools — manifest tools coexist with dynamic tools (different IDs)
