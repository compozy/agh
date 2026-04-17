---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 538
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57y10e,comment:PRRC_kwDOR5y4QM644c8d
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
fd -type f -name "tool_mcp_resources_test.go"
```

Repository: compozy/agh

Length of output: 225

---

🏁 Script executed:

```shell
fd -type file -name "*tool_mcp_resources_test.go"
```

Repository: compozy/agh

Length of output: 225

---

🏁 Script executed:

```shell
find . -name "*tool_mcp_resources_test.go" -type f
```

Repository: compozy/agh

Length of output: 99

---

🏁 Script executed:

```shell
sed -n '523,538p' ./internal/daemon/tool_mcp_resources_test.go
```

Repository: compozy/agh

Length of output: 753

---

🏁 Script executed:

```shell
sed -n '572,582p' ./internal/daemon/tool_mcp_resources_test.go
```

Repository: compozy/agh

Length of output: 455

---

🏁 Script executed:

```shell
cat -n ./internal/daemon/tool_mcp_resources_test.go | sed -n '510,545p'
```

Repository: compozy/agh

Length of output: 1831

---

🏁 Script executed:

```shell
cat -n ./internal/daemon/tool_mcp_resources_test.go | sed -n '560,590p'
```

Repository: compozy/agh

Length of output: 1231

---

🏁 Script executed:

```shell
grep -n "func validateAndEncodeTool" ./internal/daemon/tool_mcp_resources_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
find . -name "*.go" -type f -exec grep -l "validateAndEncodeTool" {} \;
```

Repository: compozy/agh

Length of output: 139

---

🏁 Script executed:

```shell
grep -n "validateAndEncodeTool\|json.Marshal" ./internal/daemon/tool_mcp_resources.go | head -30
```

Repository: compozy/agh

Length of output: 178

---

🏁 Script executed:

```shell
sed -n '649,750p' ./internal/daemon/tool_mcp_resources.go
```

Repository: compozy/agh

Length of output: 2235

---

🏁 Script executed:

```shell
find . -name "*.go" -path "*/toolspkg/*" -o -name "tool.go" | head -10
```

Repository: compozy/agh

Length of output: 79

---

🏁 Script executed:

```shell
cat -n ./internal/tools/tool.go | head -80
```

Repository: compozy/agh

Length of output: 2701

---

🏁 Script executed:

```shell
find . -name "*.go" -path "*aghconfig*" | grep -v test
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg "type MCPServer" -A 10
```

Repository: compozy/agh

Length of output: 3089

---

🏁 Script executed:

```shell
grep -n "MarshalJSON\|MarshalText" ./internal/config/provider.go
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Replace `map[string]any` with typed structs for known payload shapes.**

Using `map[string]any` here violates the coding guideline: "Never use `interface{}`/`any` when a concrete type is known." Decode directly into typed structs matching the payload shape.

For the tool payload, decode into a struct with fields: Name, Description, Source, ReadOnly (JSON tags: "name", "description", "source", "read_only").

For the MCP payload, decode into a struct with fields: Name, Command (verify the actual JSON tag capitalization, as the current test checks `mcpPayload["Name"]` and `mcpPayload["Command"]` with capitals—this may require lowercase JSON tags per Go convention).

Applies to: lines 523-538 and 572-582

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 523 - 538, Replace
the use of map[string]any in the test by decoding toolEncoded into a concrete
struct instead of toolPayload: define a local struct type with fields Name,
Description, Source, ReadOnly and json tags "name", "description", "source",
"read_only", then call json.Unmarshal(toolEncoded, &thatStruct) and assert
against its fields (replace checks against toolPayload["..."] with struct field
comparisons). Do the same for the MCP payload: replace mcpPayload map usage by
decoding into a typed struct with fields Name and Command and ensure the
struct's json tags use the correct lowercase field names (e.g., "name",
"command") so the test reads struct.Name and struct.Command rather than
mcpPayload["Name"]/["Command"].
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: this test decodes two known canonical JSON payload shapes into `map[string]any`, even though the expected structures are fixed and typed. That loses compile-time checking and conflicts with the repo rule against `any` where a concrete shape is known.
- Fix plan: replace the map decodes with small typed test structs for the tool and MCP payloads.
- Resolution: implemented. `TestValidateAndEncodeToolAndMCPServer` now decodes tool and MCP payloads into concrete structs instead of `map[string]any`.
- Verification: `go test ./internal/daemon`, `make verify`.
