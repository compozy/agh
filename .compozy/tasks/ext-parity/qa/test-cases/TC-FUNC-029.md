# TC-FUNC-029: Mixed-kind adapter does not expose raw JSON

**Priority:** P1
**Type:** Functional
**Package:** internal/bundles
**Related Tasks:** 12

## Objective

Validate that the bundle.activation projector decodes dependency kinds explicitly through typed codecs without leaking json.RawMessage (or equivalent untyped byte slices) to domain code. When a bundle activation fans out to multiple resource kinds (e.g., tool, skill, automation.job), each kind's spec must be decoded into its concrete Go type before reaching domain consumers. This prevents type confusion, ensures compile-time safety, and avoids domain code performing ad-hoc JSON parsing.

## Preconditions

- Resource runtime is active with typed codecs registered for tool, skill, and automation.job kinds.
- A bundle "full-stack-bundle" exists with allowlist: tool, skill, automation.job.
- A bundle.activation is prepared that fans out to create one record of each kind, with their specs provided as part of the activation payload.

## Test Steps

1. Create a bundle.activation for "full-stack-bundle" with fan-out specs for: tool "build-tool" (with input_schema), skill "deploy-skill" (with provenance), automation.job "cleanup-job" (with cron schedule).
   **Expected:** Activation succeeds. Three owned records are created.

2. Retrieve the created tool record and inspect its spec type at the Go level (via test assertion or reflection in test code).
   **Expected:** The spec is a concrete ToolSpec struct (or equivalent typed struct), NOT a json.RawMessage, map[string]interface{}, or []byte. All fields (description, input_schema) are accessible as typed fields.

3. Retrieve the created skill record and inspect its spec type.
   **Expected:** The spec is a concrete SkillSpec struct with typed fields (provenance, mcp_config, etc.), NOT json.RawMessage.

4. Retrieve the created automation.job record and inspect its spec type.
   **Expected:** The spec is a concrete AutomationJobSpec struct with typed fields (schedule, command, etc.), NOT json.RawMessage.

5. Attempt to create a bundle.activation with a malformed tool spec (e.g., input_schema is a string instead of an object).
   **Expected:** The typed codec rejects the spec during activation fan-out. Error message references the specific field and type mismatch, not a generic JSON parse error.

6. Verify that the projector's Build method returns typed plan entries, not raw JSON.
   **Expected:** Build result contains plan entries with typed specs. Code consuming Build output can access fields without type assertions or JSON unmarshaling.

## Edge Cases

- Bundle activation with a kind whose codec is not registered: activation fails with a clear "unknown kind" error, not a panic or nil pointer from missing codec.
- Spec with optional fields omitted: typed codec produces the struct with zero-value fields, not json.RawMessage for the missing fields.
- Spec with extra fields not in the typed struct: codec either ignores them (strict decoding off) or rejects them (strict decoding on) — verify which is specified. Either way, no json.RawMessage leaks.
- Very large spec (e.g., 100KB input_schema): decoded into typed struct without intermediate json.RawMessage allocation visible to domain code.
- Roundtrip: create activation -> retrieve records -> re-serialize to JSON -> deserialize again: produces identical typed structs. No lossy encoding from json.RawMessage shortcuts.
