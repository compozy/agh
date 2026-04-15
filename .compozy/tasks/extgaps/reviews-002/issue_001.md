---
status: resolved
file: internal/api/core/bundles.go
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110597069,nitpick_hash:4e0f4e2b2473
review_hash: 4e0f4e2b2473
source_review_id: "4110597069"
source_review_submitted_at: "2026-04-15T03:35:44Z"
---

# Issue 001: Nil receiver check is misleading — calling h.respondError on nil h will panic.
## Review Comment

The check `h == nil` is defensive but creates a false sense of safety. If `h` is actually nil, the subsequent call to `h.respondError(c, ...)` will dereference the nil pointer and panic.

In practice, `h` is instantiated during server setup and will never be nil at runtime, so the check is unnecessary. Consider removing the `h == nil` portion or responding directly via `c.JSON` if you want to keep this ultra-defensive path.

This same pattern repeats in `ListBundleActivations`, `GetBundleActivation`, `UpdateBundleActivation`, `DeleteBundleActivation`, `BundleNetworkSettings`, and `bindBundleActivateRequest`.

## Triage

- Decision: `valid`
- Root cause: the handler tries to be defensive about `h == nil`, but the defensive branch still calls the method receiver (`h.respondError(...)`), which will panic on a nil receiver before the HTTP error response is written.
- Why this is valid: the current code path is internally inconsistent and does not provide the safety it claims to provide. Either the nil check must be removed or the error response must avoid dereferencing `h`.
- Fix approach: keep the defensive behavior but route the nil-handler path through the package-level `RespondError(...)` helper, then keep `h.respondError(...)` only for non-nil handlers. Add regression coverage in `internal/api/core/handlers_internal_test.go`.
- Resolution: implemented the shared `bundleServiceRequired(...)` guard in `internal/api/core/bundles.go` and added nil-receiver regression coverage for the bundle handlers in `internal/api/core/handlers_internal_test.go`.
- Verification: targeted `go test ./internal/api/core -run 'TestBundleHandlersRejectNilReceiverWithoutPanicking|TestNetworkStatusPayloadWrapsBundleSettingsErrors' -count=1` passed, followed by a clean `make verify`.
