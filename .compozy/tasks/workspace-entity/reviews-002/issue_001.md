---
status: resolved
file: internal/acp/client.go
line: 222
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoBu,comment:PRRC_kwDOR5y4QM61T6G8
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Nil-safety issue when copying `AdditionalDirs`.**

When `normalized.AdditionalDirs` is `nil`, `append([]string(nil), normalized.AdditionalDirs...)` produces an empty slice `[]string{}`, not `nil`. If the wire protocol distinguishes between `null` and `[]` (e.g., JSON encoding), this could cause subtle behavioral differences. Consider preserving nil semantics:


<details>
<summary>Proposed fix</summary>

```diff
 	loadWireRequest := wireLoadSessionRequest{
 		Cwd:            loadRequest.Cwd,
 		McpServers:     loadRequest.McpServers,
-		AdditionalDirs: append([]string(nil), normalized.AdditionalDirs...),
+		AdditionalDirs: copyStringSlice(normalized.AdditionalDirs),
 		SessionID:      loadRequest.SessionId,
 	}
```

Add helper:
```go
func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s...)
}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client.go` around lines 217 - 222, The copy of AdditionalDirs in
the construction of loadWireRequest currently uses append([]string(nil),
normalized.AdditionalDirs...) which turns a nil slice into an empty non-nil
slice; preserve nil semantics by adding a helper (e.g., copyStringSlice) that
returns nil when input is nil and otherwise returns a copied slice, then replace
the append call with copyStringSlice(normalized.AdditionalDirs) when building
the wireLoadSessionRequest (loadWireRequest) so the wire protocol sees true null
vs empty array correctly.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - Verified with a small Go reproduction: `append([]string(nil), nilSlice...)` returns `nil`, not an empty slice.
  - The current `append([]string(nil), normalized.AdditionalDirs...)` therefore already preserves `nil` semantics for the wire payload.
  - No production change is needed here.
