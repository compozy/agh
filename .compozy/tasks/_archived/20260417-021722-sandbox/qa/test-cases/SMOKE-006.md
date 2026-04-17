## SMOKE-006: Provider registry resolves backends

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16

---

### Objective

Verify the provider registry correctly resolves `local` and `daytona` backends, and returns an error for unregistered backends.

---

### Preconditions

- [x] Registry created with local and daytona providers registered

---

### Test Steps

1. **Lookup local provider**
   - Input: `registry.Provider(BackendLocal)`
   - **Expected:** Returns local provider, `Backend() == "local"`

2. **Lookup daytona provider**
   - Input: `registry.Provider(BackendDaytona)`
   - **Expected:** Returns daytona provider, `Backend() == "daytona"`

3. **Lookup unregistered backend**
   - Input: `registry.Provider("e2b")`
   - **Expected:** Returns error

4. **Default provider**
   - Input: `registry.DefaultProvider()`
   - **Expected:** Returns local provider
