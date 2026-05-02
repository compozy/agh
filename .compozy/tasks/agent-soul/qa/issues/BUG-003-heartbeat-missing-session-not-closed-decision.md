# BUG-003: Heartbeat Missing-Session Wake Returns Raw Error

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
- Evidence: `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-wake-missing-session.log`
- Command:

```bash
agh agent heartbeat wake ops --workspace agent-soul --session sess-missing --dry-run --json
```

Observed result:

```text
error: heartbeat: read session health for "sess-missing": session: session not found: sess-missing
```

## Expected Behavior

Heartbeat wake requests against missing sessions should return a deterministic wake decision with a closed reason such as `session_not_found`, rather than an unstructured transport error.

## Root Cause

The wake service maps `heartbeat.ErrSessionHealthNotFound` to `WakeReasonSessionNotFound`, but the daemon session manager reports missing sessions as `session.ErrSessionNotFound`. The daemon composition passed the session manager directly into the wake service, so the sentinel was not normalized at the package boundary.

## Fix Plan

- Wrap the session health reader used by the wake service in daemon composition.
- Translate `session.ErrSessionNotFound` into `heartbeat.ErrSessionHealthNotFound` for wake decisions only.
- Keep direct session health APIs returning their existing session-domain errors/status behavior.

## Verification

- `CGO_ENABLED=1 go test -race ./internal/daemon -run TestHeartbeatWakeHealthReader -count=1`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-wake-missing-session-fixed.json`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-heartbeat-wake-missing-session-api.json`
- `.compozy/tasks/agent-soul/qa/evidence/BUG-003-heartbeat-missing-session-go-test.log`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-002-final-heartbeat-wake-missing-session.log`
- `.compozy/tasks/agent-soul/qa/evidence/final-make-verify.log`
