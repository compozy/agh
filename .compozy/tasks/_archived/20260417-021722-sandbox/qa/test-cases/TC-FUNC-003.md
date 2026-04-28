## TC-FUNC-003: Invalid backend returns validation error

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that a sandbox profile with an invalid backend value is rejected during config validation.

---

### Preconditions

- [x] Config validation checks backend against known values

---

### Test Steps

1. **Load config with invalid backend**
   - Input: `backend = "kubernetes"` (not a registered backend)
   - **Expected:** Validation error returned, message includes the invalid backend name

2. **Load config with empty backend**
   - Input: `backend = ""`
   - **Expected:** Validation error or defaults to `"local"`

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Misspelled backend | `backend = "locall"` | Validation error |
| Case sensitivity | `backend = "Local"` | Error or case-normalized |
