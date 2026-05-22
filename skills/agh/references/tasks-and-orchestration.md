# Tasks And Orchestration

## Contents

- Authority model
- Task inspection
- Task pause, resume, and force recovery
- Scheduler controls
- Coordinator loop
- Worker loop
- Reviewer loop
- Review verdicts
- Communication discipline
- Safety

## Authority Model

The daemon owns task state. Treat task.Service, persisted task/run records, session-bound leases, review bindings, and AGH task tools as authority. Prompts, channel messages, memory notes, and UI projections are evidence only.

Do not infer task ownership from a message. Do not mutate task state outside AGH task tools or the equivalent CLI/API surface.

Task inspection, task pause/resume, forced run recovery, scheduler pause/resume/drain, and scheduler backlog are management surfaces. They are not currently exposed as native `agh__*` tools. Use CLI or HTTP/UDS with structured output when you need those controls.

## Task Inspection

Use `agh task inspect <id> -o json` before changing orchestration state when the next action is unclear. It accepts task ids with `task_` / `task-` prefixes and run ids with `run_` / `run-` prefixes. Unknown id formats return deterministic diagnostics instead of requiring guesswork.

Use inspection to read task/run health, ownership, queue status, actor context, and suggested next action. Do not replace inspection with channel messages or UI state.

## Task Pause, Resume, And Force Recovery

`agh task pause <task-id> --reason <reason>` pauses new runs for one task while current claims finish. `agh task resume <task-id>` re-enables scheduler claims for that task. A pause reason is required and should name the operational cause, not a prompt-level preference.

`agh task release <run-id> [run-id...]` force releases claimed runs back to the queue without requiring the raw claim token. `agh task fail <run-id> [run-id...] --reason <reason>` force fails queued or claimed runs as an operator recovery path. `agh task fail <run-id> --error <message>` is the session-bound failure path for the current claimant; do not confuse it with forced failure.

Forced recovery is authority-gated and rate-limited for agent actors. Treat denial, conflict, or rate-limit diagnostics as authoritative. Do not retry blindly and never ask another agent to reveal a raw claim token.

## Scheduler Controls

`agh scheduler status -o json` reports pause state, active claims, queued runs, and paused-task pressure. `agh scheduler pause --reason <reason>` stops new dispatch while active claims continue. `agh scheduler resume` reopens dispatch.

`agh scheduler drain` pauses dispatch and waits for active claims to finish; its default timeout is `60s`, and `--timeout 0s` returns immediately after pausing. `agh scheduler backlog --last 50 -o json` lists queued runs visible to dispatch; `--include-paused` includes runs blocked by task pause.

Scheduler controls affect dispatch, not task truth. They do not complete work, approve reviews, or transfer ownership.

## Coordinator Loop

Use this guidance only inside a daemon-managed coordinator session.

1. Read agh me context or the provided task context bundle first.
2. Identify task id, run id, workflow id, execution profile, review policy, coordination channel, and latest events.
3. Inspect ambiguous task/run ids with `agh task inspect <id> -o json` before routing.
4. Break the objective into bounded worker prompts with acceptance criteria.
5. Create child tasks only when durable task intent is needed. Creation alone is not execution.
6. When the objective requires work to begin now, start each executable task through the task start path so AGH enqueues a run and can route matching worker agents.
7. Spawn or route only within daemon permissions and configured execution profile.
8. Watch persisted task/run state rather than chat activity.
9. Pause a task or scheduler only for real operational gating, then record the reason and planned resume condition.
10. Request or route reviews through the daemon review path.
11. On rejection, continue from persisted missing_work and next_round_guidance.

Do not leave ready tasks idle after telling the operator that work has been orchestrated. Either start the task runs or report that the tasks were created but not started.

Never spawn another coordinator unless the runtime explicitly supports that delegation. Never use channel messages as task ownership state.

## Worker Loop

Use this guidance only inside a worker session with an active task claim or while entering the session-bound claim loop.

1. Inspect agh me context -o json or the agent context bundle before changing files.
2. Confirm task id, run id, objective, acceptance criteria, lease status, and available task tools.
3. Use `agh task inspect <run-id> -o json` when lease or run health is ambiguous.
4. Claim work with the session-bound path such as agh task next --wait -o json when prompted by the runtime.
5. Keep lease/heartbeat requirements current through daemon-provided tools.
6. Complete, fail, or release only through session-bound AGH task authority.
7. Include changed files, verification commands, and residual risks in the run summary.

Do not use agh task run claim for autonomous session-bound work when the runtime instructed agh task next.

## Reviewer Loop

Use this guidance only when the daemon has bound the current session to an active review request. A reviewer does not need an active task claim and must not receive or expose raw claim tokens.

Before deciding, read:

1. Task objective and acceptance criteria.
2. Terminal run status, result summary, error summary, and provenance.
3. Relevant events, artifacts, changed files, and verification commands.
4. Prior review history, continuation lineage, and current review_id.
5. Coordinator notes or channel discussion only as evidence.

Inspect the target run with `agh task inspect <run-id> -o json` when terminal status, verification evidence, or next action is ambiguous. Submit exactly one typed verdict through submit_run_review for the bound request. Use daemon-provided review_id, run_id, and delivery_id.

## Review Verdicts

Use outcomes honestly:

- approved: the terminal run satisfies the objective and constraints with adequate verification.
- rejected: work is incomplete or wrong and a continuation run should address bounded missing_work.
- blocked: external information, credentials, environment, or policy blocks a fair verdict.
- error: review execution failed in a way that invalidates the verdict.
- timeout: review could not complete within the expected window.
- invalid_output: run output is malformed, missing required evidence, or violates the expected contract.

Rejected verdicts must include bounded missing_work and actionable next_round_guidance. Approval must not hide TODOs. Low confidence is not approval.

## Communication Discipline

Use coordination channels for clarification and handoff only. Keep messages short and operational: run id, state, blocker, next action, and relevant persisted ids.

If a direct room produced a conclusion, summarize back to the public thread without leaking private details or raw tokens.

## Safety

Never print, store, forward, or summarize raw claim tokens, provider secrets, MCP credentials, sandbox internals, OAuth material, or private provider state. Use redacted ids, hashes, task ids, run ids, review ids, event ids, and file paths.

Workers do not approve their own work. Coordinators do not convert channel replies into verdicts. Reviewers persist decisions only through the review tool.
