## TC-FUNC-001: Valid sandbox profile parses from TOML

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that a complete SandboxProfile with all fields (backend, sync_mode, persistence, runtime_root, env, network, daytona) parses correctly from TOML config.

---

### Preconditions

- [x] Config loader available
- [x] SandboxProfile type defined in `internal/config/config.go`

---

### Test Steps

1. **Load TOML config with complete sandbox profile**
   - Input: TOML with `[sandboxes.test]` containing all fields
   - **Expected:** All fields populated: `Backend`, `SyncMode`, `Persistence`, `RuntimeRoot`, `Env`, `Network.*`, `Daytona.*`

2. **Verify nested DaytonaProfile fields**
   - **Expected:** `Daytona.APIURL`, `Target`, `Image`, `Snapshot`, `Class`, `AutoStop`, `AutoArchive` all populated

3. **Verify nested NetworkProfile fields**
   - **Expected:** `Network.AllowPublicIngress`, `AllowOutbound`, `AllowList`, `DenyList`, `Required` all accessible

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Minimal profile (backend only) | `backend = "local"` | Other fields use zero values |
| Profile with only network section | Missing daytona section | DaytonaProfile is zero-value |
