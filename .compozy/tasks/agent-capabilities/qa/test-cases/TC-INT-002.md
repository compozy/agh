## TC-INT-002: Loader accepts single-file JSON capability catalog with strict decoding

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `internal/config`
**Traceability:** Task 01; TechSpec "Testing Approach" single-file JSON loader coverage; shared-memory note about strict JSON parsing.
**Execution Surfaces:** Runtime loader entrypoints, workspace discovery.
**Durable Regression Anchors:** `TestLoadAgentCapabilitiesFromSingleFileJSONStrictness`, `TestLoadAgentDefFileLoadsCapabilityCatalogAndMCPSidecar`

### Objective

Verify a valid `capabilities.json` loads successfully and that JSON decoding stays strict about unknown fields and trailing data.

### Preconditions

- [ ] Temporary agent directory exists with a valid `AGENT.md`.
- [ ] A valid `capabilities.json` exists with at least one capability.
- [ ] The executor can run a strict-decoding negative variation if the happy path succeeds.

### Test Steps

1. Create a temporary agent directory with `AGENT.md` and a valid `capabilities.json`.
   - Input: one or more capability objects with required fields.
   - **Expected:** The JSON catalog matches the documented top-level `capabilities` array shape.
2. Load the agent through the runtime entrypoint.
   - **Expected:** The agent loads successfully and attaches a normalized non-nil catalog.
3. Re-run the load with a negative variation.
   - Input: either an unknown field or trailing JSON data.
   - **Expected:** The loader rejects the invalid JSON with a hard validation error instead of ignoring the problem.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Layout | `capabilities.json` | Single-file JSON mode |
| Capability ID | `review-copy` | Simple baseline |
| Negative data | unknown field or trailing payload | Strict decode proof |

### Post-conditions

- Temporary fixtures can be removed.
- Evidence includes one successful load and one strict-decoder rejection.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Unknown field | Add a non-schema field | Loader fails hard |
| Trailing JSON | Append a second JSON object | Loader fails hard |

### Related Test Cases

- `TC-INT-001`
- `TC-INT-005`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This case is P1 because TOML is the smoke baseline, but JSON support remains release-relevant.
