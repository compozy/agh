---
status: resolved
file: internal/mcp/auth/service.go
line: 529
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59lV10,comment:PRRC_kwDOR5y4QM67Ri1L
---

# Issue 019: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`Does RFC 7636 specify that OAuth servers support S256 PKCE by default if code_challenge_methods_supported is not present in metadata?`

💡 Result:

No, RFC 7636 does not specify that OAuth servers must support S256 PKCE by default if code_challenge_methods_supported is not present in metadata. RFC 7636 defines the PKCE protocol, stating that servers supporting PKCE are required to support S256 (section 7.2), and the code_challenge_method parameter defaults to "plain" if omitted in requests (section 4.3). However, it does not mention server metadata or code_challenge_methods_supported. This metadata parameter is defined in RFC 8414 (OAuth 2.0 Authorization Server Metadata, section 2), where it is OPTIONAL, and explicitly states: "If omitted, the authorization server does not support PKCE." Therefore, absence of the parameter signals no PKCE support, not default S256 support.

Citations:

- 1: https://datatracker.ietf.org/doc/html/rfc7636
- 2: https://www.rfc-editor.org/rfc/rfc7636.txt
- 3: https://rfcinfo.com/rfc-7636/4/
- 4: https://tools.ietf.org/html/rfc8414
- 5: https://rfcinfo.com/rfc-8414/2
- 6: https://www.rfc-editor.org/rfc/rfc8414
- 7: https://rfcinfo.com/rfc-7636/7/

---



</details>

**Fix S256 PKCE support check to follow RFC 8414: return `false` for empty methods list.**

RFC 8414 (OAuth 2.0 Authorization Server Metadata, section 2) explicitly states that when `code_challenge_methods_supported` is omitted from the authorization server metadata, "the authorization server does not support PKCE." The current implementation incorrectly returns `true` for an empty methods list, violating this specification. This can cause failures when attempting PKCE with servers that don't advertise PKCE support.

<details>
<summary>Current code</summary>

```
func supportsS256(methods []string) bool {
	if len(methods) == 0 {
		return true
	}
	for _, method := range methods {
		if strings.EqualFold(strings.TrimSpace(method), "S256") {
			return true
		}
	}
	return false
}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/mcp/auth/service.go` around lines 519 - 529, The supportsS256
function currently returns true when methods is empty, which violates RFC 8414;
change supportsS256 so that if len(methods) == 0 it returns false, and otherwise
keep the existing loop that normalizes and compares each entry using
strings.TrimSpace/strings.EqualFold to "S256" and return true only when a match
is found; ensure the function signature and behavior remain the same except for
the empty-list result.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `supportsS256` returns true when metadata omits `code_challenge_methods_supported`. OAuth metadata treats that omission as no advertised PKCE support, so login should reject missing methods instead of assuming S256. Change the empty-list result to false and add coverage.
