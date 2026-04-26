## TC-AUTO-005: Agent Self And Channel Verbs

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify `agh me`, `agh me context`, and `agh ch` commands work over validated agent identity,
support stable machine-readable output, preserve coordination metadata, and keep operator network
commands intact.

### Traceability

- Task: task_06, Agent Self And Channel Verbs.
- TechSpec: Agent Kernel CLI, Task-Channel Coordination Contract, API Endpoints.
- ADR: ADR-002, ADR-007, ADR-010, ADR-012.
- Resource lesson: Multica inbox references treat inbox/channel updates as notifications, not durable task authority.
- Surfaces: `internal/api/udsapi`, `internal/cli`, `internal/network`, channel DTOs.

### Preconditions

- Active managed session with valid `AGH_SESSION_ID` and `AGH_AGENT`.
- At least one channel visible to the session.
- Optional queued inbox message for reply-by-message coverage.

### Test Steps

1. Run `agh me -o json` and `agh me context -o json`.
   - **Expected:** Commands return stable JSON with caller identity, workspace/session facts, active channels, and context sections.

2. Run `agh ch list -o json`.
   - **Expected:** Visible channels are listed with discovery metadata scoped to the caller session/workspace.

3. Send `status`, `request`, `blocker`, `handoff`, `result`, and `review_request` messages with task/run/channel correlation metadata.
   - **Expected:** Each MVP kind is accepted, metadata is preserved, and task run ownership/status remains unchanged.

4. Attempt `agh ch send` with raw `claim_token` in body or metadata extension.
   - **Expected:** Command fails with structured error and the token value is not echoed in logs or output.

5. Use `agh ch recv --wait -o jsonl` and `agh ch reply --to-message`.
   - **Expected:** Receive emits one valid JSON object per line and reply inherits or accepts explicit correlation metadata.

6. Run existing operator `agh network ...` command path.
   - **Expected:** Operator network commands still require explicit flags and continue to work.

### Evidence To Capture

- `qa/logs/TC-AUTO-005/me.json`
- `qa/logs/TC-AUTO-005/ch-list.json`
- `qa/logs/TC-AUTO-005/ch-message-kinds.jsonl`
- `qa/logs/TC-AUTO-005/ch-token-rejection.log`
- `qa/logs/TC-AUTO-005/operator-network-regression.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Missing channel | `agh ch send` without channel | Structured validation error |
| Invalid message kind | `--kind done` | Rejected; only MVP kinds accepted |
| JSONL stream | `recv --wait -o jsonl` | One object per line, no human prefix |
| Reply outside inbox context | explicit correlation flags | Reply succeeds with explicit metadata |

### Related Test Cases

- TC-AUTO-014: Channel messages cannot mutate task state.
- TC-AUTO-016: CLI reference pages match implemented flags.
