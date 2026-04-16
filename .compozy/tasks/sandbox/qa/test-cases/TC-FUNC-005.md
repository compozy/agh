## TC-FUNC-005: Environment overlay merge preserves provider fields

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that workspace-scoped environment profile overlays merge correctly with global config, preserving provider-specific fields (Daytona, Network) across merge boundaries.

---

### Preconditions

- [x] Config merge system supports `[environments.*]` overlay
- [x] `environmentOverlay.Apply()` implemented

---

### Test Steps

1. **Load global config with daytona profile**
   - Input: Global config sets `api_url`, `target`, `image`
   - **Expected:** Profile fully populated

2. **Apply workspace overlay that overrides subset of fields**
   - Input: Workspace overlay sets `image = "custom:latest"` but not `api_url` or `target`
   - **Expected:** Merged profile has `api_url` and `target` from global, `image` from overlay

3. **Verify Env map merges (not replaces)**
   - Input: Global sets `Env = {"KEY1": "val1"}`, overlay sets `Env = {"KEY2": "val2"}`
   - **Expected:** Merged `Env` has both `KEY1` and `KEY2`

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Overlay with nil Env | No Env in overlay | Global Env preserved |
| Overlay overrides Network.Required | `required = true` in overlay | Merged value is true |
