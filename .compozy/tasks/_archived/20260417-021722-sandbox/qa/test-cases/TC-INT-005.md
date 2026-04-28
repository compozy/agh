## TC-INT-005: Concurrent sessions same workspace no corruption

**Priority:** P2 (Medium)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that two concurrent sessions referencing the same workspace with local provider can both modify the same file without data corruption, using last-write-wins semantics.

---

### Test Steps

1. **Create two sessions on the same workspace**
   - **Expected:** Both sessions created, each with unique SandboxID

2. **Both sessions write to the same file**
   - Input: Session A writes "contentA", Session B writes "contentB"
   - **Expected:** No panic, no corruption, file contains one of the two values (last write wins)

3. **Stop both sessions**
   - **Expected:** Both stop cleanly, no errors

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Interleaved writes | Rapid alternating writes | Last writer wins |
| Different files | Each writes to different file | Both files correct |
