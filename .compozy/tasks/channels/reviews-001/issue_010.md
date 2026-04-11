---
status: resolved
file: internal/api/httpapi/server.go
line: 258
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLf,comment:PRRC_kwDOR5y4QM623eI4
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Require `ChannelService` here or skip registering the channel handlers.**

`New()` now wires `server.channels` into the handler graph, but the constructor still allows it to be nil. That means the HTTP server can start successfully while every `/api/channels` request fails later at runtime instead of failing fast during boot.

<details>
<summary>Possible fix</summary>

```diff
 	if server.observer == nil {
 		return nil, errors.New("httpapi: observer is required")
 	}
+	if server.channels == nil {
+		return nil, errors.New("httpapi: channel service is required")
+	}
 	if server.workspaces == nil {
 		return nil, errors.New("httpapi: workspace resolver is required")
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if server.observer == nil {
		return nil, errors.New("httpapi: observer is required")
	}
	if server.channels == nil {
		return nil, errors.New("httpapi: channel service is required")
	}
	if server.workspaces == nil {
		return nil, errors.New("httpapi: workspace resolver is required")
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/server.go` around lines 255 - 258, The handler wiring
currently injects server.channels into handlerConfig (server.handlers =
newHandlers(handlerConfig{...})) while New() still permits server.channels to be
nil; change New() or the handler registration so that if server.channels (the
ChannelService) is nil you either (a) return an error from New() to fail fast,
or (b) avoid registering the channel-related handlers inside
newHandlers/handlerConfig (skip the /api/channels setup) and log a clear
warning; locate the code that constructs handlerConfig and/or newHandlers and
add a nil-check for server.channels (ChannelService) to enforce one of the two
behaviors consistently.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The current HTTP handler graph intentionally treats `ChannelService` as optional: every channel handler checks `h.channelService()` and returns an explicit `503` when channels are not configured.
  - Failing `httpapi.New()` here would be a semantics change, not a bug fix, and it would break the existing non-channel server construction paths that intentionally omit the channel service.
  - Resolution: Closed as invalid after code inspection; the optional-channel-service behavior is deliberate and `make verify` passed without changing it.
