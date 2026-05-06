# Free Slice 002

## Objective

- Add typed task orchestration config defaults and validation for `[task.orchestration]`, `[task.orchestration.profile]`, and `[task.orchestration.review]`.
- The slice was selected in state iteration 002 and completed in state iteration 003.

## Scope

- Backend config surface only.
- No schema migration, HTTP/OpenAPI contract, web UI, site docs, or `docs/_memory` lesson was required for this slice.
- Frontend/docs delegation through Claude Opus was not applicable because this slice did not modify frontend or documentation surfaces.

## Files Changed

- `internal/config/task_orchestration.go`
- `internal/config/task_orchestration_test.go`
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/tool_surface.go`
- `internal/config/tool_surface_test.go`

## Decisions

- Keep task orchestration defaults in `internal/config` alongside existing typed daemon config defaults.
- Add `TaskConfig` under the root `Config` as `toml:"task"` so future orchestration, review-gate, and profile code consumes validated config rather than ad hoc constants.
- Expose the new task config paths through the existing agent-mutable config surface so agents can manage them via the current config tooling.
- Accept only profile/review modes that are implemented by this slice: coordinator `inherit|guided`, worker `inherit`, sandbox `inherit|none`, review `none|on_success|on_failure|always`, and review failure `block_task|fail_task`.

## Validation Evidence

- `go test ./internal/config -count=1`: pass.
- `go test -race ./internal/config -count=1`: pass.
- `make verify`: pass, including Bun lint/typecheck/tests, web build, Go lint, race tests, build, and boundaries.
- `scripts/check-test-conventions.py` was not present in this repository; `rg --files | rg 'check-test-conventions\.py$|test-conventions'` found no matching helper.

## Errors And Corrections

- First `make verify` run failed on `golangci-lint` `lll` for `internal/config/task_orchestration.go`.
- The long line was wrapped and verification was rerun instead of suppressing lint or weakening tests.

## Acceptance Mapping

- Advances the TechSpec config lifecycle requirement by adding typed defaults, TOML overlay support, validation, and agent-manageable config paths.
- Advances orchestration/review-gate implementation foundations but does not complete orchestration kernel hardening, durable state/cursors, review continuation loop, built-in tools, web, site, docs, QA pair, or CodeRabbit clean rounds.

## Next Cursor

- Continue Phase B in free mode with another small slice anchored to the TechSpecs and ADRs.
- Candidate next backend slice: durable task orchestration storage/read models or task execution profile consumption, depending on the next TechSpec cursor.
