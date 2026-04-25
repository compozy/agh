---
status: resolved
file: internal/cli/mcp_auth.go
line: 378
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59lV1p,comment:PRRC_kwDOR5y4QM67Ri0_
---

# Issue 008: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's find and examine the file
wc -l internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 84

---

🏁 Script executed:

```shell
# Look at the specific lines mentioned in the review
sed -n '361,403p' internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 1566

---

🏁 Script executed:

```shell
# Search for serveMCPAuthCallback function to see how it uses parsed.Path
rg -n "serveMCPAuthCallback" internal/cli/mcp_auth.go -A 15
```

Repository: compozy/agh

Length of output: 1158

---

🏁 Script executed:

```shell
# Search for listenForMCPAuthCallback to understand the full context
rg -n "listenForMCPAuthCallback" internal/cli/mcp_auth.go -B 5
```

Repository: compozy/agh

Length of output: 489

---

🏁 Script executed:

```shell
# Check how HandleFunc is being called with parsed.Path
rg -n "HandleFunc" internal/cli/mcp_auth.go -B 3 -A 3
```

Repository: compozy/agh

Length of output: 364

---

🏁 Script executed:

```shell
# Get the complete serveMCPAuthCallback function
sed -n '390,440p' internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 1766

---

🏁 Script executed:

```shell
# Check Go's http.ServeMux behavior by searching for any tests or usage patterns
rg -n "HandleFunc.*\"\"" internal/ -A 2 -B 2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for any error handling or validation of parsed.Path
rg -n "parsed\.Path" internal/cli/mcp_auth.go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 259

---

🏁 Script executed:

```shell
# Check the complete listenForMCPAuthCallback function
sed -n '348,380p' internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 1301

---

🏁 Script executed:

```shell
# Check if there are any tests for these functions that show expected behavior
rg -n "listenForMCPAuthCallback\|serveMCPAuthCallback" internal/ --type go -A 10 | head -50
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for any other places where redirect-url is handled
rg -n "redirect-url\|redirectURL" internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Let's check Go's http.ServeMux behavior with empty patterns
web_search "Go http.ServeMux HandleFunc empty pattern behavior panic"
```

Repository: compozy/agh

Length of output: 106

---

🏁 Script executed:

```shell
# Check what the default case returns when redirectURL is empty
sed -n '348,365p' internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 794

---

🌐 Web query:

`Go http.ServeMux HandleFunc empty string pattern`

💡 Result:

Using an empty string ("") as the pattern in http.ServeMux.HandleFunc or Handle causes a panic with the error "http: invalid pattern".

Citations:

- 1: https://echorand.me/posts/golang-dissecting-listen-and-serve/

---

🏁 Script executed:

```shell
# Let's check if there are any other redirect URL validations or uses
rg -n "redirectURL\|redirect.url" internal/cli/mcp_auth.go -i
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Verify the exact location of both functions to confirm line numbers
grep -n "^func listenForMCPAuthCallback\|^func serveMCPAuthCallback" internal/cli/mcp_auth.go
```

Repository: compozy/agh

Length of output: 193

---



</details>

**Normalize empty callback paths before registering the handler.**

When a user provides `--redirect-url="http://127.0.0.1:8080"` (without a path), `listenForMCPAuthCallback` accepts it because it only validates the scheme and host. Later, `serveMCPAuthCallback` parses this URL and calls `mux.HandleFunc(parsed.Path, ...)` with an empty string, which causes http.ServeMux to panic with "http: invalid pattern". This crashes the login flow silently.

The function already defaults to `/callback` when `redirectURL` is empty. Apply the same normalization when an explicit URL has an empty path:

<details>
<summary>Suggested fix</summary>

```diff
    parsed, err := url.Parse(strings.TrimSpace(redirectURL))
    if err != nil || parsed.Scheme == "" || parsed.Host == "" {
        return nil, "", errors.New("cli: redirect-url must be an absolute http URL")
    }
    if parsed.Scheme != "http" {
        return nil, "", errors.New("cli: redirect-url loopback listener requires http")
    }
    if !mcpAuthLoopbackHost(parsed.Hostname()) {
        return nil, "", errors.New("cli: redirect-url loopback listener requires localhost or loopback IP")
    }
+	if parsed.Path == "" {
+		parsed.Path = "/callback"
+	}
    listener, err := listenConfig.Listen(ctx, "tcp", parsed.Host)
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	parsed, err := url.Parse(strings.TrimSpace(redirectURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, "", errors.New("cli: redirect-url must be an absolute http URL")
	}
	if parsed.Scheme != "http" {
		return nil, "", errors.New("cli: redirect-url loopback listener requires http")
	}
	if !mcpAuthLoopbackHost(parsed.Hostname()) {
		return nil, "", errors.New("cli: redirect-url loopback listener requires localhost or loopback IP")
	}
	if parsed.Path == "" {
		parsed.Path = "/callback"
	}
	listener, err := listenConfig.Listen(ctx, "tcp", parsed.Host)
	if err != nil {
		return nil, "", fmt.Errorf("cli: listen for MCP auth callback: %w", err)
	}
	if parsed.Port() == "0" {
		parsed.Host = listener.Addr().String()
	}
	return listener, parsed.String(), nil
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/mcp_auth.go` around lines 361 - 378, listenForMCPAuthCallback
currently accepts an explicit redirectURL with an empty path which later causes
serveMCPAuthCallback to call mux.HandleFunc(parsed.Path, ...) with an empty
pattern and panic; fix by normalizing the parsed URL path before returning from
listenForMCPAuthCallback: after parsing redirectURL (and after possibly
replacing the host when port was "0") ensure parsed.Path is non-empty (set to
"/callback" when parsed.Path == "" or parsed.Path == "/") so downstream
serveMCPAuthCallback receives a valid path string for mux.HandleFunc; keep using
parsed.String() as the returned callback URL.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `listenForMCPAuthCallback` accepts an absolute loopback URL with an empty path, then `serveMCPAuthCallback` registers `parsed.Path` directly with `http.ServeMux`. An empty pattern panics. Normalize explicit redirect URLs with an empty path to `/callback` and add regression coverage.
