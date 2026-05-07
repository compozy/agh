---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/cli/client_provider_models.go
line: 55
severity: minor
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:bb422bf0d584
review_hash: bb422bf0d584
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 010: Reject blank provider IDs on provider-scoped model endpoints.
## Review Comment

`RefreshProviderModels` and `ProviderModelStatus` currently turn an empty `providerID` into `/api/providers/models/refresh` or `/api/providers/models/status`. That converts a caller bug into a misleading route failure instead of returning a clear client-side error.

## Triage

- Decision: `valid`
- Notes:
  - `internal/cli/client_provider_models.go` still lets `RefreshProviderModels("", ...)` and `ProviderModelStatus("")` build `/api/providers/models/refresh` and `/api/providers/models/status`.
  - That turns a caller bug into a misleading route miss instead of a deterministic client-side validation error.
  - Fix plan: reject blank provider IDs in the provider-scoped client methods and add/update CLI client tests. The test edit will require a minimal out-of-scope change because the scoped batch does not include the existing CLI test file.
  - Fixed in `internal/cli/client_provider_models.go` with a new regression test in `internal/cli/client_provider_models_test.go`, then verified with focused package tests plus `make verify`.
