## TC-INT-008: CLI workspace add --sandbox flag

**Priority:** P2 (Medium)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that the `agh workspace add` and `agh workspace edit` CLI commands accept the `--sandbox` flag and persist it correctly.

---

### Test Steps

1. **Add workspace with --sandbox**
   - Input: `agh workspace add /tmp/test --sandbox daytona-dev`
   - **Expected:** Workspace created with `sandbox_ref: "daytona-dev"`

2. **Edit workspace sandbox**
   - Input: `agh workspace edit <id> --sandbox staging`
   - **Expected:** Workspace updated with new sandbox_ref

3. **Add workspace without --sandbox**
   - Input: `agh workspace add /tmp/test2`
   - **Expected:** Workspace created with empty sandbox_ref
