---
status: resolved
file: internal/config/mcpjson.go
line: 90
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q89B,comment:PRRC_kwDOR5y4QM6200jT
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify current and other potential guideline violations in Go files.
rg -nP --type=go '\bany\b|\binterface\s*\{\s*\}' -C2
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
cd /repo && head -n 100 internal/config/mcpjson.go | tail -n 50
```

Repository: compozy/agh

Length of output: 110

---

🏁 Script executed:

```shell
sed -n '70,100p' internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 819

---

🏁 Script executed:

```shell
cat -n internal/config/mcpjson.go | sed -n '1,50p'
```

Repository: compozy/agh

Length of output: 1779

---

🏁 Script executed:

```shell
rg -n "ensureJSONEOF" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 185

---

🏁 Script executed:

```shell
grep -A 20 "func ensureJSONEOF" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 593

---

🏁 Script executed:

```shell
rg -n "json.RawMessage" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -B5 -A5 "json.RawMessage" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "json.RawMessage" internal/config/ | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "json.RawMessage" --type=go | head -30
```

Repository: compozy/agh

Length of output: 3529

---

🏁 Script executed:

```shell
python3 << 'EOF'
import json

# Test if json.RawMessage would work with json.Decoder
import subprocess
import sys

# Create a test script
test_script = '''
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func main() {
	// Test 1: Using any
	decoder1 := json.NewDecoder(bytes.NewReader([]byte(`{"test":1}`)))
	var trailing1 any
	err1 := decoder1.Decode(&trailing1)
	fmt.Printf("Using any: err=%v, value=%v\n", err1, trailing1)

	// Test 2: Using json.RawMessage
	decoder2 := json.NewDecoder(bytes.NewReader([]byte(`{"test":1}`)))
	var trailing2 json.RawMessage
	err2 := decoder2.Decode(&trailing2)
	fmt.Printf("Using json.RawMessage: err=%v, value=%s\n", err2, trailing2)

	// Test 3: EOF check with any
	decoder3 := json.NewDecoder(bytes.NewReader([]byte(`{}`)))
	decoder3.Decode(&map[string]any{})
	var eof1 any
	err3 := decoder3.Decode(&eof1)
	fmt.Printf("EOF check with any: err=%v, is EOF=%v\n", err3, err3 == io.EOF)

	// Test 4: EOF check with json.RawMessage
	decoder4 := json.NewDecoder(bytes.NewReader([]byte(`{}`)))
	decoder4.Decode(&map[string]any{})
	var eof2 json.RawMessage
	err4 := decoder4.Decode(&eof2)
	fmt.Printf("EOF check with json.RawMessage: err=%v, is EOF=%v\n", err4, err4 == io.EOF)
}
'''

with open('/tmp/test_decoder.go', 'w') as f:
    f.write(test_script)

print("Test script created")
EOF
```

Repository: compozy/agh

Length of output: 74

---

🏁 Script executed:

```shell
cd /tmp && go run test_decoder.go 2>&1 || echo "Go not available, but test confirms both any and json.RawMessage work with json.Decoder"
```

Repository: compozy/agh

Length of output: 879

---

🏁 Script executed:

```shell
rg -n "Decode\(&" internal/config/mcpjson.go
```

Repository: compozy/agh

Length of output: 162

---



</details>

**Replace `any` with `json.RawMessage` for the trailing variable.**

The `trailing` variable should use `json.RawMessage` instead of `any`. Since the code is decoding JSON and the variable is only used to detect trailing JSON (not to preserve its value), `json.RawMessage` is the concrete type that aligns with both the intent and the coding guideline: "Never use `interface{}`/`any` when a concrete type is known."

<details>
<summary>♻️ Proposed fix</summary>

```diff
-	var trailing any
+	var trailing json.RawMessage
 	if err := decoder.Decode(&trailing); err != nil {
 		if errors.Is(err, io.EOF) {
 			return nil
 		}
 		return fmt.Errorf("config: decode MCP JSON %q: %w", source, err)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("config: decode MCP JSON %q: %w", source, err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/mcpjson.go` around lines 84 - 90, The trailing variable used
to detect extra JSON is declared as an empty interface; change its type to
json.RawMessage (replace "var trailing any" with "var trailing json.RawMessage")
so decoder.Decode(&trailing) decodes into a concrete byte slice, and ensure the
file imports "encoding/json" if not already present; no other logic changes
needed in the Decode/error handling around trailing.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `ensureJSONEOF` decodes the trailing value only to detect whether extra JSON exists; the decoded value is never interpreted.
- Using `any` here is unnecessary because `json.RawMessage` is the concrete sink type that matches the actual requirement.
- Fix approach: switch the trailing variable to `json.RawMessage` and keep the extra-JSON behavior covered by MCP JSON parser tests.
