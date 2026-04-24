---
status: resolved
file: web/src/systems/network/mocks/handlers.ts
line: 70
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167301360,nitpick_hash:915d76105af6
review_hash: 915d76105af6
source_review_id: "4167301360"
source_review_submitted_at: "2026-04-24T01:39:58Z"
---

# Issue 008: Mock handler returns fixture directly without peer ID remapping.
## Review Comment

The handler validates that `peerId` exists in `networkPeersFixture` before returning, which limits it to the two hardcoded fixture peers. This is acceptable for current mock/test purposes, but if tests need to query arbitrary peer IDs, the fixture messages would have inconsistent `peer_from`/`peer_to` values.

For improved mock fidelity, consider remapping both `peer_from` and `peer_to` when the requested peer differs from the fixture's primary peer:

## Triage

- Decision: `invalid`
- Reasoning: The handler at line 70 intentionally only serves peer IDs that already exist in `networkPeersFixture`. For those two supported peers, returning the same two-party direct-history fixture is internally consistent: the conversation is still between the local and remote fixture peers, and `web/src/systems/network/mocks/network-mocks.test.ts` already locks that contract in by asserting the remote-peer request returns the fixture unchanged.
- Why no code change: The review comment describes a hypothetical expansion to arbitrary peer IDs, but this mock surface does not claim to support arbitrary IDs. Adding remapping logic here would widen the mock semantics rather than fix a concrete defect in the current test/storybook contract.
- Outcome: Analysis complete; no code change required.
