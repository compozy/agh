## TC-SEC-007: Oversized JSON Payload Rejected with 413 ErrPayloadTooLarge

**Priority:** P0
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the task system enforces payload size limits and rejects oversized JSON payloads with `ErrPayloadTooLarge`. Event payloads are capped at 64KB (`MaxPayloadBytes = 65536`), task metadata at 16KB (`MaxMetadataBytes = 16384`), and run results at 64KB (`MaxResultBytes = 65536`). Oversized payloads must not be persisted.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated principal with full write access
- [ ] Existing task and run available for event and result testing

---

### Test Steps
1. **Task metadata exceeds 16KB limit**
   - Input: `POST /api/tasks` with `metadata` field containing a JSON object > 16,384 bytes (e.g., `{"data": "<16KB+ string>"}`)
   - **Expected:** 413 Request Entity Too Large. Error wraps `ErrPayloadTooLarge`. Task NOT persisted in the store.

2. **Task metadata at exactly 16KB boundary**
   - Input: `POST /api/tasks` with `metadata` JSON object at exactly 16,384 bytes
   - **Expected:** 201 Created. Payload accepted at the boundary.

3. **Task event payload exceeds 64KB limit**
   - Input: Create a task event (e.g., via cancel with oversized metadata) with payload > 65,536 bytes
   - **Expected:** `ErrPayloadTooLarge` returned. Event NOT persisted.

4. **Run result exceeds 64KB limit**
   - Input: `POST /api/task-runs/:id/complete` with `result` JSON > 65,536 bytes
   - **Expected:** 413 Request Entity Too Large. Run status NOT updated to completed. Result NOT persisted.

5. **Run failure metadata exceeds limit**
   - Input: `POST /api/task-runs/:id/fail` with oversized `metadata` field
   - **Expected:** `ErrPayloadTooLarge` returned. Run status unchanged.

6. **Update task with oversized metadata**
   - Input: `PATCH /api/tasks/:id` with `metadata` field > 16KB
   - **Expected:** 413 Request Entity Too Large. Original metadata unchanged.

7. **Verify HTTP error mapping**
   - Input: Trigger any payload-too-large error via HTTP API
   - **Expected:** HTTP status code is exactly 413. Response body includes error classification. No 500 Internal Server Error.

---

### Attack Vectors
- [ ] Denial-of-service via repeated oversized payload submissions to exhaust server memory or disk
- [ ] Payload just over the limit (boundary testing at MaxPayloadBytes + 1)
- [ ] Deeply nested JSON within size limits but designed to consume parsing resources
- [ ] Large number of small fields that collectively exceed the byte limit
- [ ] Compressed payload that expands beyond limits after decompression (if applicable)

---

### Related Test Cases
- TC-SEC-006: SQL injection resistance
- TC-PERF-001: Task creation throughput
