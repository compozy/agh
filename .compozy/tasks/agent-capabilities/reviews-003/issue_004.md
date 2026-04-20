---
status: resolved
file: internal/daemon/daemon_test.go
line: 4464
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135966430,nitpick_hash:80cdc7e5234c
review_hash: 80cdc7e5234c
source_review_id: "4135966430"
source_review_submitted_at: "2026-04-19T12:48:57Z"
---

# Issue 004: Deep-clone recorded capabilities in the fake join log.
## Review Comment

`Line 4471` only copies the top-level slice. `session.NetworkPeerCapability` still carries nested slice aliases, so later mutation can bleed into recorded assertions. This fake will be more stable if it snapshots the nested fields too, like the session test fake does.

## Triage

- Decision: `valid`
- Notes:
  - `fakeNetworkRuntime.JoinChannel` snapshots `join.Capabilities` with `append([]session.NetworkPeerCapability(nil), ...)`, which only copies the outer slice.
  - Each `session.NetworkPeerCapability` contains nested slices (`ContextNeeded`, `ArtifactsExpected`, `ExecutionOutline`, `Constraints`, `Examples`), so later mutation of the original join payload can alias into recorded assertions.
  - Fix approach: deep-clone the capability catalog inside the daemon test fake and add a regression test that mutates the original join input after the call to prove the recorded snapshot is isolated.

## Resolution

- Replaced the shallow capability snapshot in `fakeNetworkRuntime.JoinChannel` with a daemon-local deep clone that copies every nested capability slice.
- Added `TestFakeNetworkRuntimeJoinChannelDeepClonesCapabilities` to mutate the original join payload after recording and verify the fake join log keeps the original values.
- Verification:
  - `go test ./internal/daemon -run TestFakeNetworkRuntimeJoinChannelDeepClonesCapabilities`
  - `make verify`
