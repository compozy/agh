---
status: resolved
file: internal/mcp/auth/service.go
line: 49
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLic,comment:PRRC_kwDOR5y4QM67SmDh
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`Does Go's `net/http` `http.DefaultClient` have a non-zero timeout by default, or can requests block indefinitely unless a timeout/deadline is configured?`

💡 Result:

Yes. By default, Go’s http.DefaultClient has a zero Client.Timeout, which means “no timeout,” so requests can block indefinitely unless you configure a timeout or cancel via request context. Go’s net/http defines DefaultClient as var DefaultClient = &Client{} (the zero-value Client). The Client.Timeout field documentation says “A Timeout of zero means no timeout.” Therefore, when using http.DefaultClient (e.g., via http.Get / http.DefaultClient.Do), there is no built-in overall request deadline. To prevent indefinite blocking, set either: - http.Client{Timeout: ...} (overall timeout), and/or - a deadline/cancellation via Request context (e.g., http.NewRequestWithContext and context.WithTimeout/WithDeadline).

Citations:

- 1: https://go.dev/src/net/http/client.go
- 2: https://pkg.go.dev/net/http@go1.25.1
- 3: https://pkg.go.dev/net/http

---

🏁 Script executed:

```shell
cat -n internal/mcp/auth/service.go | head -60
```

Repository: compozy/agh

Length of output: 1756

---

🏁 Script executed:

```shell
rg -n "discoverMetadata|code exchange|refresh|revoke" internal/mcp/auth/service.go -A 5 -B 2
```

Repository: compozy/agh

Length of output: 4333

---

🏁 Script executed:

```shell
rg -n "^func.*discoverMetadata" internal/mcp/auth/service.go -A 30
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "discoverMetadata" internal/mcp/auth/service.go | grep -E "func|:="
```

Repository: compozy/agh

Length of output: 236

---

🏁 Script executed:

```shell
rg -n "func discoverMetadata" internal/mcp/auth/ -A 40
```

Repository: compozy/agh

Length of output: 2804

---

🏁 Script executed:

```shell
rg -n "defaultMetadataClientTimeout" internal/mcp/auth/ -B 2 -A 2
```

Repository: compozy/agh

Length of output: 708

---



</details>

**Configure the HTTP client with an explicit timeout.**

`NewService` sets `client: http.DefaultClient`, which has zero timeout by default. This means metadata discovery (in `Login`, `Refresh`, `Logout`), code exchange, token refresh, and revocation requests can all block indefinitely on network stalls. The 10-second fallback in `discoverMetadata` only applies when the client is `nil`, which never happens since `NewService` always assigns a non-nil client.

Add a timeout to the HTTP client in `NewService` (via functional options or constructor default) instead of relying on the dead-code fallback.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/mcp/auth/service.go` around lines 27 - 49, NewService currently
assigns http.DefaultClient (which has no timeout) causing network calls to
potentially hang; change the constructor to set service.client to a new
http.Client with a sensible timeout (e.g., 10s) as the default unless a
ServiceOption overrides it: remove reliance on http.DefaultClient and the
dead-code fallback in discoverMetadata, initialize service.client =
&http.Client{Timeout: 10 * time.Second} in NewService (or provide a
ServiceOption to configure the timeout) so Login/Refresh/Logout metadata
discovery, code exchange, refresh and revocation requests cannot block
indefinitely.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `NewService` assigns `http.DefaultClient`, whose zero timeout allows metadata, token, refresh, and revocation requests to block indefinitely. The timeout fallback in `discoverMetadata` is unreachable for normal services because the constructor always installs a non-nil client.
- Fix approach: default the service HTTP client to a dedicated `http.Client` with the package timeout while preserving `WithHTTPClient` overrides.
