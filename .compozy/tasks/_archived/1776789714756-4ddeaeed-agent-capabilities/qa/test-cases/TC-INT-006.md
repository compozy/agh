## TC-INT-006: Loader rejects directory entries whose basename does not match capability ID

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; TechSpec validation rule for basename-without-extension matching `id`.
**Execution Surfaces:** Runtime loader validation.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesRejectsDirectoryBasenameMismatch`

### Objective

Verify directory-mode capability files fail hard when the filename basename does not match the normalized `id` value.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] `capabilities/` contains a file whose basename does not match the declared `id`.

### Test Steps

1. Create a directory-mode capability file with a deliberate basename mismatch.
   - Input: filename `build-site.toml` with `id = "review-copy"`.
   - **Expected:** The file is syntactically valid but semantically invalid.
2. Load the agent through runtime discovery.
   - **Expected:** The loader returns a hard validation error.
3. Inspect the validation error.
   - **Expected:** The error names both the file path and the mismatched capability ID expectation.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Filename | `build-site.toml` | Basename used for validation |
| Declared ID | `review-copy` | Deliberate mismatch |

### Post-conditions

- Invalid fixtures can be removed.
- Evidence includes the exact validation message.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| JSON directory mode | `build-site.json` with mismatched `id` | Same hard validation behavior |
| Normalized mismatch | Whitespace around `id` | Validation uses the normalized value |

### Related Test Cases

- `TC-INT-003`
- `TC-INT-004`
- `TC-INT-007`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This remains P1 because it is a narrow authoring error, but it must still block release if it regresses.
