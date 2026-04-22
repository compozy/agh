## TC-INT-008: HTTP, UDS, CLI, and Host API Stay Aligned on the Effective Provider

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Explicit surface parity
**Traceability:** Task 04 contract requirements; ADR-004; TechSpec "API Endpoints" and CLI/Host API notes

---

### Objective

Verify that every explicit session create/read surface agrees on provider request semantics and on the effective provider reported for the same session state.

---

### Preconditions

- [ ] HTTP, UDS, CLI, and Host API surfaces are available in the execution environment.
- [ ] One canonical workspace and agent fixture can be used across all surfaces.
- [ ] Provider `B` is valid in the chosen workspace.

---

### Test Steps

1. Create a session with explicit `provider=B` through one explicit surface.
   **Expected:** The create response confirms `provider=B`.

2. Read the same session through the remaining explicit surfaces.
   **Expected:** Every surface reports the same effective provider and agent identity.

3. Create another session with `provider=B` through a different explicit surface, such as Host API or CLI.
   **Expected:** The created session also reports `provider=B` consistently when read elsewhere.

4. Compare output shapes and field visibility across all surfaces.
   **Expected:** No surface omits, renames, or disagrees on the effective provider.

---

### Evidence to Capture

- HTTP request/response payloads.
- UDS request/response payloads.
- CLI create/list/detail output.
- Host API request and response samples.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Provider omitted on one create surface | No provider | Surface still reports resolved default provider on read. |
| Same session read across all surfaces | One persisted session | Provider is identical everywhere. |
| Create initiated from different surfaces | HTTP, CLI, Host API | Downstream reads still agree on the persisted provider. |

---

### Related Test Cases

- `TC-FUNC-003` for create-time invalid provider failure
- `TC-INT-009` for workspace provider option discovery

---

### Notes

Task 08 should keep raw payloads and CLI output side-by-side in the verification report to make drift obvious.
