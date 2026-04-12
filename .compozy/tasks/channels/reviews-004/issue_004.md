---
status: resolved
file: internal/daemon/daemon_test.go
line: 2864
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093927386,nitpick_hash:d8aaad935c6b
review_hash: d8aaad935c6b
source_review_id: "4093927386"
source_review_submitted_at: "2026-04-11T15:47:00Z"
---

# Issue 004: Preserve explicit empty capability/action/security lists in fixtures.
## Review Comment

Using `len(...) == 0` here makes `nil` and `[]string{}` behave the same, so callers cannot build a manifest that intentionally declares no `provides`, `requires`, or `security.capabilities`. That weakens negative handshake/authorization coverage because the helper silently injects `"memory.backend"`, `"sessions/list"`, and `"session.read"`.

---

## Triage

- Decision: `valid`
- Notes:
  - `daemonTestExtensionManifest` currently applies default `provides`, `requires`, and `security.capabilities` values whenever `len(...) == 0`.
  - That collapses `nil` and explicit empty slices, so tests cannot construct a manifest that intentionally declares no capabilities/actions/security requirements.
  - Fix approach: preserve explicit empty slices and only inject defaults when the option slice is `nil`, then add a regression test for the empty-list case.

## Resolution

- Changed `daemonTestExtensionManifest` to use defaults only when the option slices are `nil`, preserving explicit empty lists.
- Added regression coverage for both nil-default and explicit-empty manifest cases.
- Verified with `go test ./internal/daemon` and `make verify`.
