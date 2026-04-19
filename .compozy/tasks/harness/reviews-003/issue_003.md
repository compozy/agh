---
status: resolved
file: internal/extension/host_api_test.go
line: 5376
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135421543,nitpick_hash:20fcfaf0d6e9
review_hash: 20fcfaf0d6e9
source_review_id: "4135421543"
source_review_submitted_at: "2026-04-19T03:26:18Z"
---

# Issue 003: Make unexpected Prompt and Events calls fail by default.
## Review Comment

The other stub methods are strict, but these two silently succeed when no callback is wired. That can let a test pass even when `HostAPIHandler.submitPrompt` takes an unintended interaction path.

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls."

## Triage

- Decision: valid
- Root cause: `HostAPIHandler.submitPrompt` always calls both `sessions.Prompt` and `sessions.Events`, but `promptSessionManagerStub` is permissive when `promptFn` or `eventsFn` are unset. `Prompt` returns a closed channel with no error, and `Events` returns `(nil, nil)`, so a test can omit one callback and still avoid an explicit failure.
- Impact: helper misuse can either make `submitPrompt` appear successful when no prompt submission actually happened or surface a misleading downstream error (`turn id not found`) instead of the real problem that the test never wired the expected session-manager call.
- Fix approach: make both stub methods strict by default so unwired `Prompt` and `Events` calls return explicit unexpected-call errors, and add focused submit-prompt regression coverage for both missing-callback cases.
- Implemented: `promptSessionManagerStub.Events` and `promptSessionManagerStub.Prompt` now return explicit `unexpected ... call` errors when the corresponding callback is not wired. This matches the strict behavior of the other stub methods and fails at the real misuse point instead of allowing silent success.
- Regression coverage: added `TestHostAPIHandlerSubmitPromptRejectsUnexpectedStubCalls` in `internal/extension/host_api_test.go` to cover both missing `promptFn` and missing `eventsFn` paths. The pre-existing `TestHostAPIHandlerSubmitPromptRejectsMissingBoundaryEvents` still verifies the downstream missing-boundary failure when both required session-manager calls are intentionally wired.
- Verification:
  - `go test ./internal/extension -run 'TestHostAPIHandlerSubmitPrompt(RejectsMissingSessionManager|RejectsMissingBoundaryEvents|RejectsUnexpectedStubCalls)' -count=1` → `ok  	github.com/pedronauck/agh/internal/extension	0.013s`
  - `go test ./internal/extension -count=1` → `ok  	github.com/pedronauck/agh/internal/extension	3.143s`
  - `make verify` → exit code `0`; frontend checks passed (`167` test files, `1173` tests), Go lint reported `0 issues`, Go test suite completed with `DONE 5345 tests in 8.753s`, and package boundary checks ended with `OK: all package boundaries respected`.
