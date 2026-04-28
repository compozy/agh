## TC-REG-006: Existing session manager tests pass

**Priority:** P0 (Critical)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 04

---

### Objective

Verify that all pre-existing session manager tests pass after the sandbox lifecycle integration, confirming no regression in session create/stop/resume flows.

---

### Test Steps

1. **Run session package tests**
   - Input: `go test ./internal/session/ -race`
   - **Expected:** All tests pass

2. **Run session integration tests**
   - Input: `go test -tags integration ./internal/session/ -race`
   - **Expected:** All integration tests pass
