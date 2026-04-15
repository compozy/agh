## SMOKE-008: All Eight Providers Compile and Pass Unit Tests

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-15

---

### Objective

Verify all eight provider extension packages compile without errors and their unit test suites pass with the `-race` flag.

### Preconditions

- [ ] Go toolchain installed matching `go.mod`
- [ ] All provider dependencies resolved

### Test Steps

1. **Run `go build ./extensions/bridges/...`**
   - **Expected:** All 8 providers compile without errors

2. **Run `go test -race ./extensions/bridges/slack/`**
   - **Expected:** All tests pass

3. **Run `go test -race ./extensions/bridges/discord/`**
   - **Expected:** All tests pass

4. **Run `go test -race ./extensions/bridges/telegram/`**
   - **Expected:** All tests pass

5. **Run `go test -race ./extensions/bridges/teams/`**
   - **Expected:** All tests pass

6. **Run `go test -race ./extensions/bridges/whatsapp/`**
   - **Expected:** All tests pass

7. **Run `go test -race ./extensions/bridges/gchat/`**
   - **Expected:** All tests pass

8. **Run `go test -race ./extensions/bridges/github/`**
   - **Expected:** All tests pass

9. **Run `go test -race ./extensions/bridges/linear/`**
   - **Expected:** All tests pass

### Related Test Cases

- TC-INT-012
