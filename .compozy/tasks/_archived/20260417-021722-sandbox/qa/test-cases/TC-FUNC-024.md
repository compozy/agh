## TC-FUNC-024: Reconciliation skips local backend sessions

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 07

---

### Objective

Verify that reconciliation skips sessions with `backend = local` since local sandboxes have no remote resources to reconcile.

---

### Preconditions

- [x] Mix of local and remote sessions in store

---

### Test Steps

1. **Boot with local and remote sessions**
   - **Expected:** Only remote backend sessions are processed by reconciliation. No provider calls made for local sessions.

2. **Verify local sessions untouched**
   - **Expected:** Local session metadata unchanged after reconciliation
