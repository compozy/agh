# SMOKE-002: Typed Codec Encode/Decode Round-Trip

**Priority:** P0
**Type:** Smoke
**Package:** internal/resources
**Related Tasks:** 02

## Objective

Validate that a registered KindCodec correctly encodes a typed Go struct into a raw JSON spec and decodes it back without data loss. Verify that version, scope, and owner metadata survive the round-trip through the typed store layer, ensuring the codec abstraction does not silently drop or corrupt fields.

## Preconditions

- A KindCodec registered for kind="test.codec" that encodes/decodes a test struct (e.g., `TestWidget{Name string; Count int; Tags []string}`)
- A typed store instance wrapping the raw persistence kernel
- SQLite database initialized via `t.TempDir()`

## Test Steps

1. **Create a typed spec struct** with Name="alpha", Count=42, Tags=["go","test"].
   **Expected:** Struct is valid and ready for encoding.

2. **Encode the struct via the KindCodec** into a raw JSON spec ([]byte or json.RawMessage).
   **Expected:** Returns valid JSON containing all fields. `json.Valid()` returns true. Deserializing the JSON independently confirms Name, Count, and Tags are present with correct values.

3. **Put the encoded spec through the typed store** with kind="test.codec", scope="global", owner_kind="system", owner_id="bootstrap", expected_version=0.
   **Expected:** Returns a typed record with version=1, correct scope/owner metadata, and the encoded spec persisted.

4. **Get the record back through the typed store** by ID and kind.
   **Expected:** Returns the record with version=1 and all metadata fields intact.

5. **Decode the raw spec from the returned record** back into the typed struct via the KindCodec.
   **Expected:** The decoded struct is deeply equal to the original: Name="alpha", Count=42, Tags=["go","test"]. No fields are zero-valued or missing.

6. **Update the struct** (Count=99, append "updated" to Tags), encode, and PutTyped with expected_version=1.
   **Expected:** Returns version=2. Decode the returned spec and verify Count=99, Tags=["go","test","updated"].

## Edge Cases

- Encoding a struct with zero-value fields (empty string, 0, nil slice) round-trips correctly without omitting them
- Decoding a spec with extra unknown JSON fields does not error (forward compatibility)
- Decoding a spec with a missing optional field sets the Go struct field to its zero value
- Registering a duplicate KindCodec for the same kind returns an error or panics at registration time
- Encoding a nil struct pointer returns an error, not a nil JSON blob
- A codec whose Encode returns invalid JSON is caught at Put time, not silently persisted
