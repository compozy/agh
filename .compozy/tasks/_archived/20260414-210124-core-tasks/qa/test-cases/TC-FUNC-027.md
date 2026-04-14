## TC-FUNC-027: Complete run with result_json > 64KB returns ErrPayloadTooLarge

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-14

---

### Objective
Validate that completing a run with result_json exceeding MaxResultBytes (64 KiB = 65,536 bytes) is rejected with ErrPayloadTooLarge. Valid results at or just below the limit must be accepted.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with a running run (status="running")
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Complete the run with result exactly at the 64KB boundary**
   - Generate valid JSON result of exactly 65,536 bytes
   - Call CompleteRun(ctx, runID, RunResult{Value: <65536 byte JSON>}, actor)
   - **Expected:** Run completed successfully; result stored

2. **Create another running run and attempt to complete with result at 64KB + 1 byte**
   - Generate valid JSON result of 65,537 bytes
   - Call CompleteRun(ctx, run2ID, RunResult{Value: <65537 byte JSON>}, actor)
   - **Expected:** Error returned; `errors.Is(err, ErrPayloadTooLarge)` == true; error message contains "result" and "65536"

3. **Verify the over-limit run was not modified**
   - Read run2 from store
   - **Expected:** Status still == "running"; no result stored; no EndedAt set

4. **Complete a run with nil/empty result**
   - Create another running run, complete with RunResult{Value: nil}
   - **Expected:** Run completed successfully; result is nil

5. **Direct validation: ValidateResultSize**
   - Call ValidateResultSize with 65,537 byte payload
   - **Expected:** ErrPayloadTooLarge

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exactly 65,536 bytes | At boundary | Success |
| 65,537 bytes | One over | ErrPayloadTooLarge |
| 0 bytes (nil) | No result | Success |
| Invalid JSON result | `{broken` | ErrValidation |
| Large valid JSON under limit | 60KB | Success |

---

### Related Test Cases
- TC-FUNC-017: Complete running run with result
- TC-FUNC-026: Create task with metadata_json > 16KB
- TC-FUNC-028: Create task event with payload_json > 64KB
