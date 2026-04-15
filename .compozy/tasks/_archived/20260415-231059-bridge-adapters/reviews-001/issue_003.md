---
status: resolved
file: extensions/bridges/discord/provider.go
line: 1024
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwl3,comment:PRRC_kwDOR5y4QM64DQzR
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's examine the specific lines mentioned in the review
sed -n '1020,1065p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 2062

---

🏁 Script executed:

```shell
# Also check the stop() method and overall context management in the provider
sed -n '1,100p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 3818

---

🏁 Script executed:

```shell
# Search for the dispatchInboundEnvelope function definition
grep -n "func.*dispatchInboundEnvelope" extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 206

---

🏁 Script executed:

```shell
# Look for stop() method definition
grep -n "func.*stop\(\)" extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 93

---

🏁 Script executed:

```shell
# Check if there's a provider-level context or cancellation setup
grep -n "ctx\|cancel\|context" extensions/bridges/discord/provider.go | head -40
```

Repository: compozy/agh

Length of output: 2463

---

🏁 Script executed:

```shell
# Examine the stop() method and how it handles shutdown
sed -n '497,530p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 1090

---

🏁 Script executed:

```shell
# Check the overall structure: look for any context.WithCancel or cancellable root
grep -n "WithCancel\|rootCtx\|providerCtx\|cancelCtx" extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at how the provider is initialized to understand the context lifecycle
sed -n '305,350p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 1699

---

🏁 Script executed:

```shell
# Check the entire dispatchAsyncInboundEnvelope function and its usage
grep -n "dispatchAsyncInboundEnvelope" extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 320

---

🏁 Script executed:

```shell
# Get the full stop() method
sed -n '497,520p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 851

---

🏁 Script executed:

```shell
# Check where the webhook handler is called from - need to understand the context chain
grep -n "handleWebhookEvent\|handleBridgesDeliver" extensions/bridges/discord/provider.go | head -20
```

Repository: compozy/agh

Length of output: 156

---

🏁 Script executed:

```shell
# Get the full handler function that contains lines 1022-1024
sed -n '1000,1050p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 2806

---

🏁 Script executed:

```shell
# Look at handleShutdown to see if it waits for the WaitGroup
grep -n "handleShutdown" extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 144

---

🏁 Script executed:

```shell
# Get the handleShutdown method
sed -n '460,490p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 887

---

🏁 Script executed:

```shell
# Check how the provider is used - understand the context lifecycle better
grep -n "func main\|newDiscordProvider\|p.stop" extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 263

---

🏁 Script executed:

```shell
# Get the full dispatchInboundEnvelope function to see how context is used
sed -n '1084,1120p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 1583

---

🏁 Script executed:

```shell
# Check if there's a provider root context or if contexts are properly managed elsewhere
sed -n '51,75p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 866

---

🏁 Script executed:

```shell
# Verify the exact lines in question one more time with more context
sed -n '1018,1045p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 1610

---

🏁 Script executed:

```shell
# Check the function signature of handleBridgesDeliver to see if it receives context
sed -n '399,415p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 739

---

🏁 Script executed:

```shell
# Check how handleBridgesDeliver is defined - look at the full signature
grep -B 5 "func (p \*discordProvider) handleBridgesDeliver" extensions/bridges/discord/provider.go | head -20
```

Repository: compozy/agh

Length of output: 139

---

🏁 Script executed:

```shell
# Look for the request context in handleBridgesDeliver
sed -n '399,440p' extensions/bridges/discord/provider.go | cat -n
```

Repository: compozy/agh

Length of output: 1919

---

</details>

**Propagate request/provider cancellation into inbound dispatch and async operations.**

The webhook handler receives a context parameter but discards it in favor of `context.Background()` at lines 1022-1024 and 1038-1040. Additionally, `dispatchAsyncInboundEnvelope` (lines 1047-1063) creates independent contexts derived from `context.Background()` rather than from a provider-owned cancellable root. The goroutine only checks `p.stopCh` once at startup; once execution proceeds past that check, shutdown cannot cancel the in-flight `dispatchInboundEnvelope` call, allowing work to outlive provider shutdown and potentially blocking graceful termination.

Use the available `ctx` parameter at lines 1022-1024 and 1038-1040, and derive contexts in `dispatchAsyncInboundEnvelope` from a provider-owned cancellable root (canceled in `stop()`) rather than from `context.Background()`.

Also applies to: 1038-1040, 1047-1063

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider.go` around lines 1022 - 1024, The handler
currently ignores the incoming ctx and uses context.Background() for inbound
processing and async work; replace those with the provided ctx when calling
dispatchInboundEnvelope and when launching goroutines so request cancellation
propagates. In dispatchAsyncInboundEnvelope, stop deriving contexts from
context.Background(); instead derive them from a provider-owned cancellable root
context (e.g., p.ctx/p.cancel or add one if missing) that is canceled in stop(),
and ensure the goroutine selects on both p.stopCh (or p.ctx.Done()) and
ctx.Done() before/while calling dispatchInboundEnvelope so in-flight work is
cancelable during shutdown.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The Discord webhook handler still dispatches direct message and reaction envelopes with `context.Background()` at the non-batched call sites.
  - That severs the ingest call from request cancellation and from server shutdown-driven request cancellation, despite `dispatchInboundEnvelope()` already accepting a context.
  - Planned fix: thread the real request context through the synchronous webhook dispatch path and add a regression test that observes cancellation propagation.
  - Resolution: `handleWebhookRequest()` now passes the incoming `*http.Request` through to `handleEventWebhook()`, and the synchronous non-batched Discord dispatch path uses `r.Context()` instead of `context.Background()`; a canceled-context regression test now covers the ingest path.
  - Verification: `go test ./extensions/bridges/discord -count=1` and `make verify` both passed after the fix.
