## TC-INT-001: Config -> workspace -> environment resolution round-trip

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Tasks:** 01, 04

---

### Objective

Verify the full integration path from TOML config with sandbox profiles, through workspace registration with `sandbox_ref`, to workspace resolution producing a correct `ResolvedWorkspace.Sandbox`.

---

### Preconditions

- [x] Config loader, workspace resolver, and globaldb all available
- [x] `sandbox_ref` column exists in workspaces table

---

### Test Steps

1. **Load TOML config with `[sandboxes.daytona-dev]` profile**
   - **Expected:** Config loads, `Environments["daytona-dev"]` populated

2. **Register workspace with `SandboxRef = "daytona-dev"`**
   - **Expected:** Workspace persisted to DB with `sandbox_ref = "daytona-dev"`

3. **Resolve workspace**
   - **Expected:** `ResolvedWorkspace.Sandbox.Backend == "daytona"`, `DaytonaConfig` populated from profile

4. **Verify round-trip: load workspace from DB, resolve again**
   - **Expected:** Same resolved environment as step 3

---

### Data Validation

| Field | Source Value | Resolved Value | Status |
|-------|-------------|----------------|--------|
| Backend | `"daytona"` | `BackendDaytona` | [ ] |
| SyncMode | `"session-bidirectional"` | `SyncModeSessionBidirectional` | [ ] |
| Daytona.APIURL | Config value | Same | [ ] |
| Daytona.Snapshot | Config value | Same | [ ] |
