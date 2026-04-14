## TC-INT-011: Automation creates task directly with ActorKindAutomation and OriginKindAutomation

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that when the automation subsystem creates a task directly (not through an agent session), the persisted task record carries `created_by.kind = "automation"` and `origin.kind = "automation"`, with the correct actor and origin references derived from `DeriveAutomationActorContext`.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including automation engine
- [ ] At least one automation job configured and enabled
- [ ] Task manager accessible to the automation subsystem
- [ ] Clean task store (or known baseline)

---

### Test Steps

1. **Trigger an automation job that creates a task directly**
   - Input: Trigger the automation job via:
     ```http
     POST http://localhost:2123/api/automation/jobs/<job-id>/trigger
     ```
     Or use a configured trigger that fires automatically.
   - **Expected:** The automation job runs successfully

2. **Identify the task created by automation**
   - Input: `GET http://localhost:2123/api/tasks` (list all tasks)
   - **Expected:** A new task appears that was not present before the trigger
   - **Expected:** The task can be identified by its origin or metadata

3. **Verify ActorKindAutomation on the created task**
   - Input: `GET http://localhost:2123/api/tasks/<automation-task-id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.created_by.kind` equals `"automation"`
   - **Expected:** `task.task.created_by.ref` is non-empty and identifies the automation job/flow (e.g., job ID or automation name)

4. **Verify OriginKindAutomation on the created task**
   - **Expected:** `task.task.origin.kind` equals `"automation"`
   - **Expected:** `task.task.origin.ref` is non-empty (matches the automation actor ref or includes job context)

5. **Verify the actor-origin pair is valid**
   - **Expected:** The combination `actor.kind = "automation"` with `origin.kind = "automation"` passes the `validateActorOriginPair` check (this is a valid pair per the actors.go rules)

6. **Verify authority was granted**
   - **Expected:** The automation context has `FullAccessAuthority()` (read=true, write=true, create_global=true, create_workspace=true)
   - **Expected:** The task was created without ErrPermissionDenied

7. **Verify task fields are correct**
   - **Expected:** `task.task.scope` is valid (global or workspace as configured by automation)
   - **Expected:** `task.task.title` is non-empty
   - **Expected:** `task.task.status` is "pending"

8. **Verify audit events carry automation actor**
   - **Expected:** The task_created event in `task.events` has `actor.kind = "automation"` and `origin.kind = "automation"`

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| task.created_by.kind | (server-derived) | "automation" | [ ] |
| task.created_by.ref | (server-derived) | Non-empty automation ref | [ ] |
| task.origin.kind | (server-derived) | "automation" | [ ] |
| task.origin.ref | (server-derived) | Non-empty automation ref | [ ] |
| task.status | (server-derived) | "pending" | [ ] |
| Actor-origin pair validity | validateActorOriginPair | No error | [ ] |
| Event actor.kind | task_created event | "automation" | [ ] |
| Event origin.kind | task_created event | "automation" | [ ] |

---

### Error Scenarios
- [ ] Automation with incorrect origin kind (e.g., "http") is rejected by `validateActorOriginPair` (ActorKindAutomation requires OriginKindAutomation)
- [ ] Automation with empty actor ref is rejected by `ActorIdentity.Validate`
- [ ] Automation without write authority cannot create tasks (ErrPermissionDenied)

---

### Related Test Cases
- TC-INT-012: Automation-linked agent session creates task (ActorKindAgentSession + OriginKindAutomation)
- TC-INT-013: Extension creates task (ActorKindExtension)
- TC-INT-014: Network peer creates task (ActorKindNetworkPeer)
