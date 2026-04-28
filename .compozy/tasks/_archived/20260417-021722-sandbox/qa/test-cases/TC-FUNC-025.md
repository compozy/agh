## TC-FUNC-025: SandboxProfile.Env map parses key-value pairs

**Priority:** P2 (Medium)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that the `Env` map field in `SandboxProfile` correctly parses key-value pairs from TOML and preserves them through config loading.

---

### Preconditions

- [x] SandboxProfile.Env field defined as `map[string]string`

---

### Test Steps

1. **Load config with env map**
   - Input: `[sandboxes.test.env]` with `KEY1 = "value1"`, `KEY2 = "value2"`
   - **Expected:** `profile.Env["KEY1"] == "value1"`, `profile.Env["KEY2"] == "value2"`

2. **Verify empty env map**
   - Input: No `[sandboxes.test.env]` section
   - **Expected:** `profile.Env` is nil or empty map

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Special chars in values | `VAR = "hello=world"` | Preserved correctly |
| Empty value | `VAR = ""` | Empty string stored |
