## TC-INT-013: Non-loopback HTTP mutation restriction and operator messaging

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/general`, `/settings/hooks-extensions`
**Traceability:** `task_14`, ADR-004, TechSpec > Transport and security policy

---

### Objective

Verify that settings reads remain available when HTTP is bound beyond loopback, while settings mutations and HTTP extension mutations are blocked with explicit operator-facing messaging instead of silent failure.

---

### Preconditions

- [ ] Start AGH with HTTP bound to a non-loopback host.
- [ ] The web UI remains reachable over that non-loopback bind.
- [ ] A privileged local fallback path still exists for validation, such as UDS or CLI.

---

### Test Steps

1. Open `/settings/general` in the non-loopback-bound environment.
   - **Expected:** The page loads and shows read-only data normally.

2. Attempt to save a reversible General setting change.
   - **Expected:** The mutation is rejected, no false success is shown, and the operator sees a clear restriction message tied to non-loopback HTTP mutation policy.

3. Open `/settings/hooks-extensions`.
   - **Expected:** The page loads read-only data and any mutation-sensitive controls expose the current transport limitation clearly.

4. Attempt an extension enable/disable action or policy save over HTTP.
   - **Expected:** The action is blocked with explicit `403`-class messaging or disabled-state explanation that directs the operator to loopback or UDS/CLI.

5. Verify the privileged local fallback path.
   - **Expected:** UDS or CLI remains the authoritative path for local mutations, confirming the policy is transport-specific rather than a broken backend.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Non-loopback bind | Example `0.0.0.0` or host alias | Exact host chosen by executor |
| Routes | `/settings/general`, `/settings/hooks-extensions` | Covers both settings and extension mutation policy |

---

### Post-conditions

- Revert the environment to the normal loopback bind before continuing with positive mutation cases.
- Capture a screenshot of the restriction message or disabled-state explanation.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Read-only navigation while mutations are blocked | Navigate across multiple settings routes | Reads continue to work; only mutations are restricted |
| Mutation control disabled before click | Hooks/extension control already disabled | UI still explains why editing is unavailable |
| CLI/UDS unavailable unexpectedly | No local fallback path | Treat as an environment blocker, not a product pass |

---

### Related Test Cases

- `TC-FUNC-002` validates the positive restart/mutation path on loopback.
- `TC-FUNC-012` validates the positive Hooks & Extensions path on loopback.

---

### Notes

- This case must not be skipped in release validation because ADR-004 is a product policy, not just an implementation detail.
