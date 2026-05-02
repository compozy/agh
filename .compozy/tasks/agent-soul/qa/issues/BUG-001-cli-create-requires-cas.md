# BUG-001: CLI Managed Authoring Create Requires CAS Flag

## Status

Fixed

## Severity

High

## Priority

P0

## Originating Cases

- `TC-SCEN-001`: Managed Soul Authoring - Operator Can Trust Persona State
- `TC-SCEN-002`: Heartbeat Policy, Session Health, And Advisory Wake (same CLI write boundary pattern)

## Confirmed Failure

- Timestamp: 2026-05-02
- Lab: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-164846-213590-lab`
- Evidence: `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-soul-write.json`
- Command:

```bash
agh agent soul write reviewer --file <scenario>/reviewer-SOUL.md --workspace agent-soul --json
```

Observed result:

```text
error: cli: --expected-digest is required
```

## Expected Behavior

`SOUL.md` create semantics allow `put` to omit `expected_digest` when the target file is absent. `HEARTBEAT.md` uses the same service-level missing-file CAS behavior, with an empty `expected_digest` representing "create only if absent".

## Root Cause

The Cobra write commands call `changedStringFlag` for `--expected-digest` / `--if-match`, which rejects omitted flags before the request reaches the managed authoring service. The service already distinguishes absent files from updates and rejects missing digests only when replacing an existing file.

## Fix Plan

- Allow Soul and Heartbeat write commands to pass an empty expected digest when the CAS flag is omitted.
- Keep delete, rollback, and refresh commands on the existing required-CAS path.
- Add CLI regression tests for create-without-CAS on Soul and Heartbeat.

## Verification

- `CGO_ENABLED=1 go test -race ./internal/cli -run 'TestAgent(Soul|Heartbeat)Commands|TestCLIAgentAuthoredContextIntegration' -count=1`
- `.compozy/tasks/agent-soul/qa/evidence/TC-SCEN-001-soul-write-fixed.json`
- `.compozy/tasks/agent-soul/qa/evidence/BUG-001-cli-regression-go-test.log`
- `.compozy/tasks/agent-soul/qa/evidence/final-make-verify.log`
