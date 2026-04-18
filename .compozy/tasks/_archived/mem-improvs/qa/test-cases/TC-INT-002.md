## TC-INT-002: UDS and CLI Remain in Parity with the HTTP Memory Contract

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** UDS + CLI
**Requirement:** REQ-MEM-003, REQ-MEM-004

---

### Objective

Verify that CLI and UDS-backed flows return the same logical results as the HTTP API for search and reindex.

---

### Preconditions

- [ ] The same daemon instance is reachable via HTTP and UDS.
- [ ] The `agh` CLI is configured to talk to that daemon.
- [ ] The seeded corpus contains mixed global/workspace memories.

---

### Test Steps

1. Run the HTTP search request from `TC-INT-001` and record the top hit and result count.
   - **Expected:** Baseline HTTP values are captured.

2. Run `agh memory search "auth sessions"`.
   - **Expected:** The CLI succeeds and returns the same top hit, scope, and snippet semantics as HTTP.

3. Run `agh memory reindex`.
   - **Expected:** The CLI succeeds and reports the same `indexed_files` count as the HTTP reindex flow.

4. Query the UDS-backed transport directly if available.
   - **Expected:** Search and reindex behave equivalently to HTTP and CLI for the same corpus.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Global-only | `agh memory search "prefs" --scope global` | Matches HTTP global filter behavior |
| Small limit | `--limit 1` | Same top hit as HTTP `limit=1` |
| Empty result | query absent from corpus | Stable empty output, no transport-specific crash |

---

### Related Test Cases

- `SMOKE-001`
- `SMOKE-002`
- `TC-INT-001`

---

### Notes

Parity failures here usually indicate contract drift between client marshalling and API handlers rather than core search logic.
