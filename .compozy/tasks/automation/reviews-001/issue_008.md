---
status: resolved
file: internal/automation/dispatch.go
line: 777
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0N,comment:PRRC_kwDOR5y4QM623e7Z
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's check if the file exists and locate the function
fd -t f "dispatch.go" --path "*automation*"
```

Repository: compozy/agh

Length of output: 288

---

🏁 Script executed:

```shell
# Get the file and check lines around 759-777
wc -l internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 90

---

🏁 Script executed:

```shell
# Read the specific function
sed -n '759,777p' internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 507

---

🏁 Script executed:

```shell
# Get more context to understand how this function is called
sed -n '700,800p' internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 2474

---

🏁 Script executed:

```shell
# Search for calls to collectPromptError to understand the context
rg "collectPromptError" -B 3 -A 3
```

Repository: compozy/agh

Length of output: 1443

---

🏁 Script executed:

```shell
# Look at the test file to understand expected behavior
rg "TestCollectPromptError" -A 20 internal/automation/dispatch_test.go
```

Repository: compozy/agh

Length of output: 532

---

🏁 Script executed:

```shell
# Check where collectPromptError is called in the dispatch flow
sed -n '820,850p' internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 786

---

🏁 Script executed:

```shell
# Look at the overall dispatch flow to understand if ctx cancellation is expected during the drain
sed -n '550,650p' internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 2461

---

🏁 Script executed:

```shell
# Find where the events channel is created/passed to collectPromptError
rg "collectPromptError\(ctx, events\)" -B 10 -A 5 internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 541

---

🏁 Script executed:

```shell
# Look at the executeAgentPrompt function and surrounding code to see the full dispatch flow
rg -A 50 "func \(d \*Dispatcher\) executeAgentPrompt" internal/automation/dispatch.go | head -80
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for where the events channel is created and used
rg "AgentEvent" internal/automation/dispatch.go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 428

---

🏁 Script executed:

```shell
# Check if d.sessions.Prompt is expected to close the channel on context cancellation
rg -A 30 "func.*Prompt.*ctx.*context" internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 698

---

🏁 Script executed:

```shell
# Look for the implementation of SessionCreator.Prompt to see if it handles cancellation
fd -type f "*.go" | xargs rg "type SessionCreator" -A 5
```

Repository: compozy/agh

Length of output: 710

---

🏁 Script executed:

```shell
# Check if there's any documentation about how the channel is handled
rg "AgentEvent" -B 5 -A 5 internal/automation/*.go | head -60
```

Repository: compozy/agh

Length of output: 4127

---

🏁 Script executed:

```shell
# Check if there are other places in the codebase that use select with ctx.Done() for channel draining
rg "select" -A 5 internal/automation/dispatch.go | grep -A 5 "ctx.Done()"
```

Repository: compozy/agh

Length of output: 126

---

🏁 Script executed:

```shell
# Look at how sleepWithContext is implemented for pattern comparison
sed -n '805,820p' internal/automation/dispatch.go
```

Repository: compozy/agh

Length of output: 317

---

🏁 Script executed:

```shell
# Check if there are similar channel draining patterns elsewhere
rg "for.*range.*chan" internal/automation/*.go -B 2 -A 5
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Respect context cancellation while draining the event stream.**

The loop blocks indefinitely waiting for `events` to close. If the producer fails to close the channel after `ctx` is canceled, the function hangs. Use `select` to monitor both `ctx.Done()` and the event channel.

<details>
<summary>Suggested fix</summary>

```diff
 func collectPromptError(ctx context.Context, events <-chan acp.AgentEvent) error {
 	if events == nil {
 		return errors.New("automation: prompt event stream is required")
 	}
 
 	var errs []error
-	for event := range events {
-		if trimmed := strings.TrimSpace(event.Error); trimmed != "" {
-			errs = append(errs, errors.New(trimmed))
-		}
-	}
-
-	if len(errs) > 0 {
-		return errors.Join(errs...)
-	}
-	if err := ctx.Err(); err != nil {
-		return err
-	}
-	return nil
+	for {
+		select {
+		case <-ctx.Done():
+			if len(errs) > 0 {
+				return errors.Join(append(errs, ctx.Err())...)
+			}
+			return ctx.Err()
+		case event, ok := <-events:
+			if !ok {
+				if len(errs) > 0 {
+					return errors.Join(errs...)
+				}
+				return nil
+			}
+			if trimmed := strings.TrimSpace(event.Error); trimmed != "" {
+				errs = append(errs, errors.New(trimmed))
+			}
+		}
+	}
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 759 - 777, collectPromptError
currently blocks reading from events until the channel closes and doesn't
respect context cancellation; update collectPromptError to use a select loop
that watches both ctx.Done() and the events channel (e.g., for { select { case
<-ctx.Done(): return ctx.Err() case event, ok := <-events: if !ok { break out of
loop } ... } }) so you stop draining when context is canceled, still collect
trimmed event.Error into errs (as currently) when events are received, and after
the loop return errors.Join(errs...) or ctx.Err() appropriately; reference the
collectPromptError function and the errs slice and event.Error symbols when
applying the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `collectPromptError` ranges the event channel until close and only checks `ctx.Err()` afterward, so a misbehaving producer can hang dispatch forever after cancellation. I will drain with `select` on both `ctx.Done()` and the event stream, then add a regression test for cancellation without channel close.
