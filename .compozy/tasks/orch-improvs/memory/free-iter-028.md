# Free Iteration 028

## Slice

Add bundled orchestration skills `agh-orchestrator`, `agh-task-worker`, and
`agh-task-reviewer` with deterministic metadata and bundled loader coverage.

## Goal

- Ship the instructional bundled skill content required by the orchestration
  and review-gate TechSpecs.
- Keep runtime authority out of skill prose: `task.Service` remains the owner
  for task state, claims, review requests, verdicts, and continuation runs.
- Prove the bundled loader embeds, parses, and exposes the required
  `metadata.agh` guardrails.

## Implementation Notes

- `agh-task-worker` is scoped to worker sessions with active task claims and
  session-bound task tool loops.
- `agh-orchestrator` is scoped to daemon-managed coordinator sessions and
  documents channel routing as coordination, not authority.
- `agh-task-reviewer` is scoped to sessions bound to review requests and
  requires persisted verdict submission through `submit_run_review`.

## Verification

- `python3 .agents/skills/skill-best-practices/scripts/validate-metadata.py`
  passed for `agh-task-worker`, `agh-orchestrator`, and `agh-task-reviewer`.
- `go test ./internal/skills/bundled -run 'TestBundledFSContainsExpectedSkills|TestBundledSkillsParseWithLoader|TestBundledOrchestration' -count=1`
  passed.
- `go test ./internal/skills ./internal/skills/bundled -count=1` passed.
- `go test -race ./internal/skills/bundled -run 'TestBundledSkillsParseWithLoader|TestBundledOrchestration|TestBundledRegistry' -count=1`
  passed.
- `make lint` passed with `0 issues`.
- First `make verify` failed once in
  `internal/session TestMessageDeltaAsyncHooksDoNotBlockPromptStreaming` while
  waiting for the prompt event channel to close. The failure did not reproduce
  with `go test -race -parallel=4 ./internal/session -run TestMessageDeltaAsyncHooksDoNotBlockPromptStreaming -count=10`
  or `go test -race -parallel=4 ./internal/session -count=1`.
- Final `make verify` passed: Bun lint/typecheck/test, Vitest 329 files /
  2088 tests, web build, `golangci-lint` 0 issues, Go race gate `DONE 8224
  tests in 31.091s`, and package boundaries respected.

## Open Risks

- Native tool registration for `submit_run_review`, API/UDS/CLI surfaces,
  web/site docs, `docs/_memory` lessons, QA pair, and CodeRabbit rounds remain
  outside this slice.
