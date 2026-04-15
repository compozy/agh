## SMOKE-002: Bridge Instance CRUD Round-Trip

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-15

---

### Objective

Verify a bridge instance can be created, read, updated, and listed through the registry without errors.

### Preconditions

- [ ] `internal/bridges` package compiles
- [ ] SQLite test database available via `t.TempDir()`

### Test Steps

1. **Create a bridge instance with platform="slack", scope="global", extension_name="slack"**
   - **Expected:** Instance persisted with status=disabled, generated ID returned

2. **Get the instance by ID**
   - **Expected:** All fields match creation request, provider_config and delivery_defaults preserved

3. **Update display_name and DM policy**
   - **Expected:** Update succeeds, returned instance reflects new values

4. **List instances filtered by platform="slack"**
   - **Expected:** List contains the created instance, no extraneous entries

### Related Test Cases

- TC-FUNC-001, TC-FUNC-003, TC-FUNC-004
