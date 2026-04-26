# Multica Alignment Review

## Verdict

The current corrected model matches Multica on the important boundaries:

- task creation is distinct from execution start,
- creation does not need an `orchestration_required` flag,
- coordinator orchestration is triggered by start/approval, not draft creation,
- the scheduler is a wake/recovery layer, not the worker-run claimant,
- manual prompting and manual sessions remain first-class.

The only real gap is wording clarity in `_techspec.md`: one data-flow sentence can still read as if “creating ready work” and “starting execution” are the same step. Multica is stricter than that.

## Multica precedents

- Issue/task creation is separate from later task enqueueing: `server/internal/handler/issue.go:922-927` enqueues only after issue creation, and `server/internal/handler/issue.go:1119-1144` enqueues/cancels on later issue updates.
- Comment-driven work also creates a task after the message exists: `server/internal/handler/comment.go:231-247`.
- Runtime-owned work is pulled, not pushed by the scheduler: `server/internal/handler/daemon.go:662-700` claims by runtime, then `server/internal/handler/daemon.go:930-947` starts the task as a separate step.
- The task service keeps claim/start/complete separate: `server/internal/service/task.go:174-237`, `330-463`.
- The SQL layer reinforces that split: `server/pkg/db/queries/agent.sql:116-152` has `ClaimAgentTask`, `StartAgentTask`, `CompleteAgentTask`; `190-222` adds session pinning and sweep recovery.
- The scheduler/sweeper does wake/recovery, not worker ownership: `server/cmd/server/autopilot_scheduler.go:16-31,71-106` and `server/cmd/server/runtime_sweeper.go:34-49,54-95,129-149`.
- Autopilot uses task enqueueing as a consequence of run dispatch, not as a creation-time flag: `server/internal/service/autopilot.go:34-93,96-175,177-216`.
- Protocol events separate lifecycle stages cleanly: `server/pkg/protocol/events.go:25-32,61-64,82-90` and payloads in `server/pkg/protocol/messages.go:11-32,60-87`.

## Recommended AGH flow

- Create task: persist intent only. No orchestration flag at creation.
- Start/approve task: enqueue the run and let that action trigger coordinator spawn by default.
- Coordinator trigger: only on start/approval of coordinated execution, idempotent per workspace.
- Worker claim: the owning session pulls work through `ClaimNextRun`; the scheduler only nudges/wakes and recovers expired leases.
- Complete/fail/release: require the claim token and lease fencing on every terminal transition.
- Manual control: keep direct session prompting and counter-check sessions available without routing them through the coordinator.

## Changes to make before `cy-create-tasks`

1. Tighten `_techspec.md:53-62` so step 1 distinguishes “task creation” from “run enqueue/start.” The run is what becomes claimable.
2. Keep `_techspec.md:74-78` and `:80-89` as the authoritative claim/coordinator boundary; they already match Multica and should stay explicit.
3. No ADR rewrite looks necessary. `adr-003`, `adr-004`, `adr-005`, and `adr-010` are already aligned; they only need to remain consistent with the clarified wording in the tech spec.

## Overengineering warnings

- Do not add a second durable scheduler queue. Multica keeps scheduler state separate from worker ownership, but not as a second source of truth.
- Do not add an `orchestration_required` task-creation flag. It pushes execution policy into drafting and makes user intent harder to change later.
- Do not let the scheduler become the direct claimant for worker runs. That creates a second ownership path and weakens the `ClaimNextRun` contract.
- Do not merge manual prompting with claim/lease mechanics. Manual sessions are control surfaces, not ownership transitions.

## Safety invariants

- One durable ownership source for runs.
- One authoritative next-work primitive.
- Claim, heartbeat, complete, fail, and release all fence on the same claim token.
- Boot recovery runs before wake/claim traffic.
- Coordinator spawn is idempotent per workspace and never self-recurses.
- Creation never auto-starts execution; start/approval does.
