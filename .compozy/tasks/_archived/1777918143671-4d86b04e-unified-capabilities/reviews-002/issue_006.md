---
status: resolved
file: internal/api/udsapi/prompt.go
line: 30
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyx0,comment:PRRC_kwDOR5y4QM654NoY
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify prompt lifecycle safety with detached request context.
set -euo pipefail

echo "== Prompt interface + implementations =="
rg -nP --type=go -C3 '\bPrompt\s*\(\s*ctx\s+context\.Context'

echo
echo "== Prompt event channel creation/send sites =="
rg -nP --type=go -C4 'make\(\s*chan\s+|events\s*<-|<-events'

echo
echo "== Cancellation/unsubscribe handling around sends =="
rg -nP --type=go -C6 'ctx\.Done\(\)|select\s*\{'

echo
echo "== Coverage for disconnect + detached prompt context =="
rg -nP --type=go -C3 'DetachesPromptContext|CancelPrompt|request\.Context\(\)\.Done\(\)'
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
cat -n internal/session/manager_prompt.go | head -150
```

Repository: compozy/agh

Length of output: 5245

---

🏁 Script executed:

```shell
cat -n internal/acp/client.go | sed -n '449,550p'
```

Repository: compozy/agh

Length of output: 3714

---

🏁 Script executed:

```shell
rg -A 50 'func \(m \*Manager\) pumpPrompt' internal/session/manager_prompt.go
```

Repository: compozy/agh

Length of output: 1408

---

🏁 Script executed:

```shell
rg -A 50 'func \(d \*Driver\) runPrompt' internal/acp/client.go | head -80
```

Repository: compozy/agh

Length of output: 1425

---

🏁 Script executed:

```shell
rg -A 15 'normalized = m.preparePromptEvent' internal/session/manager_prompt.go | head -25
```

Repository: compozy/agh

Length of output: 648

---

🏁 Script executed:

```shell
rg 'out <-' internal/session/manager_prompt.go -A 2 -B 2
```

Repository: compozy/agh

Length of output: 123

---

🏁 Script executed:

```shell
rg -B 5 -A 20 'select {' internal/session/manager_prompt.go | grep -A 20 'out <-'
```

Repository: compozy/agh

Length of output: 134

---

🏁 Script executed:

```shell
# Check if there are any explicit cancellations of the detached context or internal lifecycle contexts
rg 'context.WithCancel|context.WithTimeout' internal/session/manager_prompt.go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -A 25 'out := make.*acp.AgentEvent' internal/session/manager_prompt.go
```

Repository: compozy/agh

Length of output: 933

---

🏁 Script executed:

```shell
# Check the entire pumpPrompt to see the full loop structure
rg -A 80 'func \(m \*Manager\) pumpPrompt' internal/session/manager_prompt.go | tail -60
```

Repository: compozy/agh

Length of output: 1528

---



</details>

**Remove `context.WithoutCancel()` or ensure prompt producer cancellation is tied to request lifecycle**

The detached context prevents `pumpPrompt()` from receiving cancellation when the HTTP client disconnects. Even though `pumpPrompt` guards the output channel send with `select { case out <- normalized: case <-ctx.Done(): }`, the detached context's `Done()` channel never fires on request cancellation. If the output channel buffer fills and the consumer stops reading, the producer goroutine blocks indefinitely waiting on the detached context.

Either pass the request context directly, or ensure an internal lifecycle context cancellation is tied to request completion.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/prompt.go` around lines 29 - 30, The code creates a
detached context via context.WithoutCancel (promptCtx) before calling
h.Sessions.Prompt which prevents pumpPrompt's ctx.Done() from firing on HTTP
client disconnects; change to pass the request context directly (use
c.Request.Context()) or derive a child context that is cancelled when the
request ends, and update the call to h.Sessions.Prompt to use that context
(replace promptCtx usage). Ensure the cancellation propagation also reaches the
producer goroutine used by pumpPrompt so its select { case out <- ...; case
<-ctx.Done(): } unblocks on client disconnect.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `udsapi` has the same detached-context leak as `httpapi`: request cancellation no longer reaches `Sessions.Prompt`, so the producer can continue after the SSE consumer exits.
- Fix plan: use a detached-but-cancelable prompt context and cancel it on every return path. Update the existing coverage in `internal/api/udsapi/handlers_test.go` so the request-disconnect test verifies explicit prompt cancellation instead of permanent detachment.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
