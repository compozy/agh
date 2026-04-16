## TC-FUNC-004: Invalid sync_mode returns validation error

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that an environment profile with an invalid sync_mode value is rejected during config validation.

---

### Preconditions

- [x] Config validation checks sync_mode against known values (`none`, `session-bidirectional`)

---

### Test Steps

1. **Load config with invalid sync_mode**
   - Input: `sync_mode = "real-time"` (not a valid mode)
   - **Expected:** Validation error returned

2. **Load config with valid sync_mode**
   - Input: `sync_mode = "session-bidirectional"`
   - **Expected:** Loads without error

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Empty sync_mode | `sync_mode = ""` | Defaults to `"none"` or error |
| Reserved mode | `sync_mode = "turn-bidirectional"` | May be accepted as reserved |
