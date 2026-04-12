---
status: resolved
file: internal/extension/host_api.go
line: 723
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbp,comment:PRRC_kwDOR5y4QM624L_Y
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's look at the specific file and line ranges
cd /repo 2>/dev/null || cd . 
wc -l internal/extension/host_api.go
```

Repository: compozy/agh

Length of output: 90

---

🏁 Script executed:

```shell
# Read the submitPrompt function around lines 718-723
sed -n '708,756p' internal/extension/host_api.go
```

Repository: compozy/agh

Length of output: 1400

---

🏁 Script executed:

```shell
# Read the drainAgentEvents function around lines 1183-1186
sed -n '1183,1195p' internal/extension/host_api.go
```

Repository: compozy/agh

Length of output: 279

---

🏁 Script executed:

```shell
# Look for context.Context usage and WaitGroup patterns in this file
rg "sync\.WaitGroup|context\.Context|ctx\.Done\(\)" internal/extension/host_api.go -A 2 -B 2
```

Repository: compozy/agh

Length of output: 5892

---

🏁 Script executed:

```shell
# Check the full function signature for submitPrompt to see if context is used
sed -n '708,730p' internal/extension/host_api.go
```

Repository: compozy/agh

Length of output: 791

---

🏁 Script executed:

```shell
# Check for any WaitGroup or goroutine tracking in the function or nearby code
sed -n '700,850p' internal/extension/host_api.go | grep -E "WaitGroup|wg\.|Add\(|Wait\(|Done\(\)" || echo "No WaitGroup patterns found"
```

Repository: compozy/agh

Length of output: 116

---

🏁 Script executed:

```shell
# Look for any defer or cleanup mechanism that might track the goroutine
sed -n '700,850p' internal/extension/host_api.go | grep -E "defer|cleanup|cancel|Close"
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the context.WithoutCancel usage - understand why the context is detached
rg "context\.WithoutCancel" internal/extension/host_api.go -B 3 -A 3
```

Repository: compozy/agh

Length of output: 263

---

🏁 Script executed:

```shell
# Understand the Prompt interface - what does it return and when does the channel close
rg "type hostAPISessionManager" internal/extension/host_api.go -A 10
```

Repository: compozy/agh

Length of output: 618

---

🏁 Script executed:

```shell
# Check if there's any ownership of the channel elsewhere or goroutine group management
rg "drainAgentEvents" internal/extension/host_api.go -B 5 -A 5
```

Repository: compozy/agh

Length of output: 593

---

🏁 Script executed:

```shell
# Verify that the channel could potentially stay open by checking session implementation hints
rg "Prompt.*context\.Context" internal/ -A 5 | head -40
```

Repository: compozy/agh

Length of output: 2912

---

🏁 Script executed:

```shell
# Check if there's any documentation or patterns about how long the channel might stay open
rg "eventsCh\|AgentEvent\|Prompt" internal/extension/host_api.go | head -20
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Give the event-drain goroutine an owner and shutdown path.**

The `go drainAgentEvents(eventsCh)` spawned at line 723 is fire-and-forget with no tracking mechanism. The `drainAgentEvents` function at lines 1183-1186 accepts only a channel parameter and contains a bare `for range` loop with no context cancellation support. While the channel will eventually close, the goroutine lacks explicit ownership or graceful shutdown semantics.

Per coding guidelines for `**/*.go` files:
- Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation
- No fire-and-forget goroutines — track with `sync.WaitGroup` or equivalent
- Use `select` with `ctx.Done()` in all long-running goroutine loops

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api.go` around lines 718 - 723, The goroutine spawned
for draining events is fire-and-forget and lacks cancellation/ownership; update
the call site that invokes h.sessions.Prompt (where promptCtx, eventsCh are
created) to pass a derived cancellable context (e.g., ctxWithOwner) and a
wait-group or tracker, then change drainAgentEvents to accept (ctx
context.Context, wg *sync.WaitGroup, eventsCh <-chan AgentEvent) and use defer
wg.Done() plus a select loop that listens on ctx.Done() and the eventsCh to exit
gracefully; ensure the caller increments the wg before starting the goroutine
and cancels the derived context when the session/owner shuts down so the drain
goroutine has an explicit shutdown path and is tracked.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Invalid`
- Notes:
  This drain goroutine is intentionally owned by the prompt stream returned from `session.Manager.Prompt`. The manager closes that output channel when the prompt terminates or the prompt context ends, and the Host API must keep draining it because nothing else consumes the stream after submission.
  Adding an independent cancellation path here would risk removing the only receiver while `pumpPrompt` is still sending later events, which could block prompt completion instead of improving shutdown semantics. Closed with no code change after tracing `session.Manager.Prompt` and `pumpPrompt`.
