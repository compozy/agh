## TC-INT-005: Loader rejects mixed layouts and mixed formats with hard validation errors

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; ADR-002; TechSpec validation rules for one storage mode and one format.
**Execution Surfaces:** Runtime loader validation, agent discovery.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesRejectsMixedFileAndDirectoryModes`, `TestLoadAgentCapabilitiesRejectsMultipleSingleFiles`, `TestLoadAgentCapabilitiesRejectsMixedDirectoryFormats`

### Objective

Verify AGH never merges unsupported capability layouts or formats and instead returns hard validation errors that name the conflicting files.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] The executor can create multiple invalid catalog layouts and inspect error output.

### Test Steps

1. Create an invalid agent directory with both `capabilities.toml` and `capabilities/`.
   - **Expected:** The layout is rejected with a mixed-layout validation error.
2. Create an invalid agent directory with both `capabilities.toml` and `capabilities.json`.
   - **Expected:** The layout is rejected with a multiple-single-file validation error.
3. Create an invalid `capabilities/` directory containing both `.toml` and `.json` files.
   - **Expected:** The layout is rejected with a mixed-format validation error.
4. Inspect the error details for each rejection.
   - **Expected:** Errors identify the conflicting files or directories and AGH performs no merge behavior.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Invalid layout A | `capabilities.toml` + `capabilities/` | Mixed mode |
| Invalid layout B | `capabilities.toml` + `capabilities.json` | Multiple single-file modes |
| Invalid layout C | `capabilities/*.toml` + `capabilities/*.json` | Mixed directory formats |

### Post-conditions

- Invalid fixtures can be removed.
- Evidence includes the exact validation message for each rejection path.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Conflicting file names | Multiple valid-looking files | All conflicting paths are named in the error |
| Workspace scan | Invalid agent discovered through workspace search | Workspace load fails for the offending agent rather than silently merging |

### Related Test Cases

- `TC-INT-002`
- `TC-INT-003`
- `TC-INT-004`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This case stays in smoke because merge behavior would undermine every downstream discovery claim.
