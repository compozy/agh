# BUG-005: Spawn Lineage Lost Parent Soul Digest In Global Session Rows

**Severity:** High
**Priority:** P0
**Type:** Data
**Status:** Fixed

## Environment

- **Build:** current branch during `agent-soul-qa` continuation on 2026-05-02.
- **OS:** macOS local isolated QA lab.
- **Browser:** not in scope.
- **URL:** runtime session creation/spawn path.
- **Live provider/LLM:** reproduced with live Codex-backed parent/child sessions in the isolated lab.

## Summary

During TC-SCEN-004, a spawned child session was visible through session APIs, but the persisted `sessions` row did not store `parent_session_id` or `parent_soul_digest`. This broke the techspec requirement that child sessions retain provenance-only lineage to the parent Soul digest.

## Behavioral Impact

- **Operator/User Goal:** operators could not audit child-session provenance from durable global session state.
- **Agent Behavior:** spawned child sessions could appear detached from the parent Soul provenance even though the runtime created them from a parent session.
- **Business Outcome:** lineage and auditability for agent delegation were incomplete.
- **Cross-Surface State:** API/session payload and SQLite `sessions` persistence disagreed before the fix.

## Reproduction

```bash
AGH_HOME="$LAB/.agh/runtime" ./bin/agh session new --workspace agent-soul-lab --agent reviewer --provider codex -o json
AGH_HOME="$LAB/.agh/runtime" ./bin/agh session spawn <parent-session-id> --agent general --provider codex -o json
sqlite3 "$LAB/.agh/runtime/agh.db" 'select id,parent_session_id,parent_soul_digest from sessions where id in (...);'
```

Observed before the fix:

- Child session creation succeeded.
- The child row in `sessions` had empty lineage/provenance fields.

## Expected

The child row must store `parent_session_id` and `parent_soul_digest` as provenance metadata while leaving the child behavioral Soul empty unless the child agent has its own Soul.

## Root Cause

The hooks bridge registered global session rows from the hook payload's reduced `sessionFromHookPayload` representation. That representation does not carry full runtime lineage or Soul provenance fields. `hooksNotifier.OnSessionCreated` and `OnSessionStopped` did not forward the full runtime `*session.Session` to downstream lifecycle observers.

## Fix

- Updated `internal/daemon/hooks_bridge.go` so lifecycle notifications forward the full runtime session to downstream observers after hook dispatch.
- Added `TestHooksNotifierLifecycleForwarding` in `internal/daemon/notifier_test.go`.

## Verification

- Focused regression: `.compozy/tasks/agent-soul/qa/evidence/BUG-005-hooks-notifier-focused-go-test.log`.
- Live lab proof after rebuild/restart:
  - `.compozy/tasks/agent-soul/qa/evidence/BUG-005-parent-session-new-after-fix.json`
  - `.compozy/tasks/agent-soul/qa/evidence/BUG-005-child-spawn-after-fix.json`
  - `.compozy/tasks/agent-soul/qa/evidence/BUG-005-parent-child-sessions-sqlite-after-fix.json`

## Impact

- **Users Affected:** operators and agents relying on spawned-session provenance.
- **Frequency:** always for child sessions routed through the hook lifecycle observer path before the fix.
- **Workaround:** none.

## Related

- Test Case: TC-SCEN-004
