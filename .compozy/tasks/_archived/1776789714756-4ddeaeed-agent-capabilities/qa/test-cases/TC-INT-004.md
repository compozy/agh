## TC-INT-004: Loader accepts directory-mode JSON catalogs

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; TechSpec directory-mode JSON rules; ADR-002 directory mode without merge.
**Execution Surfaces:** Runtime loader entrypoints, workspace discovery.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesDirectoryModeLoadsSelectedRegularFilesOnly`, `TestLoadWorkspaceAgentDefsPreservesPrecedenceWithCapabilities`

### Objective

Verify directory-mode JSON capability catalogs load correctly, preserve required/optional fields, and remain isolated from unsupported entries.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] `capabilities/` contains one or more `.json` capability files.
- [ ] The executor can inspect the normalized catalog output.

### Test Steps

1. Create a temporary agent directory with `capabilities/` containing valid JSON capability files.
   - Input: one file per capability with required fields.
   - **Expected:** The layout matches supported directory-mode JSON authoring.
2. Load the agent through runtime discovery.
   - **Expected:** The agent loads successfully with a normalized non-nil catalog.
3. Inspect the resulting catalog.
   - **Expected:** Required fields are preserved, optional fields remain intact, and the catalog excludes non-JSON entries.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Layout | `capabilities/*.json` | Directory JSON mode |
| Capability IDs | `build-site`, `review-copy` | Baseline entries |
| Optional field | `artifacts_expected` | Confirms richer fields survive |

### Post-conditions

- Temporary fixtures can be removed.
- Evidence includes the loaded normalized catalog.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Ignored text file | `capabilities/notes.txt` | Entry is ignored |
| Mixed-format probe | Add `.toml` alongside `.json` | Covered by `TC-INT-005`, should fail hard |

### Related Test Cases

- `TC-INT-003`
- `TC-INT-005`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- Keep JSON happy-path evidence separate from strict JSON rejection evidence in `TC-INT-002`.
