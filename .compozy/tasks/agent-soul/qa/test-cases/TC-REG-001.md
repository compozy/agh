## TC-REG-001: Invalid Authored Context Fails Closed Without Cross-Feature Bleed

**Priority:** P0
**Type:** Regression
**Status:** Passed
**Estimated Time:** 30 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Objective

Verify invalid `SOUL.md` and invalid `HEARTBEAT.md` produce deterministic diagnostics and do not silently disable the other authored-context feature.

---

### Preconditions

- [ ] Bootstrap manifest exists and daemon/API readiness is confirmed.
- [ ] Scenario workspace contains `reviewer` and `ops` agents.
- [ ] Valid Soul and Heartbeat artifacts were already written through managed authoring or validated as preconditions.

---

### Test Steps

1. **Validate invalid Soul with forbidden operational authority**
   - Input: `agh agent soul validate reviewer --file <invalid-soul> --workspace <workspace> --json`
   - **Expected:** Validation fails with a deterministic Soul diagnostic such as `forbidden_field` or reserved-section equivalent.

2. **Attempt invalid Soul managed write**
   - Input: `agh agent soul write reviewer --file <invalid-soul> --workspace <workspace> --json`
   - **Expected:** Mutation is rejected, existing valid `SOUL.md` remains unchanged, and no partial revision replaces the valid body.

3. **Confirm Heartbeat remains diagnosable**
   - Input: `agh agent heartbeat inspect ops --workspace <workspace> --json`
   - **Expected:** Heartbeat inspect/status still returns the current policy or its own diagnostics; Soul failure does not hide Heartbeat state.

4. **Validate invalid Heartbeat with forbidden queue/task authority**
   - Input: `agh agent heartbeat validate ops --file <invalid-heartbeat> --workspace <workspace> --json`
   - **Expected:** Validation fails with deterministic Heartbeat diagnostics such as `heartbeat_forbidden_field` or `heartbeat_reserved_section`.

5. **Confirm Soul remains inspectable**
   - Input: `agh agent soul inspect reviewer --workspace <workspace> --json`
   - **Expected:** Soul inspect still reports the prior valid digest/read model; Heartbeat failure does not alter Soul state.

---

### Required Evidence

- `qa/evidence/TC-REG-001-invalid-soul.json`
- `qa/evidence/TC-REG-001-invalid-heartbeat.json`
- `qa/evidence/TC-REG-001-cross-feature.log`

---

### Pass Criteria

- Invalid authored content fails closed.
- The valid counterpart feature remains readable and independently diagnosable.
- Existing valid files are not partially overwritten after failed managed writes.
