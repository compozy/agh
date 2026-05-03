---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: extensions/bridges/linear/api.go
line: 298
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGP,comment:PRRC_kwDOR5y4QM69ZeEq
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Find and examine the Linear API file
fd "api.go" extensions/bridges/linear/
wc -l extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 125

---

🏁 Script executed:

```shell
# Read the specific lines mentioned in the review comment
sed -n '290,300p' extensions/bridges/linear/api.go
sed -n '363,380p' extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 1047

---

🏁 Script executed:

```shell
# Search for the validLinearCredentialedURL function definition
rg "validLinearCredentialedURL" extensions/bridges/linear/
```

Repository: compozy/agh

Length of output: 1243

---

🏁 Script executed:

```shell
# Check for httpClient initialization and redirect policy configuration
rg -A 10 "NewClient\|httpClient\s*=" extensions/bridges/linear/api.go | head -50
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for any custom redirect handling or CheckRedirect configuration
rg -i "checkredirect\|redirect\|30[0-9]" extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Find the validLinearCredentialedURL implementation
rg -A 20 "func validLinearCredentialedURL" extensions/bridges/linear/
```

Repository: compozy/agh

Length of output: 1357

---

🏁 Script executed:

```shell
# Search for httpClient creation and configuration
rg -B 5 -A 15 "c\.httpClient\s*=" extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at the full provider.go file around the function and client initialization
sed -n '1,50p' extensions/bridges/linear/provider.go
sed -n '100,150p' extensions/bridges/linear/provider.go
```

Repository: compozy/agh

Length of output: 2854

---

🏁 Script executed:

```shell
# Check if there's a NewClient or initialization for the http client
rg -B 5 -A 10 "NewClient\|http\.Client" extensions/bridges/linear/provider.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for the Client struct definition and httpClient field
rg -B 5 -A 10 "type Client struct\|httpClient" extensions/bridges/linear/api.go | head -40
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for NewClient or client initialization function
rg -B 5 -A 20 "func New\|func newClient" extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the full context of lines 290-310 and 360-380
sed -n '280,310p' extensions/bridges/linear/api.go
sed -n '360,385p' extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 1925

---

🏁 Script executed:

```shell
# Search for any CheckRedirect or redirect policy setup
rg -i "CheckRedirect\|NoRedirect\|redirect" extensions/bridges/linear/ --type go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for the linearClient struct definition
rg -B 3 -A 20 "type linearClient struct" extensions/bridges/linear/
```

Repository: compozy/agh

Length of output: 1469

---

🏁 Script executed:

```shell
# Find where linearClient is created or initialized
rg -B 5 -A 15 "linearClient{" extensions/bridges/linear/
```

Repository: compozy/agh

Length of output: 145

---

🏁 Script executed:

```shell
# Check if httpClient is a plain http.Client or has special configuration
rg "httpClient\s*\*http\.Client\|httpClient.*Client" extensions/bridges/linear/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at the entire api.go file structure to understand httpClient setup
head -100 extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 2414

---

🏁 Script executed:

```shell
# Search for where the http.Client is created or assigned to the linearClient
rg -B 10 -A 5 "httpClient\s*:" extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for newLinearClient or any function that creates a linearClient
rg -B 5 -A 20 "func newLinearClient\|func NewLinearClient\|&linearClient\{" extensions/bridges/linear/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there are any other bridge implementations to compare redirect handling
find extensions/bridges -name "*.go" -type f | head -5
rg "type.*Client struct" extensions/bridges/ --type go
```

Repository: compozy/agh

Length of output: 778

---

🏁 Script executed:

```shell
# Look for the complete initialization of linearClient in provider.go
sed -n '1,469p' extensions/bridges/linear/provider.go | tail -100
```

Repository: compozy/agh

Length of output: 2522

---

🏁 Script executed:

```shell
# Search the entire Linear bridge directory for where linearClient is instantiated
rg -n "linearClient\s*\{" extensions/bridges/linear/provider.go
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
# Look for the initLinear or initialization function that creates the client
rg -B 15 -A 10 "linearClient\s*\*http\.Client" extensions/bridges/linear/provider.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the config initialization and see how the http.Client is provided
rg -B 20 "httpClient" extensions/bridges/linear/provider.go | head -60
```

Repository: compozy/agh

Length of output: 801

---

🏁 Script executed:

```shell
# Look at the complete provider initialization code - search around lines 200-250
sed -n '200,300p' extensions/bridges/linear/provider.go
```

Repository: compozy/agh

Length of output: 3295

---

🏁 Script executed:

```shell
# Verify the complete http.Client initialization
sed -n '220,240p' extensions/bridges/linear/provider.go
```

Repository: compozy/agh

Length of output: 704

---

🏁 Script executed:

```shell
# Check if other bridges have similar redirect handling issues
rg -B 2 -A 5 "http\.Client\{" extensions/bridges/ --type go
```

Repository: compozy/agh

Length of output: 13001

---

🏁 Script executed:

```shell
# Look at the Teams client mentioned in the review for comparison
rg -B 5 -A 10 "type teamsBotClient struct" extensions/bridges/teams/
```

Repository: compozy/agh

Length of output: 1070

---

🏁 Script executed:

```shell
# Check if there's any CheckRedirect configuration anywhere in the codebase
rg -i "checkredirect\|ErrUseLastResponse" extensions/bridges/ --type go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Verify Go's http.Client redirect behavior by looking at Go documentation
# Let's check if there are any tests that show redirect behavior expectations
find extensions/bridges -name "*test.go" -type f | xargs rg -l "redirect\|30[0-9]" 2>/dev/null | head -3
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there's any discussion of redirect handling in comments or docs
rg -i "redirect\|follow\|30[0-9]" extensions/bridges/linear/ --type go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at what happens when httpResponse is received - does it validate the response URL?
sed -n '300,320p' extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 673

