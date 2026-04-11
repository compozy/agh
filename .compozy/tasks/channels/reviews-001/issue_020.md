---
status: resolved
file: internal/cli/command_paths_test.go
line: 82
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLp,comment:PRRC_kwDOR5y4QM623eJC
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Assert the channel CLI inputs in these new stub paths.**

These closures return success no matter which `id` or `ChannelTestDeliveryRequest` the command sends, so the new cases still pass if the CLI drops `chan-1`, `--peer-id`, or `--mode`. Add argument checks inside the stub closures so this matrix fails on request-wiring regressions.

<details>
<summary>Suggested tightening</summary>

```diff
-		getChannelFn: func(context.Context, string) (ChannelRecord, error) {
+		getChannelFn: func(_ context.Context, id string) (ChannelRecord, error) {
+			if id != "chan-1" {
+				t.Fatalf("getChannel id = %q, want %q", id, "chan-1")
+			}
 			return ChannelRecord{ID: "chan-1", Scope: "global", Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: true, Status: "ready"}, nil
 		},
-		channelRoutesFn: func(context.Context, string) ([]ChannelRouteRecord, error) {
+		channelRoutesFn: func(_ context.Context, id string) ([]ChannelRouteRecord, error) {
+			if id != "chan-1" {
+				t.Fatalf("channelRoutes id = %q, want %q", id, "chan-1")
+			}
 			return []ChannelRouteRecord{{RoutingKeyHash: "hash-1", Scope: "global", ChannelInstanceID: "chan-1", PeerID: "peer-1", SessionID: "sess-1", AgentName: "coder", LastActivityAt: fixedTestNow}}, nil
 		},
-		testChannelDeliveryFn: func(context.Context, string, ChannelTestDeliveryRequest) (ChannelTestDeliveryRecord, error) {
+		testChannelDeliveryFn: func(_ context.Context, id string, req ChannelTestDeliveryRequest) (ChannelTestDeliveryRecord, error) {
+			if id != "chan-1" || req.PeerID != "peer-1" || req.Mode != "reply" {
+				t.Fatalf("testChannelDelivery(%q, %#v) did not receive expected CLI inputs", id, req)
+			}
 			return ChannelTestDeliveryRecord{Status: "resolved", DeliveryTarget: DeliveryTargetRecord{ChannelInstanceID: "chan-1", PeerID: "peer-1", Mode: "reply"}}, nil
 		},
```
</details>

As per coding guidelines, "MUST test meaningful business logic, not trivial operations."



Also applies to: 93-95

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/command_paths_test.go` around lines 74 - 82, The test stubs
(getChannelFn, channelRoutesFn, testChannelDeliveryFn) unconditionally return
success and don't assert the inputs, allowing regressions to pass; modify these
closures to validate the incoming parameters (e.g., check the channel ID
argument passed to getChannelFn and channelRoutesFn, and assert fields of the
ChannelTestDeliveryRequest in testChannelDeliveryFn such as PeerID and Mode) and
return a non-nil error when the inputs don't match the expected values (e.g.,
expected "chan-1", expected peer id, expected mode) so the test matrix fails on
wiring regressions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The command-specific CLI tests already assert the exact `chan-1` / `peer-1` / `reply` request wiring in [internal/cli/channel_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/cli/channel_test.go:34) and [internal/cli/channel_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/cli/channel_test.go:303).
  - `command_paths_test.go` is a broad smoke matrix over many commands; duplicating the detailed channel-input assertions there would add maintenance noise without expanding behavioral coverage.
  - Resolution: Closed as invalid after code inspection; the detailed CLI wiring remains covered by the dedicated channel command tests and `make verify` passed without changing the smoke matrix.
