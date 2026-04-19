## TC-INT-003: Loader accepts directory-mode TOML catalogs and ignores unsupported entries

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; TechSpec directory-mode rules; ADR-002 directory mode without merge.
**Execution Surfaces:** Runtime loader entrypoints, workspace discovery.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesDirectoryModeLoadsSelectedRegularFilesOnly`, `TestLoadWorkspaceAgentDefsPreservesPrecedenceWithCapabilities`

### Objective

Verify directory-mode TOML catalogs load every valid regular file, ignore dotfiles and unrelated entries, and preserve normalized capability order.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] `capabilities/` contains at least two `.toml` files plus one ignored entry.
- [ ] The executor can inspect the final normalized catalog.

### Test Steps

1. Create a temporary agent directory with `capabilities/` containing valid `.toml` capability files.
   - Input: `review-pr.toml`, `draft-spec.toml`, plus `.hidden.toml` or `notes.txt`.
   - **Expected:** The directory matches supported TOML directory mode.
2. Load the agent through runtime discovery.
   - **Expected:** Only valid regular TOML files are loaded into the catalog.
3. Inspect the resulting catalog and ignored files.
   - **Expected:** Hidden files, nested directories, and unrelated extensions do not appear in the catalog; valid entries remain in normalized order.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Layout | `capabilities/*.toml` | Directory mode |
| Valid IDs | `review-pr`, `draft-spec` | Loaded into final catalog |
| Ignored entries | `.hidden.toml`, `notes.txt` | Must not be loaded |

### Post-conditions

- Temporary fixtures can be removed.
- Evidence includes loaded IDs and proof that ignored files stayed excluded.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Nested directory | `capabilities/nested/ignored.toml` | Nested file is ignored |
| Workspace precedence | Same agent exists in multiple roots | Winning definition retains directory-mode catalog |

### Related Test Cases

- `TC-INT-004`
- `TC-INT-005`
- `TC-INT-006`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- Capture a short list of loaded IDs and ignored-file names in the execution evidence.
