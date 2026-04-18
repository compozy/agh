## TC-SEC-001: Invalid Scope, Workspace, and Limit Inputs Are Rejected Safely

**Priority:** P1
**Type:** Security
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Memory Validation
**Requirement:** REQ-MEM-008

---

### Objective

Verify that attacker-controlled scope, workspace, and limit parameters are validated consistently and do not mutate catalog state or escape workspace boundaries.

---

### Preconditions

- [ ] HTTP or UDS memory endpoints are reachable.
- [ ] A known-good corpus exists so state changes can be detected.
- [ ] The tester can compare health/search results before and after invalid requests.

---

### Test Steps

1. Capture baseline search and health results for the current corpus.
   - **Expected:** Baseline state is known before negative testing.

2. Call search with invalid `limit` values such as `0`, `-1`, and `not-a-number`.
   - **Expected:** Each request returns a validation error and no server crash occurs.

3. Call search or reindex with `scope=workspace` but no workspace.
   - **Expected:** The request is rejected with a validation error.

4. Call the same endpoints with an unsupported scope such as `bogus`.
   - **Expected:** The request is rejected with a validation error.

5. Re-check search and health after all invalid requests.
   - **Expected:** Catalog state is unchanged and baseline valid operations still succeed.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Whitespace workspace | `"   "` | Treated as missing and rejected when required |
| Unsupported scope | `scope=bogus` | Validation error |
| Invalid JSON body | malformed `POST /api/memory/reindex` | `400` without state mutation |

---

### Related Test Cases

- `TC-INT-001`
- `TC-INT-002`

---

### Notes

This case is about safe rejection and state preservation, not authentication or authorization.
