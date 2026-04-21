---
status: resolved
file: internal/network/router_integration_test.go
line: 220
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58PQ7L,comment:PRRC_kwDOR5y4QM65eHnn
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Registry expiry window is too short and can make these new integration tests flaky.**

Line 215, Line 219, Line 340, and Line 344 use `50 * time.Millisecond`, while nearby integration tests were already moved to `time.Second` for stability. Under CI jitter, remotes can expire before assertions complete.



<details>
<summary>Suggested stability fix</summary>

```diff
-	registryA, err := NewPeerRegistry(50 * time.Millisecond)
+	registryA, err := NewPeerRegistry(time.Second)
...
-	registryB, err := NewPeerRegistry(50 * time.Millisecond)
+	registryB, err := NewPeerRegistry(time.Second)
...
-	registryA, err := NewPeerRegistry(50 * time.Millisecond)
+	registryA, err := NewPeerRegistry(time.Second)
...
-	registryB, err := NewPeerRegistry(50 * time.Millisecond)
+	registryB, err := NewPeerRegistry(time.Second)
```
</details>


Also applies to: 340-345

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/router_integration_test.go` around lines 215 - 220, The
tests create peer registries with too-short expiry windows (NewPeerRegistry
called for registryA, registryB and the other registry instances) using 50 *
time.Millisecond which makes the integration tests flaky under CI jitter; change
those NewPeerRegistry calls to use a longer timeout (e.g., time.Second) for each
instance (registryA, registryB and the other NewPeerRegistry invocations
referenced) so remotes do not expire before assertions complete.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the two new rich-`whois` integration tests create peer registries with `50 * time.Millisecond` expiry windows while the test logic depends on asynchronous greet/discovery/message delivery over real transport and wall-clock timestamps.
- Why this is valid: nearby integration tests in the same file already use `time.Second`, and the 50ms window can let remote presence expire under CI jitter before the assertions complete, creating flakiness unrelated to the behavior under test.
- Fix approach: raise those four `NewPeerRegistry(...)` calls to `time.Second` so the tests continue exercising rich discovery/refresh behavior without a race against registry expiry.

## Resolution

- Raised the four rich-`whois` integration-test peer registries from `50 * time.Millisecond` to `time.Second` so remote presence does not expire during asynchronous discovery and response assertions.
- Verification:
  - `go test -tags integration ./internal/network -run 'TestDirectedWhoisRichDiscoveryDeliversPeerCardAndCapabilityCatalog|TestDirectedWhoisRichDiscoveryFilteringRefreshesRemotePresence' -count=1`
  - `make verify`
