# TC-SCEN-002: Session Snapshot, Sub-Agent Inheritance, And Forensic Ledger

**Priority:** P0
**Type:** Real Scenario
**Status:** Not Run
**Estimated Time:** 60 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Behavioral Scenario Charter

- Startup situation: isolated daemon with one root agent, one sub-agent, and workspace/global/agent memories.
- Operator intent: prove the session sees a frozen memory snapshot, sub-agents inherit read-only memory, reload affects next session only, and stopped sessions produce forensic ledgers.
- Expected business outcome: operators can reason about what memory a session saw and inspect it later without mutable forensic replay controls.
- AGH surfaces used: CLI, session API, web Session Inspector, filesystem ledger, prompt/session evidence.
- Real provider/LLM expectation: use a provider-backed session if credentials are available; otherwise use the project e2e harness and document the provider boundary.

## Preconditions

- [ ] Fresh isolated QA lab and web proxy target are available.
- [ ] Root agent allows memory writes; sub-agent write policy is denied/read-only.
- [ ] Workspace has at least one global, workspace, agent-global, and agent-workspace memory fixture.

## Journey Steps

1. **Start a root session**
   - Input: create/start a session for the root agent in the scenario workspace.
   - **Expected:** Prompt/session evidence includes a memory block built from the initial frozen snapshot.

2. **Write or edit memory mid-session**
   - Input: supported CLI or API memory write for the same selector.
   - **Expected:** Current session prompt evidence does not mutate; a new decision/event is persisted.

3. **Run `agh memory reload`**
   - Input: `agh memory reload --scope workspace -o json`
   - **Expected:** The current session remains unchanged; reload invalidates the next boot only.

4. **Start a second root session**
   - Input: create/start another session after reload.
   - **Expected:** New session sees the updated memory.

5. **Spawn or exercise a sub-agent**
   - Input: root agent delegates a bounded task to sub-agent.
   - **Expected:** Sub-agent receives inherited memory snapshot and write attempts fail closed or are absent by policy.

6. **Stop sessions and inspect ledgers**
   - Input: stop root and sub-agent sessions.
   - **Expected:** `ledger.jsonl` exists at `$AGH_HOME/sessions/<workspace_id>/<session_id>/ledger.jsonl`, contains lineage meta, and sub-agent ledger records `parent_session_id` or `spawn_parent_id`.

7. **Open web Session Inspector**
   - Input: browser route for the stopped session.
   - **Expected:** Memory panel shows ledger meta and redaction-safe event fields only; no editor, promote, replay, or arbitrary payload controls.

## Required Evidence

- Session IDs and workspace_id.
- Prompt/session transcript excerpts showing frozen vs next-session memory.
- CLI/API reload response.
- Sub-agent write denial or absent write tool evidence.
- `ledger.jsonl` path, checksum, and first ledger lines.
- Browser screenshot/DOM snapshot of Session Inspector memory panel.

## Pass Criteria

- Mid-session mutation does not alter the current snapshot.
- Reload affects only the next session boot.
- Sub-agent memory remains read-only/inherited.
- Session ledger is materialized and shown truthfully in web UI.

## Failure Criteria

- Current session prompt changes after write/reload.
- Sub-agent can directly mutate curated memory.
- Ledger is missing or UI exposes unsupported controls.

