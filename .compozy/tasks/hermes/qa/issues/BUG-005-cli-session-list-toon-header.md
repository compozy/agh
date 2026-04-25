# BUG-005: CLI session-list TOON regression expected the pre-Hermes header

## Status

Fixed in Task 11.

## Severity

P1 verification regression. The CLI integration lane failed even though the current session-list payload correctly includes Hermes lifecycle failure diagnostics.

## Reproduction

Run:

```bash
go test -tags integration ./internal/cli -run TestSessionListOutputFormatsIntegration
```

## Observed

The TOON output assertion expected the older `session_id,name,status,...` header and failed once `failure_kind` became part of the session list contract.

## Expected

The CLI integration test should assert the current contract, including `failure_kind`, so future lifecycle-diagnostics regressions are caught rather than hidden by stale expectations.

## Root Cause

Task 03 added typed session failure diagnostics, but the integration assertion for TOON output remained pinned to the older header shape.

## Fix

Updated `internal/cli/cli_integration_test.go` to assert the current TOON header including `failure_kind`.

## Verification Evidence

- Failing repro: `.compozy/tasks/hermes/qa/logs/final/failure-repro/cli-session-list-toon.log`
- Focused proof after fix: `.compozy/tasks/hermes/qa/logs/final/failure-repro/cli-session-list-toon-after-fix.log`
- Full integration after fixes: `.compozy/tasks/hermes/qa/logs/final/make-test-integration-after-fixes.log`
