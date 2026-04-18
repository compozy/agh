---
status: resolved
file: internal/config/mcpjson.go
line: 57
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRr,comment:PRRC_kwDOR5y4QM65B60T
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's examine the current state of the file
cat -n internal/config/mcpjson.go | head -80
```

Repository: compozy/agh

Length of output: 3033

---

🏁 Script executed:

```shell
# Check if ensureJSONEOF function exists in the codebase
rg -n "ensureJSONEOF" --type go
```

Repository: compozy/agh

Length of output: 339

---

🏁 Script executed:

```shell
# Check the git history to see the previous implementation
git log --oneline -n 20 -- internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 75

---

🏁 Script executed:

```shell
# Look at the broader context around the ParseMCPServersJSON function
rg -A 50 -B 5 "func ParseMCPServersJSON" --type go
```

Repository: compozy/agh

Length of output: 3536

---

🏁 Script executed:

```shell
# Check imports in the file
head -30 internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 696

---

🏁 Script executed:

```shell
# View the ensureJSONEOF function
rg -A 10 "^func ensureJSONEOF" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 396

---

🏁 Script executed:

```shell
# Check if there are any tests that validate the strict field checking behavior
rg -l "mcpServers|mcp_servers" --type go | grep -i test
```

Repository: compozy/agh

Length of output: 906

---

🏁 Script executed:

```shell
# Look at test files to see how unknown fields are expected to be handled
rg -A 20 "unknown.*field|typo|misspell" --type go
```

Repository: compozy/agh

Length of output: 8015

---

🏁 Script executed:

```shell
# Let's examine the mcpjson_test.go file to see if there are tests for unknown field handling
cat -n internal/config/mcpjson_test.go | head -200
```

Repository: compozy/agh

Length of output: 7460

---

🏁 Script executed:

```shell
# Search for any tests related to validation or field handling in mcpjson_test.go
rg -A 30 "Test.*MCP|unknown|field" internal/config/mcpjson_test.go | head -100
```

Repository: compozy/agh

Length of output: 3112

---

🏁 Script executed:

```shell
# Check if there are any tests that expect unknown fields to be rejected in the nested server objects
rg "unknown.*field|envv|argz" internal/config/mcpjson_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Let's see the complete ensureJSONEOF function to understand what it does
rg -A 15 "^func ensureJSONEOF" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 492

---

🏁 Script executed:

```shell
# Check if the code has any validation of unknown fields in nested objects
rg -B 5 -A 10 "DisallowUnknownFields|strict" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Let's verify what the mcpJSONServer struct looks like and if there are any validation tags
rg -B 2 -A 8 "type mcpJSONServer" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 480

---

🏁 Script executed:

```shell
# Let's check if there's a test that verifies unknown fields in nested objects are handled
# by creating a test scenario with an unknown field
cat > /tmp/test_unknown_field.go << 'EOF'
package main

import (
	"encoding/json"
	"fmt"
)

type mcpJSONServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func main() {
	// Test if unknown fields are silently ignored with json.Unmarshal
	jsonData := []byte(`{"command":"test","typo":"value"}`)
	
	var server mcpJSONServer
	err := json.Unmarshal(jsonData, &server)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Success - typo field was silently ignored: %+v\n", server)
	}
}
EOF
cd /tmp && go run test_unknown_field.go
```

Repository: compozy/agh

Length of output: 130

---

🏁 Script executed:

```shell
# Check if there's a previous version of the code in git history
git log --all --oneline --format="%H %s" -- internal/config/mcpjson.go | head -10
```

Repository: compozy/agh

Length of output: 179

---

🏁 Script executed:

```shell
# Look at git diff to see what changed in this file recently
git show --stat HEAD:internal/config/mcpjson.go 2>/dev/null | head -20
```

Repository: compozy/agh

Length of output: 369

---

🏁 Script executed:

```shell
# Let's check the git diff to see what the previous implementation was
git show 455b329675b0a9fde855adbc53f2be23e3097437:internal/config/mcpjson.go 2>/dev/null | grep -A 30 "func ParseMCPServersJSON"
```

Repository: compozy/agh

Length of output: 1042

---

🏁 Script executed:

```shell
# Let's also check if there's a test that should be catching unknown fields in nested objects
rg -A 5 "TestParseMCPServersJSON" internal/config/mcpjson_test.go | grep -E "func|unknown|field|envv|argz|typo"
```

Repository: compozy/agh

Length of output: 325

---



</details>

**Apply strict validation when decoding nested `mcpServers` entries.**

The switch to `json.Unmarshal` for individual server objects means unknown fields inside each server are silently ignored. A typo like `envv` instead of `env` will parse successfully and silently drop the misconfigured field instead of failing fast—a regression from the previous strict decode path. Use `json.NewDecoder` with `DisallowUnknownFields()` for each supported collection.

<details>
<summary>Proposed fix</summary>

```diff
 	if raw := root["mcpServers"]; len(raw) > 0 {
-		if err := json.Unmarshal(raw, &document.MCPServersCamel); err != nil {
+		dec := json.NewDecoder(bytes.NewReader(raw))
+		dec.DisallowUnknownFields()
+		if err := dec.Decode(&document.MCPServersCamel); err != nil {
 			return nil, fmt.Errorf("config: decode MCP JSON %q mcpServers: %w", sourceName, err)
 		}
 	}
 	if raw := root["mcp_servers"]; len(raw) > 0 {
-		if err := json.Unmarshal(raw, &document.MCPServersSnake); err != nil {
+		dec := json.NewDecoder(bytes.NewReader(raw))
+		dec.DisallowUnknownFields()
+		if err := dec.Decode(&document.MCPServersSnake); err != nil {
 			return nil, fmt.Errorf("config: decode MCP JSON %q mcp_servers: %w", sourceName, err)
 		}
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/mcpjson.go` around lines 47 - 57, The current json.Unmarshal
calls for decoding into document.MCPServersCamel and document.MCPServersSnake
silently ignore unknown fields; replace each json.Unmarshal(raw,
&document.MCPServersCamel) and json.Unmarshal(raw, &document.MCPServersSnake)
with a json.NewDecoder(bytes.NewReader(raw)) that calls DisallowUnknownFields()
before Decode(...) so unknown fields cause an error; ensure you import bytes if
missing and preserve the existing error wrapping (e.g., fmt.Errorf("config:
decode MCP JSON %q mcpServers: %w", sourceName, err) and similarly for
mcp_servers) and apply the same decoder pattern for both MCPServersCamel and
MCPServersSnake in mcpJSONDocument decoding.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `ParseMCPServersJSON`: the top-level document uses a strict decoder, but each nested `mcpServers` collection is decoded with `json.Unmarshal`, so unknown per-server fields are silently ignored. I will switch the nested collection decoding to `json.Decoder` with `DisallowUnknownFields()` and add regression coverage for typoed nested keys.
