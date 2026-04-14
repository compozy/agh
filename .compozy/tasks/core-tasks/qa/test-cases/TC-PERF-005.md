## TC-PERF-005: ListTasks with Composite Filter on 10K Tasks

**Priority:** P2
**Type:** Performance
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that `ListTasks` with scope + status + owner composite filters returns results within 200ms when querying against a store containing 10,000 tasks. This measures SQLite query plan efficiency, index utilization, and the Go-side filtering and serialization overhead.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] 10,000 tasks seeded in the store with varied attributes:
  - 5,000 global-scoped, 5,000 workspace-scoped (across 10 workspaces)
  - Even distribution across all 7 task statuses
  - 5 distinct owner kinds with varied refs
  - 3 distinct network channels
- [ ] SQLite WAL mode enabled (default AGH configuration)
- [ ] System under normal load

---

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| ListTasks(scope=global, status=ready) | <200ms | <500ms | | [ ] |
| ListTasks(scope=workspace, workspace=ws-01, status=in_progress, owner_kind=human) | <200ms | <500ms | | [ ] |
| ListTasks(network_channel=builders, status=pending) | <200ms | <500ms | | [ ] |
| ListTasks with limit=50 on 10K dataset | <100ms | <200ms | | [ ] |
| ListTasks with no filters (full scan, limit=100) | <200ms | <500ms | | [ ] |
| Result serialization overhead | <50ms | <100ms | | [ ] |

---

### Test Steps
1. **Seed 10,000 tasks with varied attributes**
   - Input: Programmatically create tasks with distributed scope, status, owner, and channel values
   - **Expected:** All 10K tasks persisted. Seeding completes within 30s (bulk insert acceptable).

2. **Query with scope + status filter**
   - Input: `ListTasks(ctx, TaskQuery{Scope: "global", Status: "ready"}, actor)`
   - Record response time
   - **Expected:** Results returned in < 200ms. Only global + ready tasks in response.

3. **Query with scope + workspace + status + owner composite filter**
   - Input: `ListTasks(ctx, TaskQuery{Scope: "workspace", WorkspaceID: "ws-01", Status: "in_progress", OwnerKind: "human", OwnerRef: "user-1"}, actor)`
   - **Expected:** Results returned in < 200ms. Only matching tasks in response.

4. **Query with network channel filter**
   - Input: `ListTasks(ctx, TaskQuery{NetworkChannel: "builders", Status: "pending"}, actor)`
   - **Expected:** Results returned in < 200ms. Only channel-matched pending tasks.

5. **Query with limit parameter**
   - Input: `ListTasks(ctx, TaskQuery{Limit: 50}, actor)` -- broad query, limited results
   - **Expected:** Exactly 50 results returned in < 100ms.

6. **Concurrent query load**
   - Input: 10 concurrent `ListTasks` queries with different filters
   - **Expected:** All 10 complete within < 500ms. SQLite WAL handles concurrent reads.

---

### Related Test Cases
- TC-PERF-001: Sequential task creation throughput
- TC-PERF-006: Observe projection query performance
- SMOKE-003: Basic task listing
