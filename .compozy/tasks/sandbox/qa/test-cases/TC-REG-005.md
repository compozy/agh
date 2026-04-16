## TC-REG-005: Existing ACP client_integration_test passes

**Priority:** P0 (Critical)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 02

---

### Objective

Verify that the existing `internal/acp/client_integration_test.go` passes without any modifications, confirming the Launcher/ToolHost extraction has zero observable impact on the ACP protocol layer.

---

### Test Steps

1. **Run existing ACP integration tests**
   - Input: `go test -tags integration ./internal/acp/ -run Integration -race`
   - **Expected:** All tests pass, zero modifications needed

2. **Run existing ACP unit tests**
   - Input: `go test ./internal/acp/ -race`
   - **Expected:** All tests pass
