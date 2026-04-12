---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 660
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLe,comment:PRRC_kwDOR5y4QM623eI2
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`StartInstance` and `RestartInstance` leave the test channel permanently unavailable.**

This harness moves instances to `ChannelStatusStarting` but never advances them to `ChannelStatusReady`, so any enable/restart flow followed by an availability-gated operation will keep behaving like the channel never finished booting. If this runtime is meant to be synchronous, mark it `ready` here after the state transition.

<details>
<summary>One way to model the ready transition synchronously</summary>

```diff
 func (s *integrationChannelService) StartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
-	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
+	if _, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
 		ID:      id,
 		Enabled: true,
 		Status:  channelspkg.ChannelStatusStarting,
-	})
+	}); err != nil {
+		return nil, fmt.Errorf("start channel instance %q: %w", id, err)
+	}
+	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
+		ID:      id,
+		Enabled: true,
+		Status:  channelspkg.ChannelStatusReady,
+	})
 }
@@
 func (s *integrationChannelService) RestartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
-	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
+	if _, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
 		ID:      id,
 		Enabled: true,
 		Status:  channelspkg.ChannelStatusStarting,
-	})
+	}); err != nil {
+		return nil, fmt.Errorf("restart channel instance %q: %w", id, err)
+	}
+	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
+		ID:      id,
+		Enabled: true,
+		Status:  channelspkg.ChannelStatusReady,
+	})
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 638 - 660,
StartInstance and RestartInstance set instances to ChannelStatusStarting and
leave them stuck; update them to mark the instance ready synchronously by
calling UpdateInstanceState with Status: channelspkg.ChannelStatusReady (or
perform a second UpdateInstanceState to transition from ChannelStatusStarting to
ChannelStatusReady) after enabling the instance; change the implementations of
StartInstance and RestartInstance to ensure the final state is
ChannelStatusReady so availability-gated operations see the channel as ready.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The HTTP integration harness `StartInstance` / `RestartInstance` methods currently stop at `ChannelStatusStarting`, so availability-gated flows can observe the test instance as permanently unavailable.
  - I will make the harness transition synchronously to `ready` after the intermediate `starting` state and keep the UDS mirror helper aligned for transport parity.
  - Resolution: Updated the HTTP and mirrored UDS integration harness lifecycle helpers in [internal/api/httpapi/httpapi_integration_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/httpapi/httpapi_integration_test.go:638) and [internal/api/udsapi/udsapi_integration_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/udsapi/udsapi_integration_test.go:316), and added a readiness regression test in the HTTP integration harness; verified with `make verify`.
