## TC-INT-008: CLI workspace add --environment flag

**Priority:** P2 (Medium)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that the `agh workspace add` and `agh workspace edit` CLI commands accept the `--environment` flag and persist it correctly.

---

### Test Steps

1. **Add workspace with --environment**
   - Input: `agh workspace add /tmp/test --environment daytona-dev`
   - **Expected:** Workspace created with `environment_ref: "daytona-dev"`

2. **Edit workspace environment**
   - Input: `agh workspace edit <id> --environment staging`
   - **Expected:** Workspace updated with new environment_ref

3. **Add workspace without --environment**
   - Input: `agh workspace add /tmp/test2`
   - **Expected:** Workspace created with empty environment_ref
