## SMOKE-008: Daemon boot completes with sandbox registry

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16

---

### Objective

Verify daemon boot sequence successfully creates and wires the sandbox registry, including local and daytona providers, without blocking or erroring.

---

### Preconditions

- [x] Daemon boot code includes `buildSandboxRegistry`
- [x] Sandbox reconciliation step exists in `bootRuntime`

---

### Test Steps

1. **Boot daemon**
   - **Expected:** Daemon starts successfully. Environment registry created. Local and daytona providers registered. Reconciliation step runs (no-op when no prior sessions).

2. **Verify daemon status shows sandbox info**
   - **Expected:** Daemon status accessible, no environment-related errors in boot logs

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| No DAYTONA_API_KEY set | Missing env var | Daytona provider registered but will error on Prepare; boot does not fail |
| Previous crash with no remote sessions | Clean boot | Reconciliation is a no-op |
