## TC-FUNC-002: Snapshot wins over Image in DaytonaProfile

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that when both `snapshot` and `image` are set in a DaytonaProfile, the resolved profile policy uses `snapshot` as the startup input, treating `image` as documentation/fallback.

---

### Preconditions

- [x] DaytonaProfile type defined with both Snapshot and Image fields

---

### Test Steps

1. **Create profile with both snapshot and image**
   - Input: `snapshot = "snap-abc"`, `image = "ubuntu:22.04"`
   - **Expected:** Both fields stored. During provider Prepare, snapshot is used for sandbox creation.

2. **Create profile with snapshot only**
   - Input: `snapshot = "snap-abc"`, `image = ""`
   - **Expected:** Snapshot used, image empty

3. **Create profile with image only (no snapshot)**
   - Input: `snapshot = ""`, `image = "ubuntu:22.04"`
   - **Expected:** Image used as fallback for sandbox creation

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Both empty | `snapshot = ""`, `image = ""` | Validation error or provider uses default |
| Snapshot missing at runtime | Snapshot ID references deleted snapshot | Provider.Prepare returns actionable error |
