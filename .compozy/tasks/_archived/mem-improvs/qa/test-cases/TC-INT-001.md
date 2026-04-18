## TC-INT-001: HTTP API Search and Reindex Contract

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** HTTP API
**Requirement:** REQ-MEM-003, REQ-MEM-004

---

### Objective

Verify that the HTTP surface exposes the new memory search and reindex operations with the expected request/response contract.

---

### Preconditions

- [ ] Daemon HTTP API is running against a temp corpus.
- [ ] The tester can send `GET /api/memory/search` and `POST /api/memory/reindex`.
- [ ] Searchable global and workspace memories already exist.

---

### Test Steps

1. Call `GET /api/memory/search?q=auth%20sessions&workspace=<workspace-root>`.
   - **Expected:** Response status is `200` and returns an array of search results.

2. Validate the top result fields.
   - **Expected:** `filename`, `scope`, `workspace`, `type`, `name`, `score`, `snippet`, and `mod_time` are present and correct.

3. Call `POST /api/memory/reindex` with a JSON body containing the same workspace.
   - **Expected:** Response status is `200` and returns `indexed_files`, `workspace`, and `completed_at`.

4. Re-run the search after reindex.
   - **Expected:** Search still succeeds and returns the same top result ordering.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Invalid `limit` | `limit=0` or `limit=-1` | `400` with validation error |
| Workspace scope without workspace | `scope=workspace` and no workspace | `400` with validation error |
| Missing matches | query not in corpus | `200` with empty array |

---

### Related Test Cases

- `TC-INT-002`
- `TC-SEC-001`
- `TC-REG-001`

---

### Notes

This case validates the shared transport contract that also feeds generated OpenAPI artifacts.
