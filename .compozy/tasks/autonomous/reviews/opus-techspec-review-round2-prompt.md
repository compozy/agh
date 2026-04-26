# Autonomous AGH TechSpec Review Round 2

You are reviewing the current autonomy TechSpec and ADR set before it is decomposed into implementation tasks.

## Goal

Find gaps, contradictions, unclear boundaries, overengineering, under-specification, and cross-document inconsistencies that could cause bad `cy-create-tasks` output or flawed implementation.

This is a review task, not an implementation task.

## Required Inputs To Read

Read these files first:

- `.compozy/tasks/autonomous/_techspec.md`
- `.compozy/tasks/autonomous/adrs/adr-001.md`
- `.compozy/tasks/autonomous/adrs/adr-002.md`
- `.compozy/tasks/autonomous/adrs/adr-003.md`
- `.compozy/tasks/autonomous/adrs/adr-004.md`
- `.compozy/tasks/autonomous/adrs/adr-005.md`
- `.compozy/tasks/autonomous/adrs/adr-006.md`
- `.compozy/tasks/autonomous/adrs/adr-007.md`
- `.compozy/tasks/autonomous/adrs/adr-008.md`
- `.compozy/tasks/autonomous/adrs/adr-009.md`
- `.compozy/tasks/autonomous/adrs/adr-010.md`
- `.compozy/tasks/autonomous/analysis/analysis.md`
- `.compozy/tasks/autonomous/reviews/opus-techspec-review.md`
- `.compozy/tasks/autonomous/reviews/gpt54mini-multica-analysis.md`
- `.compozy/tasks/autonomous/reviews/gpt54mini-paperclip-analysis.md`
- `.compozy/tasks/autonomous/reviews/gpt54mini-agh-code-analysis.md`

Then inspect AGH code only where needed to validate assumptions. The most relevant packages are:

- `internal/task`
- `internal/store/globaldb`
- `internal/session`
- `internal/daemon`
- `internal/api/contract`
- `internal/api/httpapi`
- `internal/api/udsapi`
- `internal/cli`
- `internal/hooks`
- `internal/resources`
- `internal/network`
- `internal/memory`
- `internal/skills`
- `internal/tools`

## Context And Decisions Already Made

Preserve these decisions unless you find a concrete inconsistency that makes them unworkable:

- Autonomy is additive. Users can still create tasks, start sessions, prompt sessions directly, and use manual counter-check flows.
- Task creation does not trigger coordinator startup and does not create claimable work by itself.
- Publish/start/approval or equivalent execution action enqueues a task run.
- The coordinator trigger is the coordinated task-run enqueue boundary, represented in current AGH by `task.run_enqueued`, not `task.created`.
- Coordinator-agent is a normal managed AGH session, configurable by provider/model through global config plus workspace overrides.
- Scheduler is daemon-owned sweep/notify/recovery only in the MVP. It must not directly claim runs.
- `ClaimNextRun(criteria)` is the sole authoritative next-work primitive.
- `task_runs` remains the durable ownership source. No separate durable scheduler queue.
- Task-run lease is separate from sandbox/workspace/environment leases.
- Safe spawn requires lineage, TTL, caps, auto-stop, and child permission narrowing.
- New autonomy behavior must respect existing hooks/resources extensibility and must not use ad-hoc callbacks or a generic event bus.
- MVP decomposition should focus on TechSpec steps 1-10; steps 11-15 are post-MVP unless explicitly pulled in later.

## Review Questions

Answer these concretely:

1. Are there contradictions between `_techspec.md` and any ADR?
2. Are there contradictions between the TechSpec/ADRs and the current AGH code?
3. Are any MVP steps ordered incorrectly for implementation?
4. Are any extension points missing or over-specified?
5. Are any safety invariants underspecified, especially around claim/lease, coordinator spawn, safe spawn, hooks, or manual flows?
6. Are any pieces likely to cause `cy-create-tasks` to generate tasks that are too broad, too vague, or impossible to implement independently?
7. Are there terms that need to be renamed or made consistent before task decomposition?
8. Is anything overengineered for the first task decomposition and should be pushed to post-MVP?
9. Is anything important missing that must be added before task decomposition?

## Output Format

Write the full review to:

`.compozy/tasks/autonomous/reviews/opus-techspec-review-round2.md`

Use this format:

```md
# Opus Review Round 2: Autonomous AGH TechSpec

## Verdict

Approve / Approve with changes / Block

Brief reason.

## Blocking Issues

List only issues that must be fixed before `cy-create-tasks`.
For each issue include:
- Severity
- Files/sections
- Why it matters
- Concrete fix

## Non-Blocking Issues

Same structure, but for improvements that can be done before or during task decomposition.

## Consistency Checks

- TechSpec vs ADRs
- TechSpec/ADRs vs current AGH code
- Review round 1 findings
- GPT-5.4 Mini findings
- Manual control model
- Coordinator trigger model
- Claim/lease model
- Hooks/resources extensibility model
- MVP/post-MVP boundary

## Recommended Edits

Specific edits, grouped by file.

## Things To Keep As-Is

List decisions that are sound and should not be reopened.
```

If you find no blockers, say that directly. Be strict: this review is meant to catch flaws before task decomposition.

Do not modify any file except the requested review output file.
