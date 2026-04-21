---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 130
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMr,comment:PRRC_kwDOR5y4QM65IPD_
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Bound prompt collection with a timeout to prevent hanging CI runs.**

Lines 116-130 use `context.Background()` for prompt submission and Lines 2350-2352 drain channels without timeout. If a prompt stream stalls, this test can block indefinitely.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 func TestUDSSessionTranscriptEndpointIncludesSyntheticTurns(t *testing.T) {
     runtime := newIntegrationRuntime(t)
     sessionID := createIntegrationSession(t, runtime)
+    promptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
+    defer cancel()

-    userEvents, userErr := runtime.manager.Prompt(context.Background(), sessionID, "hello")
-    collectIntegrationPromptEvents(t, mustIntegrationPrompt(t, userEvents, userErr))
+    userEvents, userErr := runtime.manager.Prompt(promptCtx, sessionID, "hello")
+    collectIntegrationPromptEvents(t, mustIntegrationPrompt(t, userEvents, userErr), 5*time.Second)

-    networkEvents, networkErr := runtime.manager.PromptNetwork(context.Background(), sessionID, "network hello")
-    collectIntegrationPromptEvents(t, mustIntegrationPrompt(t, networkEvents, networkErr))
+    networkEvents, networkErr := runtime.manager.PromptNetwork(promptCtx, sessionID, "network hello")
+    collectIntegrationPromptEvents(t, mustIntegrationPrompt(t, networkEvents, networkErr), 5*time.Second)

-    syntheticEvents, syntheticErr := runtime.manager.PromptSynthetic(context.Background(), sessionID, session.SyntheticPromptOpts{
+    syntheticEvents, syntheticErr := runtime.manager.PromptSynthetic(promptCtx, sessionID, session.SyntheticPromptOpts{
         Message: "daemon wake-up",
         Metadata: acp.PromptSyntheticMeta{
@@
-    collectIntegrationPromptEvents(t, mustIntegrationPrompt(t, syntheticEvents, syntheticErr))
+    collectIntegrationPromptEvents(t, mustIntegrationPrompt(t, syntheticEvents, syntheticErr), 5*time.Second)
@@
-func collectIntegrationPromptEvents(t *testing.T, events <-chan acp.AgentEvent) []acp.AgentEvent {
+func collectIntegrationPromptEvents(t *testing.T, events <-chan acp.AgentEvent, timeout time.Duration) []acp.AgentEvent {
     t.Helper()

     collected := make([]acp.AgentEvent, 0, 4)
-    for event := range events {
-        collected = append(collected, event)
-    }
+    timer := time.NewTimer(timeout)
+    defer timer.Stop()
+Loop:
+    for {
+        select {
+        case event, ok := <-events:
+            if !ok {
+                break Loop
+            }
+            collected = append(collected, event)
+        case <-timer.C:
+            t.Fatal("timed out waiting for prompt events")
+        }
+    }
```
</details>


Also applies to: 2346-2358

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/udsapi_integration_test.go` around lines 116 - 130,
Replace uses of context.Background() when calling runtime.manager.Prompt,
PromptNetwork and PromptSynthetic with a cancellable timeout context (e.g., use
context.WithTimeout) so each prompt call times out instead of hanging; update
the invocations that pass session.SyntheticPromptOpts (and the calls referencing
runtime.manager.Prompt, PromptNetwork, PromptSynthetic) to use that timeout
context and cancel when done, and also update the channel-draining logic in
collectIntegrationPromptEvents / the drain loop (the code referenced around
lines 2346-2358) to use a select with a timeout case so draining stops after the
same or a configurable deadline rather than blocking indefinitely.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - This is the UDS counterpart to issue 003: prompt submission uses `context.Background()` and `collectIntegrationPromptEvents` drains with an unbounded range loop.
  - A stalled or non-closing prompt stream would hang the test until the package timeout.
  - I will introduce a bounded prompt context plus timeout-based draining for the UDS transcript integration path.
