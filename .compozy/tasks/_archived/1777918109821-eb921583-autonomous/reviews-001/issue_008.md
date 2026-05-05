---
status: resolved
file: internal/api/core/agent_identity.go
line: 29
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsF,comment:PRRC_kwDOR5y4QM67YHCe
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Return `503` when the session service is not configured.**

`api: session service is not configured` is a dependency-availability problem, so mapping it to `500` makes `AgentMe` inconsistent with `AgentSpawn`, which already reports missing runtime services as unavailable.



<details>
<summary>Suggested fix</summary>

```diff
+var errAgentIdentityUnavailable = errors.New("api: session service is not configured")
+
 // StatusForAgentIdentityError maps agent identity failures to transport statuses.
 func StatusForAgentIdentityError(err error) int {
 	switch {
 	case err == nil:
 		return http.StatusOK
+	case errors.Is(err, errAgentIdentityUnavailable):
+		return http.StatusServiceUnavailable
 	case errors.Is(err, agentidentity.ErrIdentityUnauthorized):
 		return http.StatusForbidden
 	case errors.Is(err, agentidentity.ErrIdentityRequired),
 		errors.Is(err, agentidentity.ErrIdentityMismatch),
 		errors.Is(err, agentidentity.ErrIdentityStale):
@@
 func (h *BaseHandlers) resolveAgentCaller(
 	ctx context.Context,
 	credentials agentidentity.Credentials,
 	action string,
 ) (agentidentity.Caller, error) {
 	if h == nil || h.Sessions == nil {
-		return agentidentity.Caller{}, errors.New("api: session service is not configured")
+		return agentidentity.Caller{}, errAgentIdentityUnavailable
 	}
 	return agentidentity.Resolve(ctx, agentidentity.ResolveOptions{
```
</details>


Also applies to: 64-66

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/agent_identity.go` around lines 17 - 29,
StatusForAgentIdentityError currently maps the "session service not configured"
dependency error to 500; update StatusForAgentIdentityError to return
http.StatusServiceUnavailable (503) for that case by checking the
session-service-specific sentinel error (e.g., errors.Is(err,
agentidentity.ErrSessionServiceNotConfigured) or, if no sentinel exists,
matching err.Error() == "api: session service is not configured"); also apply
the same 503 mapping in the other similar handler referenced (the block around
the other occurrence at lines 64-66).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `resolveAgentCaller` returns a plain `"api: session service is not configured"` error when the handler has no session service, so `StatusForAgentIdentityError` falls through to HTTP 500. This is a dependency availability failure and should match other agent runtime-service failures as HTTP 503. Fix by introducing a sentinel unavailable error and mapping it, plus lookup-unavailable identity errors, to `http.StatusServiceUnavailable`.
