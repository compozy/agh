## TC-FUNC-028: Create task event with payload_json > 64KB returns ErrPayloadTooLarge

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-14

---

### Objective
Validate that creating a task event with payload_json exceeding MaxPayloadBytes (64 KiB = 65,536 bytes) is rejected with ErrPayloadTooLarge via the TaskEvent.Validate() method and any operation that emits events with oversized payloads.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task with known ID
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Validate a TaskEvent with payload at the 64KB boundary**
   - Construct a TaskEvent with Payload of exactly 65,536 bytes of valid JSON
   - Call TaskEvent.Validate()
   - **Expected:** No error returned

2. **Validate a TaskEvent with payload at 64KB + 1 byte**
   - Construct a TaskEvent with Payload of 65,537 bytes of valid JSON
   - Call TaskEvent.Validate()
   - **Expected:** Error returned; `errors.Is(err, ErrPayloadTooLarge)` == true; error message contains "task_event.payload" and "65536"

3. **Validate a TaskEvent with nil payload**
   - Construct a TaskEvent with Payload = nil
   - Call TaskEvent.Validate()
   - **Expected:** No error (payload is optional)

4. **Test through CancelTask metadata (which becomes event payload)**
   - Call CancelTask with Metadata > 64KB
   - **Expected:** ErrPayloadTooLarge from CancelTask.Validate

5. **Test through CancelRun metadata**
   - Call CancelRun with Metadata > 64KB
   - **Expected:** ErrPayloadTooLarge from CancelRun.Validate

6. **Direct validation: ValidatePayloadSize**
   - Call ValidatePayloadSize with 65,537 byte payload
   - **Expected:** ErrPayloadTooLarge

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exactly 65,536 bytes | At boundary | Success |
| 65,537 bytes | One over | ErrPayloadTooLarge |
| 0 bytes (nil) | No payload | Success |
| Empty JSON object | `{}` | Success (2 bytes) |
| Invalid JSON payload | `{broken` | ErrValidation (not ErrPayloadTooLarge) |
| Whitespace-padded JSON | `   {"k":"v"}   ` | Size computed after trimming |

---

### Related Test Cases
- TC-FUNC-026: Create task with metadata_json > 16KB
- TC-FUNC-027: Complete run with result_json > 64KB
