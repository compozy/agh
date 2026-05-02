## TC-REG-004: Revision Rollback, Delete, Restart Recovery, And Redacted Output

**Priority:** P1
**Type:** Regression
**Status:** Passed
**Estimated Time:** 40 minutes
**Created:** 2026-05-02
**Last Updated:** 2026-05-02

---

### Objective

Verify Soul and Heartbeat managed authoring remains trustworthy across multi-revision edits, rollback, delete, daemon restart, stale CAS disruption, and operator-facing redaction boundaries.

---

### Preconditions

- [ ] Bootstrap manifest exists and daemon/API readiness is confirmed.
- [ ] Scenario workspace contains `reviewer` and `ops` agents.
- [ ] Current Soul and Heartbeat digests are known from inspect responses.
- [ ] Daemon can be stopped and restarted using the isolated bootstrap env without touching the user's default AGH runtime.

---

### Test Steps

1. **Create two Soul revisions and rollback to the first**
   - Input: write Soul v1, write Soul v2 with the v1 digest, then `agh agent soul rollback reviewer --revision-id <v1-revision> --expected-digest <v2-digest> --workspace <workspace> --json`.
   - **Expected:** Rollback succeeds, inspect shows the v1 content digest/projection, history contains write/write/rollback entries, and stale rollback with `sha256:stale` fails without mutation.

2. **Delete Soul with stale then fresh CAS**
   - Input: `agh agent soul delete reviewer --expected-digest sha256:stale ...`; then repeat with the current digest.
   - **Expected:** Stale delete returns `soul_conflict`; fresh delete succeeds, inspect reports `present=false` or inactive deterministic state, and history records the delete revision.

3. **Create two Heartbeat revisions and rollback**
   - Input: write Heartbeat v1, write Heartbeat v2 with current digest, then `agh agent heartbeat rollback ops --revision-id <v1-revision> --if-match <v2-digest> --workspace <workspace> --json`.
   - **Expected:** Rollback succeeds, status/inspect show the v1 policy digest/config digest, and history records all revisions without creating a task run or wake queue row.

4. **Restart daemon and inspect recovered state**
   - Input: stop and restart the isolated daemon, then inspect Soul/Heartbeat/session health through CLI/API.
   - **Expected:** Persisted revision history and current state survive restart; session health is either recovered or reports an operator-readable inactive/stale state.

5. **Verify redaction and output bounds**
   - Input: scan CLI/API JSON/log evidence from the above steps for `$WORKSPACE_PATH`, `$AGH_HOME`, `claim_token`, `agh_claim_`, provider secret-like values, and unbounded raw prompt transcript text.
   - **Expected:** Operator-facing output uses workspace-relative source paths and does not leak absolute runtime paths, raw claim tokens, provider secrets, or full hidden prompt transcripts.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Soul v1 | Evidence-first launch reviewer | Used as rollback target |
| Soul v2 | Stricter rollback probe persona | Must be replaced by rollback |
| Heartbeat v1 | Min interval 30m, session health context | Used as rollback target |
| Heartbeat v2 | Min interval 45m, wake audit context | Must be replaced by rollback |

---

### Required Evidence

- `qa/evidence/TC-REG-004-soul-rollback.json`
- `qa/evidence/TC-REG-004-soul-delete.json`
- `qa/evidence/TC-REG-004-heartbeat-rollback.json`
- `qa/evidence/TC-REG-004-restart-recovery.log`
- `qa/evidence/TC-REG-004-redaction-scan.log`

---

### Pass Criteria

- Rollback/delete operations enforce CAS and persist revision history.
- Restart recovery preserves current authored-context state and operator-readable diagnostics.
- Heartbeat rollback does not create task ownership, task runs, task leases, or a wake queue.
- Redaction scan finds no raw token, secret, absolute source path, or full hidden prompt transcript in operator-facing evidence.

### Failure Criteria

- Any stale CAS mutation changes persisted state.
- Revision history is missing required write/delete/rollback entries.
- Restart loses current state or reports a raw internal error.
- Operator-facing output leaks raw sensitive or absolute path material.
