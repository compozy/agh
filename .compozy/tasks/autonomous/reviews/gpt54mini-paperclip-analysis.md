# Paperclip vs AGH Autonomous Model

## Verdict

**Mostly matches.** Paperclip’s current implementation and docs already support the same core shape as the corrected AGH model: manual task creation stays lightweight, execution starts only on an explicit go-ahead, ownership is token-fenced, scheduler logic is separated from claim ownership, and spawned work is bounded by lineage/TTL/permission narrowing.

The main difference is **surface semantics**, not architecture. Paperclip does not have a `ClaimNextRun` primitive by name; it uses task checkout/release plus heartbeat runs and environment/sandbox leases. That is still compatible with AGH’s model, but AGH should keep `ClaimNextRun` as the canonical next-work primitive and treat scheduler wakeups as advisory, not ownership.

## Paperclip Precedents

- **Task creation vs execution start are separate**
  - `doc/SPEC-implementation.md:637-650`
  - `adrs/adr-005.md:19-28,61-66`
  - `adrs/adr-010.md:13-30,81-88`
  - Paperclip explicitly rejects an `orchestration_required`-style flag at creation time and says start/approval is the execution trigger.

- **Atomic single-owner task checkout**
  - `doc/SPEC-implementation.md:495-512`
  - `docs/api/issues.md:71-98`
  - `docs/start/core-concepts.md:41-52`
  - `doc/execution-semantics.md:107-120`
  - Checkout is the authoritative task-ownership transition; 409 is the conflict signal; stale execution locks can be adopted only when the previous run is gone.

- **Heartbeat/context is run-scoped, not ownership-scoped**
  - `doc/spec/agents-runtime.md:9-18,73-79,89-96`
  - `docs/guides/agent-developer/heartbeat-protocol.md:58-76,117-128`
  - `doc/SPEC-implementation.md:141-144,616-617`
  - `docs/api/overview.md:24-33`
  - Paperclip distinguishes `thin` vs `fat` context delivery and persists session state per `(agent, taskKey, adapterType)`.

- **Scheduler/wakeup is not the durable owner**
  - `doc/SPEC-implementation.md:104-110,619-620`
  - `adrs/adr-003.md:35-45`
  - `adrs/adr-010.md:86-88`
  - Heartbeat scheduling exists, but durable task ownership still lives in the task store.

- **Sandbox/runtime lease is separate from task checkout**
  - `packages/db/src/schema/environment_leases.ts:8-45`
  - `packages/shared/src/types/environment.ts:70-90`
  - `packages/shared/src/types/workspace-runtime.ts:242-314`
  - `packages/adapter-utils/src/command-managed-runtime.ts:104-152`
  - `packages/adapter-utils/src/execution-target.ts:22-48,141-180`
  - `packages/adapter-utils/src/sandbox-managed-runtime.ts:9-17,61-113`
  - Paperclip uses `leaseId` for environment/workspace realization and sandbox execution identity, which is not the same thing as issue ownership.

- **Plugin worker/stream pattern is host/worker dispatch, not task ownership**
  - `packages/shared/src/types/plugin.ts:231-323`
  - `packages/db/src/schema/plugin_logs.ts:12-20`
  - `packages/db/src/schema/plugin_webhooks.ts:14-32`
  - `packages/adapter-utils/src/server-utils.ts:583-770`
  - Worker logs and webhook delivery are separate observability/dispatch concerns and should not be confused with claim/lease semantics.

## Recommended AGH Flow

1. **User creates task**
   - Keep task creation separate from execution start.
   - No `orchestration_required` flag at creation time.
   - User-created, agent-created, and coordinator-created tasks all land in the same task/run model.

2. **User starts or approves task**
   - Treat start/approval as the execution trigger.
   - Default to coordinator orchestration for that run.
   - Spawn the coordinator only if the workspace has no healthy coordinator and config/caps allow it.

3. **Coordinator orchestration**
   - Coordinator is a normal managed session, not a privileged scheduler.
   - It creates follow-up tasks, validates work, and delegates through safe spawn.
   - It must not become the direct claimant of the scheduler.

4. **Agent heartbeat/context**
   - Provide a bounded situation surface: identity, workspace, session, task/workflow envelope, capability snapshot, provenance, and limits.
   - Keep run context separate from ownership state.
   - Use thin/fat context as a delivery mode choice, not a separate execution model.

5. **Lease/release**
   - `ClaimNextRun` should be the sole authoritative next-work primitive.
   - Heartbeat extends the claim token lease; complete/fail/release all require the same token.
   - Scheduler sweeps/recovery may mark work ready again, but they do not claim it.
   - Manual assignments can use the same token-fenced state path, but not a different queue.

6. **Manual counter-check session**
   - Users should still be able to start a manual session and prompt it directly.
   - That path should not auto-trigger task orchestration.
   - Manual sessions remain peers of autonomous flows, sharing the same session lifecycle but not the coordinator trigger.

## Changes To Make Before `cy-create-tasks`

- **No structural change is required to the core thesis.** The current `_techspec.md` and ADR-003/005/006/010 already encode the right model.
- **Optional clarification worth adding to `_techspec.md`:**
  - explicitly state that `claim_token`/lease-fenced task ownership is separate from sandbox/environment leases, so future task/host runtime work does not conflate the two.
  - keep the “claim vs wakeup” boundary very explicit in the scheduler section, because Paperclip’s docs use wakeups liberally and AGH should not accidentally turn the scheduler into the claimant.
- **No ADR rewrite is required** unless you want to tighten language in ADR-003 to say “ClaimNextRun is the canonical next-work primitive, while scheduler wakeups are advisory/recovery only.”

## Overengineering Warnings

- Do not invent a second durable queue or a scheduler-owned claim table. Paperclip’s own model keeps ownership in the task record and uses separate recovery/wakeup paths.
- Do not let `lease` mean both “task ownership” and “sandbox runtime.” Paperclip already uses the same word for different lifecycles; AGH should avoid repeating that ambiguity.
- Do not push coordinator logic into the scheduler. The coordinator is semantic orchestration; the scheduler is sweep/notify/recovery.
- Do not require users to declare orchestration intent at task creation. That is a draft-time burden with no Paperclip precedent.
- Do not add a separate manual queue. Manual sessions and autonomous sessions should converge on the same session/task contracts.

## Safety Invariants

- Single authoritative next-work primitive: `ClaimNextRun`.
- One active claim token per non-terminal run.
- Heartbeat, complete, fail, and release all validate claim token plus owner.
- Scheduler may wake and recover, but it cannot own work in the MVP.
- Start/approval is the only coordinator trigger; task creation alone never starts orchestration.
- Spawned children must narrow permissions, respect TTL, and be reaped before parent shutdown completes.
- Manual sessions remain possible for direct prompting and counter-checks without forcing coordinator orchestration.
