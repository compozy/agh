## TC-AUTO-011: Session Lineage And Spawn Metadata Persistence

**Priority:** P1 (High)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify root, coordinator, and spawned session metadata are typed, durable, queryable after restart,
and exposed through public read models without leaking internal policy details.

### Traceability

- Task: task_12, Session Lineage And Spawn Metadata.
- TechSpec: Safe Spawn and Lineage, Data Models, Manual Control Contract.
- ADR: ADR-006, ADR-009, ADR-010, ADR-011.
- Resource lesson: Paperclip agent management references require durable agent/session identity and hierarchy before autonomous delegation.
- Surfaces: `internal/session`, `internal/store/globaldb`, session read DTOs, generated web/session types.

### Preconditions

- Isolated global DB.
- Ability to create manual user, coordinator, and spawned session records.
- Public session list/get endpoint or conversion helper is available.

### Test Steps

1. Create manual user, dream, or system sessions.
   - **Expected:** Sessions are root rows with no parent, `root_session_id=session_id`, depth `0`, and manual flows unchanged.

2. Create a coordinator session with future TTL.
   - **Expected:** Session type is `coordinator`, root metadata is valid, and invalid/missing TTL is rejected where required.

3. Create a spawned child session with parent/root/depth/role/budget/policy metadata.
   - **Expected:** Lineage validates and persists with typed columns/read models.

4. Reopen DB or restart daemon and list/filter sessions by type, parent, root, and role.
   - **Expected:** Lineage is durable and queryable without scanning opaque JSON.

5. Convert sessions to public DTOs and generated web types.
   - **Expected:** DTOs expose lineage fields needed by operators and web, while internal-only permission policy remains safe.

### Evidence To Capture

- `qa/logs/TC-AUTO-011/session-root.json`
- `qa/logs/TC-AUTO-011/session-lineage-db.log`
- `qa/logs/TC-AUTO-011/session-restart-read.json`
- `qa/logs/TC-AUTO-011/session-dto-redaction.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Expired TTL | coordinator/spawned TTL in past | Validation error |
| Missing root | child without root | Validation error |
| Manual session | no parent | Root metadata synthesized correctly |
| Invalid depth | negative or skipped depth | Validation error |

### Related Test Cases

- TC-AUTO-012: Spawn API consumes lineage.
- TC-AUTO-013: Coordinator session bootstrap uses coordinator type.
