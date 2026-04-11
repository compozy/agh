---
status: resolved
file: internal/api/core/session_stream.go
line: 127
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrWJ,comment:PRRC_kwDOR5y4QM62twbP
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Add justification comments for ignored `WriteSSE` errors.**

Lines 102, 119, and 126 ignore `WriteSSE` errors with `_`. Per coding guidelines, every error must be handled or have a written justification. While ignoring these may be intentional (client disconnect means no recovery is possible), adding brief comments clarifies the intent.


<details>
<summary>📝 Suggested justification comments</summary>

```diff
 	if pollErr != nil {
-		_ = WriteSSE(writer, SSEMessage{
+		// Best-effort error notification; client may have disconnected.
+		_ = WriteSSE(writer, SSEMessage{
 			Name: "error",
 			Data: contract.ErrorPayload{Error: pollErr.Error()},
 		})
 		return afterSequence, info, true
 	}
```

Apply similar comments at lines 119 and 126.
</details>

As per coding guidelines: "Never ignore errors with _ — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/session_stream.go` around lines 100 - 127, The three
occurrences where the WriteSSE or writeSessionStoppedEvent return values are
discarded (the calls to WriteSSE after pollErr and statusErr, and the `_ =
h.writeSessionStoppedEvent(writer, latest)` call) must include a brief comment
justifying the ignored error per guidelines; update each site (the WriteSSE
calls handling pollErr and statusErr, and the writeSessionStoppedEvent call)
with a one-line comment explaining this is intentional/unrecoverable (e.g.,
client disconnect or SSE stream closed) so the error cannot be handled or
retried.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: The three ignored SSE write errors are intentional best-effort paths, but the current code drops them without documenting why that is safe. That conflicts with the workspace rule that ignored errors must be justified at the call site.
- Fix approach: Add short comments at each ignored write explaining that the stream may already be closed and there is no meaningful recovery path.
