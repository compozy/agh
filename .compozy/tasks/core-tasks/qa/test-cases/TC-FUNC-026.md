## TC-FUNC-026: Create task with metadata_json > 16KB returns ErrPayloadTooLarge

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-14

---

### Objective
Validate that creating a task with metadata_json exceeding MaxMetadataBytes (16 KiB = 16,384 bytes) is rejected with ErrPayloadTooLarge. Valid metadata at or just below the limit must be accepted.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] ActorContext with Authority.Write=true, Authority.CreateGlobal=true

---

### Test Steps

1. **Create a task with metadata exactly at the 16KB boundary**
   - Generate valid JSON metadata of exactly 16,384 bytes
   - Input: scope="global", title="At limit", metadata=<16384 byte JSON>
   - **Expected:** Task created successfully; no error

2. **Create a task with metadata at 16KB + 1 byte**
   - Generate valid JSON metadata of 16,385 bytes
   - Input: scope="global", title="Over limit", metadata=<16385 byte JSON>
   - **Expected:** Error returned; `errors.Is(err, ErrPayloadTooLarge)` == true; error message contains "metadata" and "16384"

3. **Verify the over-limit task was not persisted**
   - Query store for task with title "Over limit"
   - **Expected:** No such task exists

4. **Create a task with empty metadata**
   - Input: scope="global", title="No metadata", metadata=nil
   - **Expected:** Task created successfully; metadata is nil/empty

5. **Update a task with metadata exceeding 16KB via TaskPatch**
   - Patch existing task with Metadata > 16,384 bytes
   - **Expected:** ErrPayloadTooLarge

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exactly 16,384 bytes | At boundary | Success |
| 16,385 bytes | One over | ErrPayloadTooLarge |
| 0 bytes (nil) | No metadata | Success |
| Valid JSON "null" | metadata=null | Success (0 effective bytes) |
| Invalid JSON | metadata=`{broken` | ErrValidation (not valid JSON) |
| Whitespace-padded JSON | `  {"k":"v"}  ` | Size computed after trimming whitespace |

---

### Related Test Cases
- TC-FUNC-001: Create global task with valid fields
- TC-FUNC-004: Update mutable fields
- TC-FUNC-027: Complete run with result_json > 64KB
- TC-FUNC-028: Create task event with payload_json > 64KB
