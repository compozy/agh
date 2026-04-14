## TC-INT-012: Automation-linked agent session creates task with ActorKindAgentSession and automation origin

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-14

---

### Objective
Validate that when an agent session launched by automation creates a task (via tool call or session API), the persisted task record carries `created_by.kind = "agent_session"` and `origin.kind = "automation"`, correctly linking the session actor to its automation origin via `DeriveAutomationLinkedAgentSessionActorContext`.

---

### Preconditions
- [ ] AGH daemon running with all subsystems including automation engine and ACP client
- [ ] At least one automation job configured that launches an agent session
- [ ] The agent session has task-creation capability (tool or API access)
- [ ] Task manager accessible to agent sessions
- [ ] Clean task store (or known baseline count)

---

### Test Steps

1. **Trigger automation that launches an agent session**
   - Input: Trigger the automation job:
     ```http
     POST http://localhost:2123/api/automation/jobs/<job-id>/trigger
     ```
   - **Expected:** Automation job starts and spawns an agent session
   - Record the automation run ID and the spawned session ID

2. **Agent session creates a task during execution**
   - The agent session, while executing its automation-assigned work, creates a task via tool call or direct API invocation
   - This may happen automatically as part of the agent's workflow
   - Alternatively, prompt the session to create a task if the agent supports tool-based task creation

3. **Identify the task created by the agent session**
   - Input: `GET http://localhost:2123/api/tasks`
   - **Expected:** A new task appears
   - **Expected:** The task was created after the automation trigger

4. **Verify ActorKindAgentSession on the created task**
   - Input: `GET http://localhost:2123/api/tasks/<session-created-task-id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.created_by.kind` equals `"agent_session"`
   - **Expected:** `task.task.created_by.ref` equals the session ID that created it

5. **Verify OriginKindAutomation (not OriginKindAgentSession)**
   - **Expected:** `task.task.origin.kind` equals `"automation"` (the session was launched by automation)
   - **Expected:** `task.task.origin.ref` is non-empty and references the automation context (may be the session ref or automation activation ref)

6. **Verify the actor-origin pair is valid**
   - **Expected:** The combination `actor.kind = "agent_session"` with `origin.kind = "automation"` passes `validateActorOriginPair` (this is a valid pair per actors.go: AgentSession allows OriginKindAgentSession or OriginKindAutomation)

7. **Contrast with a non-automation agent session**
   - Create a standalone agent session (not launched by automation)
   - Have it create a task
   - **Expected:** `task.task.created_by.kind` equals `"agent_session"`
   - **Expected:** `task.task.origin.kind` equals `"agent_session"` (not automation, since this session was not automation-launched)
   - **Expected:** `task.task.origin.ref` equals the session ID

8. **Verify audit events carry the correct actor-origin**
   - Input: Check events on the automation-linked task
   - **Expected:** task_created event has `actor.kind = "agent_session"` and `origin.kind = "automation"`
   - Input: Check events on the standalone session task
   - **Expected:** task_created event has `actor.kind = "agent_session"` and `origin.kind = "agent_session"`

---

### Data Validation
| Field | Automation-Linked Session | Standalone Session | Status |
|-------|--------------------------|-------------------|--------|
| created_by.kind | "agent_session" | "agent_session" | [ ] |
| created_by.ref | Session ID | Session ID | [ ] |
| origin.kind | "automation" | "agent_session" | [ ] |
| origin.ref | Automation ref or session ref | Session ID | [ ] |
| Actor-origin pair | Valid (agent_session + automation) | Valid (agent_session + agent_session) | [ ] |
| Event actor.kind | "agent_session" | "agent_session" | [ ] |
| Event origin.kind | "automation" | "agent_session" | [ ] |

---

### Error Scenarios
- [ ] Agent session with `origin.kind = "http"` is rejected by `validateActorOriginPair` (agent_session requires agent_session or automation origin)
- [ ] Agent session with empty actor ref (no session ID) is rejected by validation
- [ ] Automation-linked session with `origin.kind = "cli"` is rejected

---

### Related Test Cases
- TC-INT-011: Automation creates task directly (ActorKindAutomation)
- TC-INT-009: Session bridge creates sessions for task runs
- TC-INT-013: Extension creates task (different actor kind)
