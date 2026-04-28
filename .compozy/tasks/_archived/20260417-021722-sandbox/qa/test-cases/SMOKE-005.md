## SMOKE-005: Session list shows environment backend

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16

---

### Objective

Verify session list/get/status API responses include the `environment` field with backend, state, and sandbox ID.

---

### Preconditions

- [x] At least one session created with environment integration
- [x] `SessionSandboxPayload` added to contract types
- [x] Conversion function maps sandbox fields

---

### Test Steps

1. **Create a session and list sessions**
   - **Expected:** Session payload includes `environment` object with `backend: "local"`, `sandbox_id` non-empty, `state` reflecting current lifecycle phase

2. **Get session info**
   - **Expected:** `SessionPayload.Environment` populated with `sandbox_id`, `backend`, `profile`, `state`

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Session with no environment (legacy) | Old session without env metadata | `environment` field omitted or null |
