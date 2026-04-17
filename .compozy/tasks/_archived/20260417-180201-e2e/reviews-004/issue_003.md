---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 558
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM570BH5,comment:PRRC_kwDOR5y4QM646Fwc
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Inspect the validation helpers and the concrete validation error contract.
rg -n -C4 'func validateAndEncode(Tool|MCPServer)|ErrValidation|type .*Validation' internal/daemon internal/resources
```

Repository: compozy/agh

Length of output: 40000

---

🏁 Script executed:

```shell
sed -n '590,610p' internal/daemon/tool_mcp_resources_test.go
```

Repository: compozy/agh

Length of output: 893

---



</details>

**Remove string-based error message assertions; use `errors.Is()` for sentinel validation.**

The tool branch correctly uses `errors.Is(err, resources.ErrValidation)` but redundantly parses error message text with `strings.Contains()`. The MCP branch omits the sentinel check entirely and relies only on message text. Since `resources.ErrValidation` is a wrapped sentinel with contextual messages, tests should verify the error type via `errors.Is()` and remove text parsing assertions.

<details>
<summary>Code location: tool branch (lines 546-558) and MCP branch (lines 598-607)</summary>

```
// Tool branch (lines 553-556): Has sentinel check but still parses text
if !errors.Is(err, resources.ErrValidation) {
    t.Fatalf("validateAndEncodeTool(invalid) error = %v, want %v", err, resources.ErrValidation)
}
if !strings.Contains(err.Error(), "tool.name is required") {  // Remove this
    t.Fatalf("validateAndEncodeTool(invalid) error = %v, want tool.name validation context", err)
}

// MCP branch (lines 602-605): Missing sentinel check, only text parsing
if !strings.Contains(err.Error(), "config: validate mcp resource spec") {  // Add: errors.Is(err, resources.ErrValidation) first
    t.Fatalf("validateAndEncodeMCPServer(invalid) error = %v, want mcp resource spec context", err)
}
if !strings.Contains(err.Error(), "mcp_server.command is required") {  // Remove these text checks
    t.Fatalf("validateAndEncodeMCPServer(invalid) error = %v, want missing command validation", err)
}
```
</details>

Per coding guidelines: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings" and test requirements: "MUST have specific error assertions (ErrorContains, ErrorAs)."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 546 - 558, Replace
string-based error assertions with sentinel checks: in the validateAndEncodeTool
test remove the strings.Contains(err.Error(), "tool.name is required") assertion
and keep/ensure the errors.Is(err, resources.ErrValidation) check; in the MCP
test add an errors.Is(err, resources.ErrValidation) assertion before any
contextual checks and remove any strings.Contains comparisons that check for
specific message text (e.g., "mcp_server.command is required"); if you need to
assert context use errors.As to extract the wrapped error type instead of
comparing error strings, referencing validateAndEncodeTool,
validateAndEncodeMCPServer, and resources.ErrValidation to locate the assertions
to change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The review assumption does not match the current production contract for MCP server resource validation.
  - `validateAndEncodeMCPServer()` delegates to `aghconfig.NewMCPServerResourceCodec()`, and `validateMCPServerSpec()` wraps `MCPServer.Validate(...)` with context only; `MCPServer.Validate(...)` returns plain field-path errors and does **not** wrap `resources.ErrValidation`.
  - I verified this by changing the test to assert `errors.Is(err, resources.ErrValidation)` and running the focused test; it failed with `resources: validate "mcp_server" spec: config: validate mcp resource spec: mcp_server.command is required`, which confirms the sentinel is absent on this path today.
  - Removing the existing message-context assertions in this file would weaken coverage without fixing a real bug, and broadening the MCP resource validation contract would require production changes outside this batch's scoped files.
