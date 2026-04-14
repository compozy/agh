## TC-SEC-001: Server-Derived created_by Identity Ignores Client Payload

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the task system enforces server-derived identity for the `created_by` field. When a client submits `created_by_kind` and `created_by_ref` in the create-task payload, the server MUST ignore those values and derive the actor identity from the authenticated principal context instead.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated human principal available via CLI or HTTP ingress
- [ ] Access to task creation endpoint (`POST /api/tasks`)

---

### Test Steps
1. **Submit create-task request with spoofed created_by fields**
   - Input: `POST /api/tasks` with JSON body:
     ```json
     {
       "scope": "global",
       "title": "Spoofed identity task",
       "created_by_kind": "daemon",
       "created_by_ref": "injected-daemon-ref"
     }
     ```
   - **Expected:** 201 Created. Response `created_by.kind` equals the authenticated principal kind (e.g., `"human"`), NOT `"daemon"`. Response `created_by.ref` equals the authenticated user ref, NOT `"injected-daemon-ref"`.

2. **Verify persisted task via GET**
   - Input: `GET /api/tasks/:id` using the ID from step 1
   - **Expected:** `task.created_by.kind` and `task.created_by.ref` match server-derived values. No trace of the spoofed values in the response or audit events.

3. **Attempt spoofing via all 6 actor kinds**
   - Input: Repeat step 1 with `created_by_kind` set to each of: `human`, `agent_session`, `automation`, `extension`, `network_peer`, `daemon`
   - **Expected:** All responses ignore the submitted `created_by_kind` and derive identity from the actual authenticated principal.

4. **Verify audit event records server-derived identity**
   - Input: `GET /api/tasks/:id` and inspect the `events` array for the `task.created` event
   - **Expected:** The `actor` field on the audit event matches the server-derived principal, not the spoofed payload values.

---

### Attack Vectors
- [ ] Privilege escalation by spoofing `created_by_kind: "daemon"` to gain system-level attribution
- [ ] Identity impersonation by setting `created_by_ref` to another user's reference
- [ ] Replay of a legitimate principal's identity fields from a captured response
- [ ] Injection of `created_by_kind` and `created_by_ref` as top-level JSON fields
- [ ] Injection of nested `created_by: {"kind": "daemon", "ref": "..."}` object in payload

---

### Related Test Cases
- TC-SEC-002: Server-derived origin identity ignores client payload
- TC-SEC-003: Unauthenticated request rejection
