---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: extensions/bridges/teams/provider.go
line: 45
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGQ,comment:PRRC_kwDOR5y4QM69ZeEr
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
git ls-files extensions/bridges/teams/ | head -20
```

Repository: compozy/agh

Length of output: 278

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | head -100
```

Repository: compozy/agh

Length of output: 3575

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '40,60p'
```

Repository: compozy/agh

Length of output: 745

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '2120,2140p'
```

Repository: compozy/agh

Length of output: 835

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '2155,2175p'
```

Repository: compozy/agh

Length of output: 758

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '2325,2345p'
```

Repository: compozy/agh

Length of output: 730

---

🏁 Script executed:

```shell
rg -n "func validatedTeamsCredentialedURL" extensions/bridges/teams/provider.go
```

Repository: compozy/agh

Length of output: 141

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '2345,2365p'
```

Repository: compozy/agh

Length of output: 838

---

🏁 Script executed:

```shell
rg -n "CheckRedirect" extensions/bridges/teams/provider.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "type teamsBotClient" extensions/bridges/teams/provider.go -A 10
```

Repository: compozy/agh

Length of output: 335

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '2794,2850p'
```

Repository: compozy/agh

Length of output: 1963

---

🏁 Script executed:

```shell
rg -n "httpClient.*=" extensions/bridges/teams/provider.go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "teamsAuthHTTPClient" extensions/bridges/teams/provider.go
```

Repository: compozy/agh

Length of output: 218

---

🏁 Script executed:

```shell
rg -n "teamsBotClient.*httpClient" extensions/bridges/teams/provider.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "apiFactory" extensions/bridges/teams/provider.go | head -20
```

Repository: compozy/agh

Length of output: 278

---

🏁 Script executed:

```shell
rg -n "func.*newTeamsBotClient" extensions/bridges/teams/provider.go -A 15
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '310,330p'
```

Repository: compozy/agh

Length of output: 815

---

🏁 Script executed:

```shell
rg -n "teamsAPI" extensions/bridges/teams/provider.go | head -20
```

Repository: compozy/agh

Length of output: 293

---

🏁 Script executed:

```shell
rg -n "type teamsAPI" extensions/bridges/teams/provider.go -A 10
```

Repository: compozy/agh

Length of output: 347

---

🏁 Script executed:

```shell
cat -n extensions/bridges/teams/provider.go | sed -n '2306,2365p'
```

Repository: compozy/agh

Length of output: 2010

---



</details>

**Block or revalidate redirects on these credentialed requests.**

`validatedTeamsCredentialedURL` validates only the initial URL's hostname. These calls use Go's default redirect policy, which follows 307/308 redirects that preserve the request body. An allowed host (e.g., `login.botframework.com`) can redirect to an arbitrary destination, forwarding client credentials and bypassing the host allowlist. Reject all redirects via `CheckRedirect` on the HTTP clients, or validate every `Location` header with `validTeamsCredentialedURL` before following it.

Affects lines 2134, 2168, and 2346.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/teams/provider.go` at line 45, The client
teamsAuthHTTPClient currently allows automatic redirects which can forward
credentialed bodies; update its redirect handling to block or validate
redirects: set teamsAuthHTTPClient.CheckRedirect to a function that returns
http.ErrUseLastResponse to reject all redirects, or implement logic that calls
validatedTeamsCredentialedURL on each redirect Location and only allows the
redirect when it returns true. Apply this change where teamsAuthHTTPClient is
used (and any other HTTP clients issuing credentialed requests) so credentialed
POST/PUT bodies are never blindly forwarded to unvalidated hosts.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: Teams validates the initial metadata/JWKS/token URLs but still uses default redirect-following clients for those requests. That allows host-allowlist bypass via redirect and can either leak credentialed token requests or trust attacker-controlled metadata/JWKS responses.
- Fix plan: reject redirects for Teams credentialed/validated HTTP fetches and add targeted tests in `extensions/bridges/teams/provider_test.go`.
