## TC-REG-004: Permission enforcement unchanged after extraction

**Priority:** P0 (Critical)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 02

---

### Objective

Verify that permission operations (`Authorize`, `PermissionDecision`) through `localToolHost` enforce the same policies as the previous `permissionPolicy` implementation.

---

### Context

Recent changes: Permission methods extracted from `acp/permission.go:94-181` into `localToolHost`.

---

### Critical Path Tests

1. [x] `approve-all` mode permits read/write/terminal operations
2. [x] `deny-all` mode rejects all operations
3. [x] `approve-reads` mode permits reads, denies writes
4. [x] Permission root path is runtime path (not local path for remote)
5. [x] Error messages match expected format
