# BUG-002: Heartbeat Dry-Run Returns Synthetic Prompt Identifiers

## Status

Fixed

## Severity

High

## Priority

P0

## Originating Case

- `TC-SCEN-002`: Heartbeat Policy, Session Health, And Advisory Wake

## Confirmed Failure

- Timestamp: 2026-05-02
- Lab: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-164846-213590-lab`
- Evidence:
  - `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-wake-dry-run.json`
  - `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-status-after-dry-run.json`
- Command:

```bash
agh agent heartbeat wake ops --workspace agent-soul --session sess-ef05dae653635e1b --dry-run --json
```

Observed result included `result: "sent"` and `synthetic_prompt_id`, even though the follow-up status evidence showed no persisted wake state or wake events.

## Expected Behavior

Dry-run wake evaluation must not imply that a prompt/event was created. It may report that the wake gates would allow a send, but it must not return identifiers for nonexistent wake events or synthetic prompts.

## Root Cause

`ManagedWakeService.newDecision` always preallocates a wake event id and assigns a synthetic prompt id for `WakeResultSent`. Dry-run paths return that decision before prompt dispatch or persistence, leaving generated identifiers in the operator response even though no event or prompt exists.

## Fix Plan

- Strip wake event and synthetic prompt identifiers from dry-run decisions before returning them.
- Add a Heartbeat wake-service regression test proving dry-run does not call the prompter, append events, update wake state, or expose nonexistent ids.

## Verification

- `CGO_ENABLED=1 go test -race ./internal/heartbeat -run TestManagedWakeServiceDecision -count=1`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-wake-dry-run-fixed-final.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-status-after-fixed-dry-run-final.json`
- `.compozy/tasks/agent-soul/qa/evidence/BUG-002-heartbeat-dry-run-go-test.log`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-wake-dry-run.json`
- `.compozy/tasks/agent-soul/qa/evidence/final-make-verify.log`
