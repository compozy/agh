# BUG-003: Settings Transport Parity Reported Unknown Mutation Availability

## Summary

- Status: `Fixed`
- Severity: `High`
- Origin cases: `TC-FUNC-012`, `TC-INT-013`
- Surface: `/settings/hooks-extensions`

## Symptom

The hooks/extensions UI could show the wrong enablement state because the daemon runtime reported transport parity as an all-zero value instead of reflecting whether settings and extension mutations were available on the active bind host.

## Root Cause

`TransportParityStatus()` returned the zero-value `TransportParityStatus` struct. That left the frontend unable to distinguish loopback-safe mutation availability from non-loopback HTTP restriction behavior.

## Fix

- Compute transport parity from the daemon bind host in `internal/daemon/settings.go`.
- Mark parity as known and separately derive HTTP vs UDS mutation availability.
- Add daemon regression coverage for loopback IPv4, `localhost`, wildcard IPv4, and non-loopback hosts.
- Keep the non-loopback operator restriction scenario in committed Playwright coverage.

## Regression Coverage

- `internal/daemon/settings_test.go`
- `web/e2e/settings.spec.ts`
- `web/e2e/settings-transport.spec.ts`
- `go test -count=1 -tags integration ./internal/api/udsapi -run TestUDSTransportSettingsMutationsRemainPrivilegedWhenHTTPIsNonLoopback`
- `make test-e2e-web`
- `make verify`

## Evidence

- `qa/screenshots/TC-FUNC-012-hooks-extensions-hybrid.png`
- `qa/screenshots/TC-INT-013-non-loopback-http-restrictions.png`
