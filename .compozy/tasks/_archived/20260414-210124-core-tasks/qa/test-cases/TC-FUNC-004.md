## TC-FUNC-004: Update mutable fields (title, description, metadata, owner, network_channel)

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that updating each mutable task field (title, description, metadata_json, owner, network_channel) via TaskPatch succeeds, persists the change, updates updated_at, and that immutable fields remain unchanged. Also validates ClearOwner behavior and the requirement that at least one mutable field must be present in the patch.

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing global task created with known ID, title="Original Title", description="Original", owner={kind:"human", ref:"owner-1"}, metadata={"key":"val"}
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Update title**
   - Input: TaskPatch{Title: ptr("Updated Title")}
   - **Expected:** Returned task has Title="Updated Title"; UpdatedAt > original UpdatedAt; all other fields unchanged

2. **Update description**
   - Input: TaskPatch{Description: ptr("Updated description")}
   - **Expected:** Returned task has Description="Updated description"; UpdatedAt advanced

3. **Update metadata**
   - Input: TaskPatch{Metadata: ptr(json.RawMessage(`{"new_key":"new_val"}`))}
   - **Expected:** Returned task has Metadata=`{"new_key":"new_val"}`; previous metadata fully replaced (not merged)

4. **Update owner**
   - Input: TaskPatch{Owner: &Ownership{Kind:"agent_session", Ref:"session-42"}}
   - **Expected:** Returned task has Owner={Kind:"agent_session", Ref:"session-42"}

5. **Clear owner**
   - Input: TaskPatch{ClearOwner: true}
   - **Expected:** Returned task has Owner=nil

6. **Update network_channel**
   - Input: TaskPatch{NetworkChannel: ptr("chan-new")}
   - **Expected:** Returned task has NetworkChannel="chan-new"

7. **Attempt empty patch (no fields set)**
   - Input: TaskPatch{} (all nil, ClearOwner=false)
   - **Expected:** ErrValidation returned; message indicates at least one mutable field is required

8. **Attempt to set both Owner and ClearOwner**
   - Input: TaskPatch{Owner: &Ownership{Kind:"human", Ref:"x"}, ClearOwner: true}
   - **Expected:** ErrValidation returned; cannot set both owner and clear_owner

9. **Attempt to set title to empty string**
   - Input: TaskPatch{Title: ptr("")}
   - **Expected:** ErrValidation returned; title is required when provided

10. **Verify task.updated event recorded for each successful update**
    - **Expected:** TaskEvent with EventType="task.updated" for each successful step

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Multiple fields in one patch | Title + Description both set | Both updated atomically |
| Metadata set to null JSON | Metadata: ptr(json.RawMessage("null")) | Metadata cleared or stored as null |
| Owner with invalid kind | Owner={Kind:"invalid", Ref:"x"} | ErrValidation |
| Whitespace-only title | Title: ptr("   ") | ErrValidation (title required when provided) |
| Update nonexistent task | ID="nonexistent" | ErrTaskNotFound |

---

### Related Test Cases
- TC-FUNC-001: Create global task with valid fields
- TC-FUNC-005: Attempt to update immutable fields
- TC-FUNC-026: Metadata payload size limit
