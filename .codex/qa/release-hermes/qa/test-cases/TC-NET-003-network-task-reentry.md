# TC-NET-003: Network-Origin Task Reentry

**Priority:** P0
**Type:** Integration / E2E
**Status:** Pass
**Created:** 2026-04-24

## Objective

Verify that network-originated detached work preserves ownership, channel metadata, task/run state, and re-enters the owning session after completion.

## Preconditions

- Daemon booted with task runtime and network runtime.
- A session is joined to a network channel.
- Detached task run uses network turn source metadata.

## Test Steps

1. Enqueue or trigger a network-owned task run.
   **Expected:** task origin identifies a network peer/channel and run is queued.

2. Claim and complete the detached run through the harness/runtime path.
   **Expected:** status transitions are persisted without duplicate claims.

3. Restart or recover the runtime where supported by existing e2e lane.
   **Expected:** unbound/running work is requeued or recovered according to task runtime rules.

4. Observe the owning session.
   **Expected:** synthetic network prompt is injected once with correct run/task/channel metadata.

## Execution History

| Date       | Tester | Build | Result | Notes                                                                                                                                                                                                                    |
| ---------- | ------ | ----- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-04-24 | Codex  | local | Pass   | `go test -race -tags integration ./internal/daemon -run TestDaemonNightlyE2EAutomationTaskResumesIntoNetworkChannel -count=1 -v` passed, and full `make test-integration` passed after provider-resolution/resume fixes. |
