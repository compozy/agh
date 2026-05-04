## TC-FUNC-001: Memory Health And Operation History Visibility

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify memory health and operation history are visible through CLI and API, bounded and redacted, filterable by operator dimensions, durable across restart, and not wired into runtime prompt assembly.

### Traceability

- Task: task_07, Memory Visibility and Future Interfaces.
- TechSpec: issues 33, 34, 35, and 60; Testing Approach memory health/history CLI output and future interface boundaries.
- ADR: ADR-005 memory health/history before runtime context references.
- Surfaces: `internal/memory`, `internal/api/core/memory.go`, `internal/api/contract`, `internal/cli/memory.go`, UDS/HTTP routes, generated OpenAPI types, site memory docs.

### Preconditions

- Isolated AGH home and workspace with memory enabled.
- Memory operation fixtures include global and workspace scopes, writes, reads, searches, deletes, reindex operations, and sentinel secret text.
- Daemon/API fixture can be restarted without deleting the memory catalog.

### Test Steps

1. Run `agh memory health -o json` and `GET /api/memory/health`.
   - **Expected:** CLI and API report consistent configured/degraded/unavailable states, scope counts, dream state, operation counts, and last operation where available.

2. Generate memory operations across scopes and operations.
   - **Expected:** Operation log persists scope, workspace, filename, operation, bounded summary, and timestamp.

3. Run `agh memory history` and `GET /api/memory/history` with workspace, scope, operation, since/until, and limit filters.
   - **Expected:** CLI and API return the same bounded redacted rows and respect filters.

4. Restart the daemon or reload service fixtures.
   - **Expected:** Operation history remains available from durable catalog state.

5. Send or assemble a normal prompt with strings that look like future context refs.
   - **Expected:** Runtime prompt assembly does not resolve or inject `@file`, `@folder`, `@git`, or `@url`; future interfaces remain compile-time seams only.

6. Validate web and site surfaces.
   - **Expected:** Generated OpenAPI types expose `getMemoryHealth` and `listMemoryHistory`; site memory CLI/API docs describe health/history and state that history is not prompt context.

### Evidence To Capture

- `qa/logs/TC-FUNC-001/go-test-memory.log`
- `qa/logs/TC-FUNC-001/memory-health-cli.json`
- `qa/logs/TC-FUNC-001/memory-health-api.json`
- `qa/logs/TC-FUNC-001/memory-history-filtered.json`
- `qa/logs/TC-FUNC-001/prompt-isolation.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Disabled memory | Memory disabled config | Health reports unavailable/disabled clearly |
| Secret summary | Sentinel secret in operation input | History summary redacted |
| Low limit | `--limit 1` | Only one row returned |
| Future context ref | `@file:secret.txt` in prompt | No resolver invoked |

### Related Test Cases

- TC-UI-003: Web generated contract compatibility.
- TC-REG-002: Site memory docs.
