## TC-FUNC-018: Bridge Instance Source Distinction

**Priority:** P2
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-15

---

### Objective

Validate that bridge instances correctly distinguish between `source=dynamic` (operator-created) and `source=package` (extension-bundle-managed), and that managed sync operations only reconcile package-sourced instances while leaving dynamic instances untouched.

### Preconditions

- [ ] Daemon is running with bridge provider extensions registered
- [ ] SQLite store is available via `t.TempDir()` isolation
- [ ] Understanding of `BridgeInstanceSource` values: `"dynamic"`, `"package"`

### Test Steps

1. **Create a dynamic-sourced instance**
   - Input:
     ```json
     {
       "scope": "global",
       "platform": "telegram",
       "extension_name": "bridges/telegram",
       "display_name": "Manual Telegram",
       "source": "dynamic",
       "enabled": false,
       "status": "disabled"
     }
     ```
   - **Expected:**
     - Instance persists with `source` = `"dynamic"`
     - Instance is updatable via standard CRUD API

2. **Create a package-sourced instance**
   - Input:
     ```json
     {
       "scope": "global",
       "platform": "slack",
       "extension_name": "bridges/slack",
       "display_name": "Bundle Slack",
       "source": "package",
       "enabled": false,
       "status": "disabled"
     }
     ```
   - **Expected:** Instance persists with `source` = `"package"`

3. **Attempt direct update on package-sourced instance**
   - Input: Update `display_name` on the package-sourced instance via CRUD API
   - **Expected:** Rejected with `ErrBridgeInstanceReadOnly` — package-sourced instances are managed and read-only through the generic CRUD surface

4. **Verify dynamic-sourced instance allows direct update**
   - Input: Update `display_name` on the dynamic-sourced instance
   - **Expected:** Update succeeds; new display_name persists

5. **Verify managed sync reconciles package-sourced instances**
   - Input: Simulate a managed sync operation that updates the package-sourced instance's `provider_config`
   - **Expected:**
     - Package-sourced instance is updated through the managed sync path (not CRUD)
     - Dynamic-sourced instance is untouched by managed sync

6. **Verify managed sync removes orphaned package-sourced instances**
   - Input: Managed sync runs with a manifest that no longer includes the package-sourced instance
   - **Expected:**
     - Package-sourced instance is removed or disabled
     - Dynamic-sourced instance remains unaffected

7. **Verify default source is dynamic**
   - Input: Create instance without specifying `source`
   - **Expected:** Normalizes to `source=dynamic`

### Edge Cases & Variations

| Variation                             | Input                                                | Expected Result                                        |
| ------------------------------------- | ---------------------------------------------------- | ------------------------------------------------------ |
| Invalid source value                  | `source: "imported"`                                 | Validation error: "unsupported bridge instance source" |
| Empty source string                   | `source: ""`                                         | Validation error: "bridge instance source is required" |
| List filtered by source               | Filter instances by `source=package`                 | Returns only package-sourced instances                 |
| Switch source from dynamic to package | Update existing dynamic instance to `source=package` | Depends on implementation; may be rejected             |
| Whitespace-padded source              | `source: " package "`                                | Normalized to `"package"`                              |

### Related Test Cases

- TC-FUNC-001 (creation)
- TC-FUNC-003 (update mechanics)
- TC-FUNC-004 (list/get)
