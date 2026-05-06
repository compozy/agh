# Free Iteration 022 - Claim Profile Eligibility

## Slice

Apply `TaskExecutionProfile` worker and participant eligibility filters to `ClaimNextRun` so ineligible agent names and missing capabilities cannot claim queued runs.

## Acceptance Mapping

- `_techspec.md` implementation step 4: worker/session profile resolution must narrow task-run claim eligibility.
- `_techspec_orchestration.md` participant policy: worker claim filters must use participant and worker agent/capability selectors as narrowing predicates, not authority-widening grants.
- ADR-010: `ClaimCriteria.AgentName` and capability selectors become effective store-level filters.

## Status

- Completed.

## Changes

- Added `ClaimNextRun` profile eligibility predicates in `internal/store/globaldb/global_db_task_claim.go`.
- Enforced `task_execution_profiles.worker_agent_name` when `ClaimCriteria.AgentName` is supplied.
- Enforced worker and participant `allowed_agent_names` selector rows as upper-bound filters.
- Enforced worker and participant required capability selector rows using the same missing-capability semantics as run-level required capabilities.
- Added `TestGlobalDBClaimNextRunAppliesExecutionProfileEligibility` covering wrong agent rejection, missing participant capability rejection, queued-state preservation, and successful eligible claim.

## Verification

- `go test ./internal/store/globaldb -run 'TestGlobalDBClaimNextRunAppliesExecutionProfileEligibility|TestGlobalDBClaimNextRunFiltersByCapabilitiesScopeAndChannel' -count=1` passed.
- `go test -race ./internal/store/globaldb -run 'TestGlobalDBClaimNextRunAppliesExecutionProfileEligibility|TestGlobalDBClaimNextRunFiltersByCapabilitiesScopeAndChannel' -count=1` passed.
- `go test ./internal/store/globaldb -count=1` passed.
- `go test -race -parallel=4 ./internal/store/globaldb -count=1` passed in `71.971s`.
- `make lint` passed with `0 issues`.
- `make verify` passed: Bun lint/typecheck/test, Vitest `329 files / 2088 tests`, web build, `golangci-lint` `0 issues`, Go race gate `DONE 8200 tests in 126.178s`, and package boundaries OK.
