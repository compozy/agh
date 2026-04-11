---
status: resolved
file: internal/extensiontest/channel_adapter_harness.go
line: 417
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBL8,comment:PRRC_kwDOR5y4QM623eJX
---

# Issue 035: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Check if the file exists and examine the method at lines 401-417
cd /repo && head -420 internal/extensiontest/channel_adapter_harness.go | tail -60
```

Repository: compozy/agh

Length of output: 110

---

🏁 Script executed:

```shell
# Get more context about the ScriptedPromptDriver.Prompt method
cd /repo && grep -n "func (d \*ScriptedPromptDriver) Prompt" internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 110

---

🏁 Script executed:

```shell
# Check the full method signature and implementation
cd /repo && sed -n '395,425p' internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 110

---

🏁 Script executed:

```shell
# Verify the stopSessions error handling mentioned in scratchpad (line 779-786)
cd /repo && sed -n '770,790p' internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 110

---

🏁 Script executed:

```shell
pwd && ls -la
```

Repository: compozy/agh

Length of output: 2633

---

🏁 Script executed:

```shell
git ls-files | grep channel_adapter_harness
```

Repository: compozy/agh

Length of output: 226

---

🏁 Script executed:

```shell
find . -name "channel_adapter_harness.go" -type f
```

Repository: compozy/agh

Length of output: 106

---

🏁 Script executed:

```shell
sed -n '395,425p' ./internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 798

---

🏁 Script executed:

```shell
sed -n '770,790p' ./internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 575

---

🏁 Script executed:

```shell
# Get full method signature to verify context handling
sed -n '395,430p' ./internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 964

---

🏁 Script executed:

```shell
sed -n '388,395p' ./internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 314

---



</details>

**Bind the scripted emitter goroutine to `ctx` and stop ignoring errors in `stopSessions`.**

The `Prompt` method ignores its context parameter and starts a goroutine that unconditionally sleeps and sends events without respecting cancellation. This violates the requirement that every goroutine must have explicit ownership via context cancellation and must use `select` on `ctx.Done()` in long-running loops. Canceled/stopped prompts can keep emitting events after teardown starts, causing flaky harness behavior.

Additionally, `stopSessions` (line 779-786) discards errors from `h.Sessions.Stop(...)` with `_ =`, which is not justified and can hide cleanup failures.

<details>
<summary>Suggested fix for Prompt</summary>

```diff
-func (d *ScriptedPromptDriver) Prompt(_ context.Context, _ *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
+func (d *ScriptedPromptDriver) Prompt(ctx context.Context, _ *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
 	d.mu.Lock()
 	d.prompts = append(d.prompts, req)
 	script := append([]ScriptedPromptEvent(nil), d.script...)
 	startedAt := d.now
 	d.mu.Unlock()

 	events := make(chan acp.AgentEvent, len(script))
 	go func() {
 		defer close(events)
 		for idx, item := range script {
 			if item.Delay > 0 {
-				time.Sleep(item.Delay)
+				timer := time.NewTimer(item.Delay)
+				select {
+				case <-timer.C:
+				case <-ctx.Done():
+					if !timer.Stop() {
+						select {
+						case <-timer.C:
+						default:
+						}
+					}
+					return
+				}
 			}
-			events <- acp.AgentEvent{
+			select {
+			case events <- acp.AgentEvent{
 				Type:      item.Type,
 				TurnID:    req.TurnID,
 				Timestamp: startedAt.Add(time.Duration(idx+1) * time.Millisecond),
 				Text:      item.Text,
 				Error:     item.Error,
-			}
+			}:
+			case <-ctx.Done():
+				return
+			}
 		}
 	}()
 	return events, nil
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extensiontest/channel_adapter_harness.go` around lines 401 - 417,
The Prompt method currently ignores the passed ctx and launches a goroutine that
sleeps and sends events unconditionally; change the goroutine in Prompt to watch
ctx.Done() (use select with time.After or a time.Timer) before each send and
exit early if ctx is cancelled to ensure no events are emitted after teardown
(update references: Prompt, events channel, startedAt, script loop, item.Delay,
acp.AgentEvent sends). Also stop discarding errors in stopSessions: remove `_ =
h.Sessions.Stop(...)` and propagate or return the error (or wrap/log it) from
stopSessions so cleanup failures aren’t hidden (reference: stopSessions and
h.Sessions.Stop).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: `ScriptedPromptDriver.Prompt` ignores `ctx` and continues sleeping/sending after cancellation, which can leak harness activity past teardown. The same file also hides cleanup failures in `stopSessions`.
- Root cause: The prompt replay goroutine uses unconditional `time.Sleep` plus unconditional sends, and the cleanup loop discards `Sessions.Stop` errors.
- Fix plan: Make prompt replay respect `ctx.Done()` while waiting and sending, and surface `Sessions.Stop` failures from harness cleanup.
- Resolution: Prompt replay now stops on cancellation, harness cleanup reports stop failures, and both the focused harness tests and `make verify` passed.
