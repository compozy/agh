---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/modelsdev.go
line: 65
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaqN,comment:PRRC_kwDOR5y4QM6-7HYp
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Reject injected clients without a timeout.**

`WithModelsDevHTTPClient` can replace the default `http.Client{Timeout: timeout}` with a zero-timeout client, so one option call removes the only outbound timeout from this source. Please validate `client.Timeout > 0` after applying options and fail construction otherwise.

<details>
<summary>Suggested guard</summary>

```diff
 	for _, option := range options {
 		if option != nil {
 			option(source)
 		}
 	}
+	if source.client == nil || source.client.Timeout <= 0 {
+		return nil, fmt.Errorf("model catalog: models.dev client timeout must be positive")
+	}
 	return source, nil
 }
```
</details>

 

As per coding guidelines "External-call timeouts in Go backend are mandatory. Outbound HTTP/network calls MUST use a client with an explicit timeout. `http.DefaultClient` is forbidden in production code paths."


Also applies to: 91-96

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/modelsdev.go` around lines 59 - 65, Reject injected
http.Clients that lack an explicit timeout: in WithModelsDevHTTPClient ensure
you only assign client if client != nil && client.Timeout > 0 (do not accept
zero-timeout clients), and in the ModelsDevSource constructor (after applying
all ModelsDevSourceOption funcs) validate that source.client is non-nil and
source.client.Timeout > 0 and return an error if not; apply the same timeout>0
guard for the other option functions noted around lines 91-96 so no option can
remove the required outbound timeout.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `WithModelsDevHTTPClient` in `internal/modelcatalog/modelsdev.go` accepts any non-nil client, including one with `Timeout == 0`.
  - That can silently remove the only explicit outbound timeout required by the models.dev source.
  - Fix plan: reject zero-timeout injected clients and validate the final configured client after options are applied.
  - Fixed in `internal/modelcatalog/modelsdev.go` with regression coverage in `internal/modelcatalog/modelsdev_test.go`, then verified with focused package tests plus `make verify`.
