# TC-FUNC-022 — `agh__network_peers` lists peers via existing network manager

- **Priority:** P1
- **Type:** Functional / native tool
- **Trace:** Task 05, TechSpec Network And Tasks

## Test Steps

1. Mock 3 peers in network manager.
2. Invoke `agh__network_peers`.
   - **Expected:** Returns 3 peers with deterministic structure; `read_only = true`.
3. Empty peer list returns empty array, not null.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestNativeNetworkPeers`
