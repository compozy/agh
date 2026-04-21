## TC-INT-001: Loader accepts single-file TOML capability catalog

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; TechSpec "Testing Approach" single-file TOML loader coverage; ADR-002 single-file mode.
**Execution Surfaces:** Runtime loader entrypoints, workspace discovery.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesFromSingleFileTOMLNormalizesEntries`, `TestLoadAgentDefFileLoadsCapabilityCatalogAndMCPSidecar`

### Objective

Verify a valid `capabilities.toml` loads through the same runtime path used by AGH agent discovery, preserves optional fields, and yields a normalized non-nil catalog.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] A valid `capabilities.toml` contains at least two capabilities.
- [ ] The executor can inspect the loaded `AgentDef.Capabilities` value through runtime output or focused regression anchors.

### Test Steps

1. Create a temporary agent directory with `AGENT.md` and `capabilities.toml`.
   - Input: capabilities such as `review-pr` and `draft-spec`, plus at least one optional list field.
   - **Expected:** The catalog matches the documented single-file TOML shape.
2. Load the agent through the same runtime entrypoint used by AGH.
   - Input: direct file load and one workspace-discovery variant.
   - **Expected:** Both paths succeed and attach a non-nil capability catalog.
3. Inspect the resulting catalog entries.
   - **Expected:** Catalog order matches file order, required fields are present, optional arrays survive normalization, and no validation error is emitted.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Layout | `capabilities.toml` | Primary single-file authoring mode |
| Capability IDs | `review-pr`, `draft-spec` | Stable-order check |
| Optional field | `context_needed` | Confirms richer fields persist |

### Post-conditions

- Temporary fixtures can be removed.
- Evidence captures both the successful load and the normalized catalog shape.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Whitespace normalization | Surround `id` or list entries with spaces | Stored values are normalized before validation |
| Workspace precedence | Same agent ID in multiple search roots | Only the winning agent definition retains the catalog |

### Related Test Cases

- `TC-INT-002`
- `TC-INT-003`
- `TC-INT-011`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- Capture one direct-load artifact and one workspace-discovery artifact during `task_07`.
