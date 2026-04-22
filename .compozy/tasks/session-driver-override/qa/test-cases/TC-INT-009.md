## TC-INT-009: Workspace Detail Exposes Sorted Provider Options for the Dialog

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Workspace provider catalog
**Traceability:** Task 05 requirements on `WorkspaceDetailPayload.providers`; ADR-004; TechSpec `GET /api/workspaces/{id}` and "Testing Approach"

---

### Objective

Verify that workspace detail surfaces expose the provider options visible in the resolved workspace config and that the list is stable enough for direct UI consumption.

---

### Preconditions

- [ ] Workspace fixture `WORKSPACE-CATALOG` exists with multiple visible providers.
- [ ] HTTP or UDS workspace detail access is available.
- [ ] The expected provider ordering is known from the fixture.

---

### Test Steps

1. Request the target workspace detail payload.
   **Expected:** The payload includes `providers`.

2. Inspect the provider option list.
   **Expected:** The list matches the providers visible in the resolved workspace config and appears in stable sorted order.

3. Compare the same payload against the web dialog picker inputs.
   **Expected:** The web client can consume the payload without re-deriving provider options from other config assumptions.

---

### Evidence to Capture

- Workspace detail response payload showing `providers`.
- Provider list ordering comparison against the fixture definition.
- One browser or adapter capture showing the same options appear in the dialog picker.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Single-provider workspace | One visible provider | Payload still includes a stable list. |
| Multi-provider workspace | Several visible providers | Ordering remains deterministic. |
| Provider removed from workspace | Updated fixture | Removed provider disappears from the payload and picker. |

---

### Related Test Cases

- `TC-UI-010` for dialog picker rendering
- `TC-INT-008` for the explicit surfaces that consume the same workspace context

---

### Notes

This case focuses on provider discovery, not dialog behavior. Dialog submission and prefill belong to `TC-UI-010`.
Task 08 should still keep the existing repo regression coverage for task 05 automatic internal creators in the full lane so empty-provider defaults stay explicit.
