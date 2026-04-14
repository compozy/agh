## TC-INT-013: Extension creates task via host API with ActorKindExtension and capability check

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that an authenticated extension runtime creates a task through the extension host API, the persisted task carries `created_by.kind = "extension"` and `origin.kind = "extension"`, and the operation is gated by the extension's task.write capability.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including extension host
- [ ] At least one extension installed and enabled with `task.write` capability
- [ ] A second extension installed with only `task.read` capability (no write)
- [ ] Task manager accessible to extension host handlers
- [ ] Clean task store (or known baseline count)

---

### Test Steps

1. **Extension with task.write capability creates a task**
   - The extension host invokes `handleTasksCreate` (or equivalent) on behalf of the authorized extension
   - The extension's actor context is derived via `DeriveExtensionActorContext(extensionName, originRef)`
   - Input: Extension sends create-task request through host API with:
     ```json
     {
       "scope": "global",
       "title": "Extension Created Task"
     }
     ```
   - **Expected:** Task created successfully (no error)

2. **Verify ActorKindExtension on the created task**
   - Input: `GET http://localhost:2123/api/tasks/<extension-task-id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.created_by.kind` equals `"extension"`
   - **Expected:** `task.task.created_by.ref` is non-empty and identifies the extension (e.g., extension name or ID)

3. **Verify OriginKindExtension on the created task**
   - **Expected:** `task.task.origin.kind` equals `"extension"`
   - **Expected:** `task.task.origin.ref` is non-empty (extension name or actor ref if originRef was empty)

4. **Verify the actor-origin pair is valid**
   - **Expected:** The combination `actor.kind = "extension"` with `origin.kind = "extension"` passes `validateActorOriginPair` (this is the only valid origin for extensions)

5. **Extension without task.write capability is denied**
   - The read-only extension attempts to create a task
   - **Expected:** The operation is rejected with ErrPermissionDenied
   - **Expected:** If surfaced via HTTP: HTTP 403 Forbidden
   - **Expected:** No task is created in the store

6. **Extension reads tasks (task.read capability)**
   - The read-only extension attempts to list or get tasks
   - **Expected:** Read operations succeed (read capability is sufficient)
   - **Expected:** The tasks list is returned normally

7. **Extension updates a task (requires task.write)**
   - The write-capable extension patches the task title
   - **Expected:** Update succeeds, `updated_at` is refreshed
   - **Expected:** The update event has `actor.kind = "extension"`

8. **Extension without task.write cannot update**
   - The read-only extension attempts to patch a task
   - **Expected:** ErrPermissionDenied / HTTP 403

9. **Verify audit events carry extension actor**
   - Input: Check events on the extension-created task
   - **Expected:** task_created event has `actor.kind = "extension"` and `origin.kind = "extension"`

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| task.created_by.kind | (server-derived) | "extension" | [ ] |
| task.created_by.ref | (server-derived) | Extension name/ID | [ ] |
| task.origin.kind | (server-derived) | "extension" | [ ] |
| task.origin.ref | (server-derived) | Extension name/ID | [ ] |
| Actor-origin pair | validateActorOriginPair | No error | [ ] |
| Read-only ext create | Permission check | ErrPermissionDenied (403) | [ ] |
| Read-only ext list | Permission check | Allowed | [ ] |
| Write ext update | Permission check | Allowed | [ ] |
| Read-only ext update | Permission check | ErrPermissionDenied (403) | [ ] |
| Event actor.kind | task_created event | "extension" | [ ] |

---

### Error Scenarios
- [ ] Extension with `origin.kind = "http"` is rejected by `validateActorOriginPair` (extension requires extension origin)
- [ ] Extension with empty actor ref is rejected by validation
- [ ] Extension with no capabilities at all is denied both read and write
- [ ] Extension attempting to create workspace-scoped task without `create_workspace` authority is denied

---

### Related Test Cases
- TC-INT-011: Automation creates task (different actor kind, same capability pattern)
- TC-INT-012: Agent session creates task (different actor kind)
- TC-INT-014: Network peer creates task (different actor kind, channel binding)
