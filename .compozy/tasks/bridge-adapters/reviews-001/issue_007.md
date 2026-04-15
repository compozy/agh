---
status: resolved
file: extensions/bridges/gchat/provider.go
line: 1088
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwmj,comment:PRRC_kwDOR5y4QM64DQ0N
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's find and examine the file
fd -t f "provider.go" extensions/bridges/gchat/
```

Repository: compozy/agh

Length of output: 91

---

🏁 Script executed:

```shell
# Read the relevant sections of the file
cat -n extensions/bridges/gchat/provider.go | sed -n '1060,1095p'
```

Repository: compozy/agh

Length of output: 1936

---

🏁 Script executed:

```shell
# Also check the second range mentioned (1126-1142)
cat -n extensions/bridges/gchat/provider.go | sed -n '1115,1150p'
```

Repository: compozy/agh

Length of output: 2014

---

🏁 Script executed:

```shell
# Check the function signature of dispatchInboundEnvelope
rg -A 5 "func.*dispatchInboundEnvelope" extensions/bridges/gchat/provider.go
```

Repository: compozy/agh

Length of output: 370

---

🏁 Script executed:

```shell
# Check what context is available in the webhook handler function
cat -n extensions/bridges/gchat/provider.go | sed -n '1000,1070p'
```

Repository: compozy/agh

Length of output: 3231

---



</details>

**Thread request context through webhook handlers instead of using context.Background().**

Four calls to `dispatchInboundEnvelope()` use `context.Background()` at lines 1067, 1087, 1126, and 1141. This detaches dispatch work from both the HTTP request cancellation and provider shutdown. The function signature already accepts a context parameter; pass a real context (from the HTTP request or provider shutdown context) instead.

Per coding guidelines: "Pass context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/gchat/provider.go` around lines 1067 - 1088, Several calls
to dispatchInboundEnvelope are using context.Background(), detaching work from
request cancellation and shutdown; replace those context.Background() calls with
the real request or provider context (e.g., use the incoming HTTP request's
Context() — r.Context() — in the webhook handler, or the provider shutdown
context (p.ctx) where appropriate) so dispatchInboundEnvelope(ctx,
cfg.instanceID, item.Envelope) uses a cancellable context; update all four call
sites that currently pass context.Background() (the branches that handle mapped
direct messages and other webhook paths, including the branch that runs when
cfg.batcher is nil) to accept/forward the correct ctx. Ensure any helper
functions invoked by the handler also take and pass the same ctx so
cancellation/shutdown propagates.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The direct and Pub/Sub webhook handlers still pass `context.Background()` into synchronous `dispatchInboundEnvelope()` calls, and the Pub/Sub reaction mapper also uses `context.Background()` for its message lookup API call.
  - That detaches inbound processing from request cancellation and shutdown propagation even though the handler already has a real request context available.
  - Planned fix: thread the request context through all four synchronous dispatch call sites and the Pub/Sub reaction lookup, with a regression test that proves cancellation reaches the ingest path.
  - Resolution: `handleWebhookRequest()` now threads the real request context into the direct and Pub/Sub helpers, all synchronous dispatch paths use that context, and the Pub/Sub reaction lookup now uses the same cancellable context for message fetches; cancellation regression coverage was added.
  - Verification: `go test -race ./extensions/bridges/gchat -count=1` and `make verify` both passed after the fix.