---

🏁 Script executed:

```shell
# Check the full doLinearGraphQL function to see complete flow
sed -n '290,330p' extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 1483

---

🏁 Script executed:

```shell
# Check the complete OAuth token request code at 368-376
sed -n '360,385p' extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 721

---

🏁 Script executed:

```shell
# Verify if there's any response URL validation anywhere
rg -n "Request\|response\.Request\|httpResponse\.Request" extensions/bridges/linear/api.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for any CheckRedirect setup in the entire codebase
rg "CheckRedirect\|ErrUseLastResponse" . --type go 2>/dev/null
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Final verification: confirm Go http.Client default redirect behavior
# Check if there are any existing examples of CheckRedirect in Go projects
web_search "Go http.Client CheckRedirect default redirect behavior security"
```

Repository: compozy/agh

Length of output: 106

---

🌐 Web query:

`Go http.Client default redirect behavior 30x security`

💡 Result:

Go’s http.Client follows redirects by default, with a safety limit, and it implements additional redirect-related header handling that matters for 3xx security. 1) Default redirect behavior (30x) - The net/http Client’s default redirect policy is: follow up to 10 consecutive requests/redirects, then stop. - Redirect handling depends on the specific 3xx code: - 301, 302, 303: the client switches the method of the redirected request to GET (or HEAD if the original request was HEAD), and it sends no request body. - 307, 308: the client preserves the original method and (if available via Request.GetBody) the body. - You can override this via http.Client.CheckRedirect; it is invoked before following a redirect. 2) How to control/stop following redirects - Use CheckRedirect. - Common pattern to “stop on first redirect” while still returning the redirect response: client := &http.Client{ CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }, } This returns the most recent redirect response (with its body unclosed) and nil error (per net/http special case). 3) Security implications: header forwarding on redirects - net/http has specific protection around forwarding sensitive headers on cross-domain redirects. - Reported and fixed issue (CVE-2023-45289 / Go issue 65065): before the fix, redirecting to a domain that is not a subdomain match or exact match of the initial domain could cause incorrect forwarding of sensitive headers such as Authorization or Cookie. This created a risk that a maliciously crafted redirect could cause sensitive headers to be unexpectedly forwarded. Practical security takeaway: - Treat attacker-controlled Location headers as untrusted input. - Use CheckRedirect (and/or additional validation in your handler) when you cannot guarantee the redirect target. - Ensure you’re on Go versions that include the fixes for redirect-related header/cookie issues (the Go security report for CVE-2023-45289 indicates the affected ranges). 4) Why people often see “redirect loops” or “unexpected EOF” - Because the default behavior will follow redirect chains until the limit (10) is hit, a misconfigured server (or hostile responder) that issues a redirect cycle can force repeated requests until the client stops. - If you need the Location itself and don’t want the client to follow, stop redirects with CheckRedirect and read resp.Location from the redirect response. If you tell me your Go version and whether you’re using Authorization/Cookie (or custom headers) on the initial request, I can suggest the safest CheckRedirect implementation for your scenario.

Citations:

- 1: https://go.dev/src/net/http/client.go
- 2: https://pkg.go.dev/net/http
- 3: https://go.dev/issue/65065
- 4: https://pkg.go.dev/vuln/GO-2024-2600
- 5: https://cvebase.io/cve/CVE-2023-45289
- 6: https://github.com/golang/go/issues/42832

---



</details>

**URL allowlist bypass via HTTP redirects on credentialed requests**

`validLinearCredentialedURL()` validates the initial request URL, but `http.Client` follows 30x redirects by default. A redirect from `api.linear.app` or a permitted loopback endpoint to an untrusted host will send the Bearer token (GraphQL) or OAuth client credentials (token endpoint) to the attacker. For 307/308 redirects, the POST body is preserved; for 301/302/303, the method switches to GET but sensitive headers are still forwarded.

Set `CheckRedirect` on the `http.Client` to deny redirects, or validate each redirect target before following it.

Also applies to lines 368–376.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/linear/api.go` around lines 295 - 298, The credentialed
HTTP requests currently validate the initial URL with
validLinearCredentialedURL() but then use an http.Client that can follow
redirects, exposing bearer tokens or client credentials; update the code paths
that build requests (e.g., where http.NewRequestWithContext(...) is called and
the client used for GraphQL/token calls) to configure the
http.Client.CheckRedirect to either deny redirects or validate each redirect
target with validLinearCredentialedURL() before following; apply the same fix
for the other request block referenced around the code handling lines 368–376 so
no redirect can send sensitive headers to untrusted hosts.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `linearClient` validates only the initial credentialed URL, but both GraphQL and OAuth requests use the injected `http.Client` redirect policy unchanged. Go follows redirects by default, so a trusted endpoint could redirect bearer-token or client-secret requests to an untrusted host.
- Fix plan: enforce a no-redirect policy for Linear credentialed requests regardless of the injected client, then add redirect regression coverage in `extensions/bridges/linear/provider_test.go`.
