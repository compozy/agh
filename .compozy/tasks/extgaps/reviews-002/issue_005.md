---
status: resolved
file: internal/daemon/bridges_test.go
line: 33
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110597069,nitpick_hash:9130ce09b3fa
review_hash: 9130ce09b3fa
source_review_id: "4110597069"
source_review_submitted_at: "2026-04-15T03:35:44Z"
---

# Issue 005: Fail fast on unexpected stub calls instead of returning zero-values.
## Review Comment

Returning success defaults (`nil` / `nil, nil`) can hide unconfigured stub usage and create false-positive tests. Prefer explicit unexpected-call errors.

---

## Triage

- Decision: `valid`
- Root cause: the test stub currently returns success zero-values when a method is invoked without an explicit function override. That can let tests pass even when the runtime unexpectedly touches the wrong bridge-secret path.
- Why this is valid: for a narrow behavior stub, silent success is the wrong default. Unexpected calls should fail loudly so the test surface stays trustworthy.
- Fix approach: change the three bridge-secret stub methods to return explicit unexpected-call errors when no override is installed, while keeping configured function hooks unchanged.
- Resolution: `internal/daemon/bridges_test.go` now returns explicit unexpected-call errors from the bridge-secret stub defaults instead of silent success values.
- Verification: targeted `go test ./internal/daemon -run 'TestBridgeRuntimeSecrets|TestComposeBridgeRuntime|TestBridgeRuntimeStartInstance' -count=1` passed, followed by a clean `make verify`.
