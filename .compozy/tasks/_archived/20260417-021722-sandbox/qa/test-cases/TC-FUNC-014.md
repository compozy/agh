## TC-FUNC-014: Session stop calls Destroy when DestroyOnStop

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that `Provider.Destroy()` is called when the sandbox profile has `Persistence = "transient"` (DestroyOnStop), and is NOT called when persistence is `"reuse"` or `"archive"`.

---

### Preconditions

- [x] Mock provider tracks Destroy calls
- [x] Different profiles with different persistence settings

---

### Test Steps

1. **Stop session with transient persistence**
   - Input: Profile with `persistence = "transient"`
   - **Expected:** `Destroy()` called after SyncFromRuntime

2. **Stop session with reuse persistence**
   - Input: Profile with `persistence = "reuse"`
   - **Expected:** `Destroy()` NOT called, sandbox left for reuse

3. **Stop session with archive persistence**
   - Input: Profile with `persistence = "archive"`
   - **Expected:** `Destroy()` called with archive semantics (provider archives, not deletes)
