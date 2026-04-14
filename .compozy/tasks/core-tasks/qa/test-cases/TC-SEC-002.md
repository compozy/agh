## TC-SEC-002: Server-Derived Origin Identity Ignores Client Payload

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the task system enforces server-derived identity for the `origin` field. When a client submits `origin_kind` and `origin_ref` in the create-task payload, the server MUST ignore those values and derive the origin from the authenticated ingress context instead.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated human principal available via HTTP ingress
- [ ] Access to task creation endpoint (`POST /api/tasks`)

---

### Test Steps
1. **Submit create-task request with spoofed origin fields**
   - Input: `POST /api/tasks` with JSON body:
     ```json
     {
       "scope": "global",
       "title": "Spoofed origin task",
       "origin_kind": "network",
       "origin_ref": "injected-peer-channel"
     }
     ```
   - **Expected:** 201 Created. Response `origin.kind` equals the actual ingress origin (e.g., `"http"` for HTTP API, `"cli"` for CLI, `"uds"` for UDS). Response `origin.ref` equals the server-determined reference, NOT `"injected-peer-channel"`.

2. **Verify persisted task origin via GET**
   - Input: `GET /api/tasks/:id` using the ID from step 1
   - **Expected:** `task.origin.kind` and `task.origin.ref` match the real ingress surface, not the spoofed values.

3. **Attempt spoofing via all 9 origin kinds**
   - Input: Repeat step 1 with `origin_kind` set to each of: `cli`, `web`, `uds`, `http`, `automation`, `extension`, `network`, `agent_session`, `daemon`
   - **Expected:** All responses ignore the submitted `origin_kind` and derive the origin from the actual transport layer.

4. **Cross-verify with audit trail**
   - Input: Inspect the `task.created` event in the task detail response
   - **Expected:** The `origin` field on the audit event matches the server-derived ingress origin, confirming no spoofing occurred at any layer.

---

### Attack Vectors
- [ ] Origin spoofing to masquerade HTTP requests as CLI or daemon-internal writes
- [ ] Network origin injection to bypass channel validation on HTTP-originated requests
- [ ] Injection of `origin_kind: "daemon"` to make writes appear as trusted system operations
- [ ] Simultaneous spoofing of both `created_by` and `origin` fields to fabricate a fully forged identity

---

### Related Test Cases
- TC-SEC-001: Server-derived created_by identity ignores client payload
- TC-SEC-005: Network peer channel mismatch validation
