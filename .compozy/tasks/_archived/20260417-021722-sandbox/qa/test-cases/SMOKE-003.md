## SMOKE-003: Config with sandbox profile loads

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16

---

### Objective

Verify that a TOML config containing `[sandboxes.daytona-dev]` section loads and validates without error.

---

### Preconditions

- [x] Config loader available
- [x] SandboxProfile, DaytonaProfile, NetworkProfile types defined

---

### Test Steps

1. **Load TOML config with sandbox profile**
   - Input: Config containing `[sandboxes.daytona-dev]` with `backend = "daytona"`, `sync_mode = "session-bidirectional"`, `persistence = "transient"`, and `[sandboxes.daytona-dev.daytona]` with `api_url`, `target`, `image`, `snapshot`
   - **Expected:** Config loads without error. `Config.Sandboxes["daytona-dev"]` is populated with all fields.

2. **Verify profile fields accessible**
   - **Expected:** `Backend == "daytona"`, `SyncMode == "session-bidirectional"`, `Daytona.Snapshot` populated, `Daytona.Image` populated

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| No environments section | Config without `[sandboxes]` | Loads fine, empty map |
| Multiple profiles | Two environment sections | Both accessible by key |
