---
status: resolved
file: internal/extension/host_api_integration_test.go
line: 293
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLy,comment:PRRC_kwDOR5y4QM623eJN
---

# Issue 033: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Concurrent test goroutines may silently fail without proper test reporting.**

In the concurrency test, `t.Fatalf` called from within a goroutine (line 271) won't properly terminate the test - it will panic instead. The current error collection pattern stores errors in `results[idx].err` which is correct, but `decodeResult` at line 262 could also call `t.Fatal` inside the goroutine if unmarshaling fails.

Consider wrapping the goroutine body to capture panics or ensure `decodeResult` doesn't call `t.Fatal`:

<details>
<summary>🛡️ Suggested safer pattern</summary>

```diff
 	for idx := range results {
 		idx := idx
 		go func() {
-			defer func() { done <- struct{}{} }()
+			defer func() {
+				if r := recover(); r != nil {
+					results[idx].err = fmt.Errorf("panic: %v", r)
+				}
+				done <- struct{}{}
+			}()
 			result, err := env.callWithContext(t, ctx, "telegram-adapter", "channels/messages/ingest", map[string]any{
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestHostAPIIntegrationChannelsMessagesIngestConcurrentSameRoutingKeyUsesOneRouteAndSession(t *testing.T) {
	env := newHostAPITestEnv(t)
	env.useSessionsWithoutObserver(t)
	env.grant("telegram-adapter", []string{"channels/messages/ingest"}, []string{"channel.write"})

	instance := env.createChannelInstance(t, channelspkg.CreateInstanceRequest{
		ID:            "chan-integration-concurrent",
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.channelContext(t, instance)

	type ingestResult struct {
		result hostAPIChannelsMessagesIngestResult
		err    error
	}

	results := make([]ingestResult, 2)
	done := make(chan struct{}, len(results))
	for idx := range results {
		idx := idx
		go func() {
			defer func() {
				if r := recover(); r != nil {
					results[idx].err = fmt.Errorf("panic: %v", r)
				}
				done <- struct{}{}
			}()
			result, err := env.callWithContext(t, ctx, "telegram-adapter", "channels/messages/ingest", map[string]any{
				"channel_instance_id": instance.ID,
				"scope":               instance.Scope,
				"workspace_id":        instance.WorkspaceID,
				"peer_id":             "peer-1",
				"platform_message_id": fmt.Sprintf("msg-%d", idx),
				"received_at":         env.currentTime().Format(time.RFC3339Nano),
				"idempotency_key":     fmt.Sprintf("idem-%d", idx),
				"content":             map[string]any{"text": fmt.Sprintf("hello-%d", idx)},
			})
			if err != nil {
				results[idx].err = err
				return
			}
			decodeResult(t, result, &results[idx].result)
		}()
	}
	for range results {
		<-done
	}

	for idx, result := range results {
		if result.err != nil {
			t.Fatalf("ingest[%d] error = %v", idx, result.err)
		}
	}

	routes, err := env.channels.ListRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("channels.ListRoutes() error = %v", err)
	}
	if got := len(routes); got != 1 {
		t.Fatalf("len(routes) = %d, want 1", got)
	}

	sessions, err := env.sessions.ListAll(testutil.Context(t))
	if err != nil {
		t.Fatalf("sessions.ListAll() error = %v", err)
	}
	if got := len(sessions); got != 1 {
		t.Fatalf("len(sessions) = %d, want 1", got)
	}
	if results[0].result.SessionID != results[1].result.SessionID {
		t.Fatalf("session IDs = %q and %q, want same session", results[0].result.SessionID, results[1].result.SessionID)
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_integration_test.go` around lines 226 - 293, The
test
TestHostAPIIntegrationChannelsMessagesIngestConcurrentSameRoutingKeyUsesOneRouteAndSession
must not call t.Fatal/t.Fatalf from background goroutines (decodeResult may call
t.Fatal). Fix by preventing goroutine panics from escaping: wrap the goroutine
body with a panic catcher that converts panics into results[idx].err (e.g. defer
func(){ if r:=recover(); r!=nil { results[idx].err = fmt.Errorf("panic: %v", r)
} }()), and/or change decodeResult to return an error instead of calling t.Fatal
so the goroutine can set results[idx].err on decode failures; locate
decodeResult and the anonymous goroutine in
TestHostAPIIntegrationChannelsMessagesIngestConcurrentSameRoutingKeyUsesOneRouteAndSession
to apply these changes.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: The concurrent ingest test calls `decodeResult(t, ...)` inside worker goroutines. `decodeResult` uses `t.Fatalf`, which is not safe for background goroutines and can panic the test instead of reporting through the collected result channel.
- Root cause: The goroutines reuse a helper that terminates via the testing object instead of returning a normal error.
- Fix plan: Decode the JSON-RPC result in the goroutine through an error-returning helper and store decode failures in `results[idx].err`.
- Resolution: Added an error-returning decode helper for the concurrent goroutines and verified the integration test plus `make verify`.
