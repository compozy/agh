## TC-AUTO-018: Post-MVP Boundary And Non-Regression Scope

**Priority:** P1 (High)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify the autonomy MVP stays within the accepted local kernel boundary and does not accidentally
introduce broad post-MVP network, memory, eval/replay, dashboard, or MCP-tool scope while planning
and executing task_18.

### Traceability

- Task: cross-cutting boundary for tasks 01-16 and task_17/task_18.
- TechSpec: MVP boundary, Development Sequencing steps 11-15, Known Risks.
- ADR: ADR-001, ADR-007, ADR-008, ADR-011.
- Resource lesson: ADR/resource reviews explicitly defer broad network/memory/eval visibility until local autonomy proves the wire shape.
- Surfaces: docs, web routes, network protocol fields, memory scope docs, task_18 bug-fix scope.

### Preconditions

- Source tree available for search.
- Task_18 has not intentionally expanded scope with a new accepted TechSpec or user approval.

### Test Steps

1. Search docs and runtime pages for cross-daemon swarm, leader election, contract-net, multi-home, vote/react/escalate, eval/replay UI, and built-in MCP mirror claims.
   - **Expected:** Features are absent or explicitly marked out of scope/post-MVP.

2. Inspect web route/system additions during task_18.
   - **Expected:** No new coordinator dashboard, scheduler dashboard, spawn lineage tree, eval/replay UI, or coordinator config GUI appears unless a confirmed bug fix requires a narrow existing-surface change.

3. Inspect network/channel changes during task_18.
   - **Expected:** Message kinds stay within MVP kinds unless a new ADR/TechSpec is accepted.

4. Inspect memory behavior touched during task_18.
   - **Expected:** No broad peer/channel memory extraction or automatic per-turn promotion is added as a workaround for coordination evidence.

5. Record any future work discovered during QA as follow-up, not silent scope expansion.
   - **Expected:** Follow-ups are written as issue notes or task memory; P0 fixes stay inside MVP invariants.

### Evidence To Capture

- `qa/logs/TC-AUTO-018/post-mvp-scope-rg.log`
- `qa/logs/TC-AUTO-018/web-route-scope.log`
- `qa/logs/TC-AUTO-018/network-message-kind-scope.log`
- `qa/logs/TC-AUTO-018/follow-up-notes.md`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Docs mention swarm | docs page | Marked post-MVP/out of scope |
| Bug fix needs UI wording | existing Tasks UI | Narrow label fix allowed |
| New network kind appears | `offer`, `accept` | Requires accepted scope change |
| Memory summary follow-up | channel memory idea | Filed as follow-up, not implemented |

### Related Test Cases

- TC-AUTO-016: Docs stay MVP-only.
- TC-AUTO-015: No broad autonomy dashboard.
