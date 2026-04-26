## TC-AUTO-012: Safe Spawn Permission Narrowing And Reaper Lease Release

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify agent-initiated spawn is bounded by daemon-enforced lineage, TTL, depth and child caps,
workspace inheritance, permission subset validation, coordinator-role denial, hooks, and structural
active lease release when the reaper stops a child.

### Traceability

- Task: task_13, Safe Spawn API CLI And Reaper.
- TechSpec: Safe Spawn and Lineage, Permission narrowing, TTL and active leases.
- ADR: ADR-006, ADR-009, ADR-010, ADR-011.
- Resource lesson: Hermes auxiliary/delegated agent references show delegation needs provider/config routing, but AGH QA must prove daemon safety instead of prompt trust.
- Surfaces: `internal/session`, `internal/daemon`, `internal/api/udsapi`, `internal/cli`, `internal/task`, spawn hooks.

### Preconditions

- Active parent managed session with known permission atoms.
- Parent has permission to spawn a child with a strict subset of atoms.
- Child can claim or be assigned an active run for reaper release checks.

### Test Steps

1. Run `agh spawn --agent reviewer --ttl-seconds 1800` with narrowed permissions.
   - **Expected:** Child session is created with durable parent/root/depth/role/TTL metadata and no widened permissions.

2. Attempt spawn with missing TTL, excessive depth, excessive child count, cross-workspace request, unknown permission atom, or superset permission.
   - **Expected:** Each request fails closed with structured error; daemon does not silently narrow and continue.

3. Attempt `--role coordinator` or coordinator-from-coordinator spawn.
   - **Expected:** Request is rejected; coordinators are daemon-managed root sessions only in the MVP.

4. Register `spawn.pre_create` hook that tries to widen permissions.
   - **Expected:** Hook patch is rejected after processing and no child session is created.

5. Give a spawned child an active task-run lease, then trigger TTL expiry or parent stop.
   - **Expected:** Reaper releases the child's active lease through task service with `ttl_expired` or `parent_stopped` before stopping the child.

6. Retry heartbeat/complete with the child's stale claim token after reap.
   - **Expected:** Stale token operations fail explicitly.

### Evidence To Capture

- `qa/logs/TC-AUTO-012/spawn-success.json`
- `qa/logs/TC-AUTO-012/spawn-denials.log`
- `qa/logs/TC-AUTO-012/spawn-hook-widening.log`
- `qa/logs/TC-AUTO-012/reaper-lease-release.log`
- `qa/logs/TC-AUTO-012/stale-child-token.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Exact permissions | child equals parent atoms | Accepted when within caps |
| Subset permissions | one parent tool only | Accepted |
| Unknown atom | `--tool made-up` | Rejected as widening |
| Parent manual stop | stop parent session | Children stop and leases release |

### Related Test Cases

- TC-AUTO-011: Lineage metadata.
- TC-AUTO-007: Stale token fencing.
