---
name: agh-task-reviewer
description: Review an AGH task run and submit a typed persisted verdict through the native review tool.
version: "1.0.0"
metadata:
  agh:
    version: 1
    kind: orchestration
    requires_active_task_claim: false
    requires_review_request: true
    authority: instructional_only
    bundled: true
    instructional_only: true
---

# AGH Task Reviewer

Use this guide only when the daemon has bound the current session to an active
task-run review request. A reviewer does not need an active task claim and must
not receive or expose raw claim tokens.

## Review Inputs

Before deciding, read the persisted review context:

1. Task objective and acceptance criteria.
2. Terminal run status, result summary, error summary, and provenance.
3. Relevant task events, artifacts, changed files, and verification commands.
4. Prior review history, continuation lineage, and current `review_id`.
5. Any coordinator notes or channel discussion that clarify intent.

Channel messages are coordination evidence only. They are not persisted review
verdicts and cannot approve, reject, or block a run.

## Verdict Rules

Submit exactly one typed verdict through `submit_run_review` for the bound
review request. Use the daemon-provided `review_id`, `run_id`, and
`delivery_id`; do not invent identifiers.

Use outcomes honestly:

- `approved`: the terminal run satisfies the objective and constraints with
  adequate verification.
- `rejected`: the work is incomplete or wrong, and a continuation run should
  address bounded `missing_work`.
- `blocked`: external information, credentials, environment, or policy blocks a
  fair verdict.
- `error`: review execution failed in a way that invalidates the verdict.
- `timeout`: the review could not complete within the expected window.
- `invalid_output`: the run result cannot be evaluated because the output is
  malformed, missing required evidence, or violates the expected contract.

Rejected verdicts must include bounded `missing_work` and actionable
`next_round_guidance`. Approval must not include hidden TODOs. When confidence
is low, use `blocked`, `error`, `timeout`, or `invalid_output` instead of
approving uncertain work.

## Safety

Do not leak raw claim tokens, provider secrets, MCP credentials, sandbox
internals, or private session implementation details in review text. Refer to
redacted ids, hashes, task ids, run ids, review ids, event ids, and file paths.

Do not ask a worker to self-approve. Do not treat a coordinator's channel
message as a verdict. Persist the final decision only through
`submit_run_review`.
