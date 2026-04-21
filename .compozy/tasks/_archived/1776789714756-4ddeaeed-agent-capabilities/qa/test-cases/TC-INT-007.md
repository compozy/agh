## TC-INT-007: Loader rejects duplicate capability IDs after normalization

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; TechSpec uniqueness rule; ADR-003 normalization semantics.
**Execution Surfaces:** Runtime loader validation.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesRejectsDuplicateNormalizedIDsAcrossDirectoryEntries`

### Objective

Verify AGH rejects duplicate capability IDs after normalization so overlapping declarations cannot silently poison discovery results.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] Two capability entries normalize to the same ID.

### Test Steps

1. Create a directory-mode catalog with two capability files whose IDs normalize to the same value.
   - Input: one file declaring `build-site`, another declaring ` build-site `.
   - **Expected:** The files look different on disk but normalize to the same capability ID.
2. Load the agent through runtime discovery.
   - **Expected:** The loader fails hard instead of choosing one entry.
3. Inspect the validation error.
   - **Expected:** The error identifies the duplicate normalized ID.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| File A ID | `build-site` | Baseline ID |
| File B ID | ` build-site ` | Normalizes to same value |

### Post-conditions

- Invalid fixtures can be removed.
- Evidence includes the duplicate-ID validation message.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Single-file catalog | Two entries in one catalog normalize to same ID | Same hard validation behavior |
| Cross-format probe | Duplicate IDs across invalid mixed formats | Mixed-format rejection takes precedence, see `TC-INT-005` |

### Related Test Cases

- `TC-INT-005`
- `TC-INT-006`
- `TC-INT-011`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This case is P0 because duplicate IDs corrupt the source-of-truth catalog even before network projection begins.
