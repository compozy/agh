## TC-INT-005: Shared Process Registry Recovery And Scoped Interrupts

**Priority:** P0 (Critical)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 70 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that the shared process registry records subprocess ownership durably, reconciles restart state using PID/start-time evidence, retires stale records without signaling unrelated processes, and interrupts only the requested session/turn/tool/process scope.

### Traceability

- Task: task_06, Tool Process Registry and Interrupts.
- TechSpec: issues 29 and 30; Testing Approach checkpointing, PID reuse detection, boot reconciliation, interrupt scoping.
- ADR: ADR-004 shared process registry and interrupt runtime.
- Surfaces: `internal/toolruntime`, `internal/acp`, ACP terminals, environment terminals, hooks, extensions, shared subprocess helpers, `internal/procutil`, site operations docs.

### Preconditions

- Isolated global DB for `tool_processes`.
- Test owners for ACP agent process, ACP terminal, local sandbox terminal, remote sandbox terminal without PID evidence, hook subprocess, extension process, and shared subprocess helper.
- Controlled process fixtures expose PID and start-time evidence.

### Test Steps

1. Start representative subprocess owners through their normal runtime paths.
   - **Expected:** Each owner writes a checkpoint record with bounded command metadata, owner IDs, state, daemon PID, PID/start-time evidence when local, and source type.

2. Update and complete selected process records.
   - **Expected:** Registry checkpoints state transitions on update/completion and no owner keeps an incompatible private registry for selected paths.

3. Restart/reconcile with a live local PID whose start time still matches.
   - **Expected:** Record remains active and can be targeted for scoped interruption.

4. Restart/reconcile with a reused PID or mismatched start-time evidence.
   - **Expected:** Record is marked stale and no signal is sent to that PID.

5. Restart/reconcile a remote terminal record without local PID evidence.
   - **Expected:** Record is retired stale rather than signaled by PID.

6. Interrupt by session/turn/tool, terminal ID, hook, extension, and direct process scope.
   - **Expected:** Only matching records transition through interrupt states and unrelated session/process work is untouched.

7. Cancel an active prompt with tool processes.
   - **Expected:** AGH sends cooperative ACP cancel first, then interrupts only registered tool processes owned by the active session turn.

8. Review site docs.
   - **Expected:** Operations daemon docs explain process recovery, stale PID safety, remote records, and scoped interrupts.

### Evidence To Capture

- `qa/logs/TC-INT-005/go-test-toolruntime.log`
- `qa/logs/TC-INT-005/process-records-before-restart.json`
- `qa/logs/TC-INT-005/process-reconciliation-report.json`
- `qa/logs/TC-INT-005/interrupt-report.json`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| PID reuse | Same PID, different start time | Mark stale, do not signal |
| Remote terminal | No local PID evidence | Retire stale after restart |
| Broad session interrupt | Session ID only | Signals only matching session-owned records |
| Tool-call interrupt | Session/turn/tool IDs | Signals only that tool call |

### Related Test Cases

- TC-INT-003: Prompt cancellation and failure state must stay coherent.
- TC-REG-002: Site docs cover restart and interrupt semantics.
