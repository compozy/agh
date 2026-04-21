## TC-INT-001: Unified capability schema, digest, and no-catalog behavior

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

---

### Objective

Verify that AGH normalizes authored capability catalogs into one canonical runtime model, computes deterministic digests across equivalent TOML/JSON sources, rejects invalid authored shapes, and preserves the optional no-catalog behavior.

---

### Preconditions

- [ ] Repository is on the unified-capabilities branch with task_01 code present.
- [ ] Equivalent TOML and JSON capability fixtures are available or can be created in temp agent directories.
- [ ] The executor can run backend verification commands and targeted config/session tests.
- [ ] A temp agent directory without a capability catalog is available.

---

### Test Data

| Field | Value | Notes |
| --- | --- | --- |
| Catalog A | Equivalent TOML capability catalog | Includes `version`, `requirements`, and one list field such as `execution_outline` |
| Catalog B | Equivalent JSON capability catalog | Same semantic content as Catalog A |
| Invalid catalog | Duplicate ID / blank requirement / authored `digest` / mixed layout | Used to confirm hard validation |
| No-catalog agent | Agent directory with `AGENT.md` only | Used to confirm optional catalog behavior |

---

### Test Steps

1. Load or construct semantically equivalent TOML and JSON capability catalogs containing `id`, `summary`, `outcome`, `version`, and `requirements`.
   - **Expected:** The only differences are serialization format and insignificant ordering/whitespace.

2. Run the targeted config/session validation path for both catalogs.
   - **Expected:** Both inputs normalize to the same structured capability records and produce the same runtime `digest`.

3. Change one meaningful field, such as `execution_outline` or `requirements`, and rerun the same validation path.
   - **Expected:** The computed `digest` changes only when meaningful capability content changes.

4. Load an agent directory with no capability catalog.
   - **Expected:** The agent still loads successfully, capability discovery is empty, and transfer support remains protocol-level (`artifacts_supported` includes `"capability"`).

5. Attempt to load invalid authored inputs: duplicate IDs, blank or duplicate `requirements`, mixed layout/format, and authored `digest`.
   - **Expected:** Each invalid input fails hard with a descriptive validation error; AGH does not silently coerce or ignore the bad state.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Reordered requirements | Same IDs in different order | Same canonical `digest` |
| Trimmed whitespace | Padded strings and list entries | Same normalized model after trimming |
| Remote-only requirement | `requirements` references an ID not present locally | Accepted as a valid remote dependency reference |
| Empty capability inventory | No catalog at all | Deterministic empty discovery shape without load failure |

---

### Traceability

- Tasks: `task_01`
- TechSpec: `Data Models`, `Testing Approach`
- ADRs: `ADR-002`
- Primary surfaces: `internal/config/capabilities.go`, `internal/config/agent.go`, `internal/session/network_peer.go`

---

### Evidence to Capture

- Targeted command output or test output proving identical digests for equivalent TOML/JSON inputs
- Output proving digest change after a meaningful mutation
- Output proving no-catalog load success
- Validation output for each invalid authored case

---

### Notes

- This case is a P0 gate because later transfer, discovery, and docs behavior all assume the canonical capability model from task_01 is correct.
