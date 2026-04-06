# Child Workgroup Activation Fix

## Summary

- Root cause is confirmed from the latest session `d76v1johqf41hojcvpig`: the child master `impl-master` is spawned and reaches `agent_ready` in `resilience_manager`, but never reaches `agent_ready` in `workgroup_manager`, so the `impl` workgroup never transitions from `create` to `active`.
- The evidence is concrete: observability shows `workgroup_created` for `impl`, then `agent_spawned`, then only the resilience-side ready event; the matching workgroup-side ready event is missing. The transcript then shows `cannot spawn agent: workgroup impl is waiting for its master` at `2026-04-02 04:31:13 UTC`, and later `cannot spawn agent: workgroup execution is waiting for its master` at `2026-04-02 04:31:57 UTC`.
- There is a second coordination bug: the kernel currently allows creating a child workgroup under a parent still in `create`, even though the spec says a `create` workgroup only allows spawning its master. That is why `execution` could be created under the still-stuck `impl` workgroup and inherit the same failure mode.
- Reference pattern to follow: keep spawn/register/status transitions owned by one lifecycle path, like `.resources/claude-code/bridge/sessionRunner.ts` and `.resources/claude-code/utils/concurrentSessions.ts`. Do not split readiness across bootstrap-only code and runtime spawn code.

## Implementation Changes

- Introduce one shared post-start activation path for every agent start, used by both bootstrap and runtime spawn. The sequence must be:
  1. `driver.Start`
  2. `ResilienceManager.attachProcess`
  3. `WorkgroupManager.MarkAgentReady`
  4. publish `transport.ReadySubject(session.ID, agent.ID)` so the hook router activates the agent and flushes any queued hook traffic
  5. return the attached agent only after all four steps succeed
- Remove the current split behavior where bootstrap manually marks agents ready but runtime spawns stop after `attachProcess`. After this change, bootstrap and `spawnAgent` must both call the same activation helper so a future agent type cannot bypass the lifecycle handshake again.
- On any failure after `attachProcess` succeeds, stop and deregister the just-started agent before returning the error. Do not leave half-ready agents or half-created workgroups alive.
- Tighten workgroup creation rules so `agh workgroup create --parent <parent>` only succeeds when the parent workgroup is `active`. Reject `create`, `closing`, and `closed` parents using the existing kernel error style: `cannot create workgroup: parent workgroup <display> is <state>`.
- Preserve current parent-to-child direct messaging. The parent supervisor must still be able to send instructions to a child master while that child is in `create`; the only behavior change is that the child will reliably flip to `active` as soon as its master is actually ready.
- Do not change the session state schema or the global observability schema in this fix. The storage layer gave enough evidence; the defect is lifecycle coordination.

## Public / Behavioral Changes

- Spawning a master into a child workgroup will fully activate that child automatically once the spawned master is ready. No manual “poke the master to activate the workgroup” step remains.
- Creating a descendant workgroup under a parent still in `create` will now fail immediately instead of allowing nested `create` trees.
- Observability for spawned agents will now show the same paired readiness transition that bootstrap agents already show: resilience readiness followed by workgroup readiness.

## Test Plan

- Add a kernel lifecycle regression that creates a child workgroup, spawns a master into it through the normal spawn API, verifies the child becomes `active`, then verifies a worker can be spawned into that child successfully.
- In that same regression, assert the spawned master produces both readiness transitions, not only the resilience-side one.
- Add a workgroup guardrail test that creating a child under a `create` parent is rejected with the expected kernel error.
- Add a hook-router integration test that queues a hook event for a spawned master’s workgroup before activation, runs the shared activation path, and verifies the queued event is flushed after the ready subject is published.
- Run focused verification first with `go test ./internal/kernel -count=1`, then run `make verify`.

## Assumptions / Defaults

- Use the existing `transport.ReadySubject(...)` path to integrate with the hook router rather than storing a direct hook-router pointer on `Session`.
- Treat the dashboard bind errors in `~/.agh/logs/agh.log` as a separate runtime-startup issue; they are not required to resolve this coordination failure.
- Keep current error wording style and current CLI surface unless an existing test/spec already requires exact wording.
