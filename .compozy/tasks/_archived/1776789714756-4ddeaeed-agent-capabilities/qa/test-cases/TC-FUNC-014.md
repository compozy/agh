## TC-FUNC-014: Capability documentation and wire keys remain consistent with implementation

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-19
**Last Updated:** 2026-04-19
**Module:** `docs/rfcs/005_capability-catalogs-agent-directories.md`, `docs/rfcs/003_agh-network-v0.md`
**Traceability:** Task 05; RFC 003 capability discovery sections; TechSpec local layout and projection rules.
**Execution Surfaces:** Documentation review, exact key-string comparison, example validation.
**Durable Regression Anchors:** Runtime guide and RFC 003 text; package tests that assert `agh.capabilities_brief` survives payload conversion.

### Objective

Verify the user-visible runtime guide and RFC text still describe the shipped layouts, required fields, and wire keys exactly as implemented by tasks 01-04.

### Preconditions

- [ ] Current repository docs are available locally.
- [ ] The executor can compare docs against the runtime behavior and exact strings used in tests/code.

### Test Steps

1. Compare `docs/rfcs/005_capability-catalogs-agent-directories.md` against the TechSpec and tasks 01-04.
   - **Expected:** The guide lists all four supported local layouts, invalid mixed layouts, required fields, optional fields, basename rules, and no-catalog behavior.
2. Compare RFC 003 capability sections against the shipped wire behavior.
   - **Expected:** Keys and semantics exactly match `agh.capabilities_brief`, `agh.include`, `agh.capability_ids`, and `agh.capability_catalog`.
3. Cross-check implementation-visible payload strings.
   - **Expected:** Documentation does not claim the rich catalog lives in `peer_card.ext`, and brief/rich boundaries stay explicit.

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Runtime guide | `docs/rfcs/005_capability-catalogs-agent-directories.md` | Local authoring source |
| RFC | `docs/rfcs/003_agh-network-v0.md` | Wire contract source |
| Wire keys | `agh.capabilities_brief`, `agh.include`, `agh.capability_ids`, `agh.capability_catalog` | Exact-string match required |

### Post-conditions

- Evidence includes the exact strings reviewed and any mismatch notes.

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| No-catalog wording | Docs describe absent catalog | Matches deterministic empty/omitted runtime behavior |
| Projection boundary | Rich catalog location | Docs keep it in envelope `ext`, not `peer_card.ext` |

### Related Test Cases

- `TC-INT-005`
- `TC-INT-009`
- `TC-INT-010`
- `TC-INT-011`

### Execution History

| Date | Tester | Build | Result | Bug ID | Notes |
| --- | --- | --- | --- | --- | --- |
|  |  |  |  |  |  |

### Notes

- This case is P1, but it must still run before `task_07` closes because the feature is operator-authored and operator-discovered.
