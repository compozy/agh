## TC-AUTO-004: Situation Context And Caller Identity

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify `/agent/context`, prompt situation rendering, and caller identity validation give agents the
runtime facts they need while rejecting stale, missing, mismatched, or unauthorized session identity.

### Traceability

- Task: task_04 Situation Surface Providers and task_05 Agent Caller Identity Layer.
- TechSpec: Situation Surface, Agent Kernel CLI, API Endpoints, Manual Control Contract.
- ADR: ADR-001, ADR-002, ADR-008, ADR-009, ADR-010, ADR-012.
- Resource lesson: Hermes context builder references require bounded sections with provenance, not shell snippets.
- Surfaces: `internal/situation`, `internal/session` prompt seams, `internal/agentidentity`, UDS identity helpers.

### Preconditions

- One active managed session with `AGH_SESSION_ID` and `AGH_AGENT`.
- Optional fixtures for active task, coordination channel, inbox messages, peers, capabilities, and limits.
- One stopped or unknown session ID for negative identity checks.

### Test Steps

1. Call `/agent/context` or `agh me context -o json` from a valid managed session.
   - **Expected:** Sections appear in stable order: `self`, `workspace`, `session`, `task`, `coordination_channel`, `inbox_summary`, `peer_roster`, `capabilities`, `limits`, `provenance`.

2. Seed lists larger than section bounds.
   - **Expected:** Each bounded section truncates deterministically and reports truncation metadata.

3. Remove optional services or context facts.
   - **Expected:** Missing sections are omitted cleanly; no fabricated placeholder facts appear.

4. Submit or assemble a prompt for the session.
   - **Expected:** Startup and dynamic prompt augmentation include bounded situation context without duplicating stale previous context.

5. Call agent endpoints with missing env, unknown session, stopped session, and mismatched `AGH_AGENT`.
   - **Expected:** Each invalid identity fails closed with structured errors and no task/channel/spawn operation executes.

### Evidence To Capture

- `qa/logs/TC-AUTO-004/agent-context.json`
- `qa/logs/TC-AUTO-004/context-truncation.json`
- `qa/logs/TC-AUTO-004/prompt-situation.log`
- `qa/logs/TC-AUTO-004/identity-negative-cases.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Active channel-bound run | claimed run has channel | Context includes active coordination channel |
| No network service | service unavailable | Context omits peer/inbox without panic |
| Stopped session env | stopped `AGH_SESSION_ID` | Agent endpoint returns stale identity error |
| Operator command with env | `agh task create --workspace ...` | Operator path remains explicit, not identity-inferred |

### Related Test Cases

- TC-AUTO-005: Agent self and channel CLI.
- TC-AUTO-008: Agent task lease API identity.
