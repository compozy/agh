## TC-REG-001: ACP session lifecycle unchanged after extraction

**Priority:** P0 (Critical)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-16
**Task:** 02
**Original Feature:** ACP subprocess spawn and JSON-RPC communication

---

### Objective

Verify that the Launcher/ToolHost extraction produces zero observable behavior change for ACP sessions. The existing `client_integration_test.go` must pass unmodified.

---

### Context

Recent changes that may affect this feature:
- `spawnProcess` extracted from `acp/client.go` into `localLauncher`
- File IO handlers extracted from `acp/handlers.go` into `localToolHost`
- Terminal handlers extracted into `localToolHost`
- Permission methods extracted into `localToolHost`

---

### Critical Path Tests

1. [x] ACP session create -> negotiate -> prompt -> response works
2. [x] JSON-RPC over stdio pipe is clean (no extra bytes)
3. [x] Agent process starts with correct working directory
4. [x] Agent process receives correct environment variables

---

### Integration Points

- [x] `client_integration_test.go` passes without modification
- [x] `handlers_test.go` passes without modification
- [x] Session manager tests pass without modification
