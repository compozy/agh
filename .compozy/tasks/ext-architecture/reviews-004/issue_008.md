---
status: resolved
file: internal/config/mcpjson.go
line: 55
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q89A,comment:PRRC_kwDOR5y4QM6200jS
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Detect duplicate server names after trimming.**

Two JSON keys like `" foo "` and `"foo"` normalize to the same `MCPServer.Name` and currently pass validation, which can create ambiguous overrides later. Reject duplicates after normalization during parse/validate.



Also applies to: 100-115

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/mcpjson.go` around lines 49 - 55, During MCP JSON parsing
validate that server names are unique after normalization
(trim/space-normalization) and reject duplicates: after building the combined
servers slice (the result of sortedMCPJSONServers and OverrideMCPServers)
iterate through servers, compute the normalized key from server.Name (trim and
any canonicalization used elsewhere), keep a map[string]int of seen names and if
a normalized name already exists return a descriptive error (use the same
error-wrapping style as the existing Validate/Validate error path) so names like
" foo " and "foo" are treated as duplicates; update the logic near the loop that
calls server.Validate (and the analogous block around lines 100-115) to perform
this duplicate check before accepting the config.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `sortedMCPJSONServers` trims JSON object keys into `MCPServer.Name`, but `ParseMCPServersJSON` never checks whether multiple original keys normalize to the same trimmed name.
- Inputs like `" foo "` and `"foo"` therefore survive parsing as duplicate logical server names, which makes later override behavior ambiguous.
- Fix approach: validate uniqueness after normalization and before per-server validation, then add a parsing test that rejects duplicate normalized names.
