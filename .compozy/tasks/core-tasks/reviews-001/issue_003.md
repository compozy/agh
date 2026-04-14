---
status: resolved
file: internal/api/core/errors_test.go
line: 196
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562anW,comment:PRRC_kwDOR5y4QM63mgQZ
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's look at the full test function context around lines 195-196
cd /tmp && find . -name "errors_test.go" -path "*/api/core/*" 2>/dev/null | head -1 | xargs wc -l
```

Repository: compozy/agh

Length of output: 56

---

🏁 Script executed:

```shell
# Read the test file to understand the context
cat -n internal/api/core/errors_test.go | sed -n '180,210p'
```

Repository: compozy/agh

Length of output: 1136

---

🏁 Script executed:

```shell
# Get more context on the test function structure
cat -n internal/api/core/errors_test.go | sed -n '150,220p'
```

Repository: compozy/agh

Length of output: 2946

---

🏁 Script executed:

```shell
# Let's also look at the errors implementation to understand what's being tested
cat -n internal/api/core/errors.go | head -100
```

Repository: compozy/agh

Length of output: 4019

---

🏁 Script executed:

```shell
# Check the Respond or error handling function being tested
rg -n "func.*Respond" internal/api/core/errors.go | head -20
```

Repository: compozy/agh

Length of output: 141

---

🏁 Script executed:

```shell
# Verify Go httptest.ResponseRecorder behavior with invalid status codes
python3 << 'EOF'
# Check Go standard library documentation behavior
# httptest.ResponseRecorder.WriteHeader panics if code is not in [100, 599]
# http.StatusText returns empty string for codes outside standard ranges

import subprocess
import sys

# Verify http.StatusText behavior for edge cases
result = subprocess.run(['go', 'run', '-'], input=b'''
package main
import (
	"fmt"
	"net/http"
)
func main() {
	fmt.Printf("http.StatusText(0) = %q\\n", http.StatusText(0))
	fmt.Printf("http.StatusText(200) = %q\\n", http.StatusText(200))
	fmt.Printf("http.StatusText(599) = %q\\n", http.StatusText(599))
	fmt.Printf("http.StatusText(600) = %q\\n", http.StatusText(600))
}
''', capture_output=True, text=True, cwd='/tmp')

print("StatusText validation:")
print(result.stdout)
if result.stderr:
	print("Error:", result.stderr)

# Now check ResponseRecorder behavior
result2 = subprocess.run(['go', 'run', '-'], input=b'''
package main
import (
	"fmt"
	"net/http/httptest"
)
func main() {
	// Test what happens with status 0
	rec := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("WriteHeader(0) panics: %v\\n", r)
		} else {
			fmt.Printf("WriteHeader(0) succeeded\\n")
		}
	}()
	rec.WriteHeader(0)
}
''', capture_output=True, text=True, cwd='/tmp')

print("\nResponseRecorder.WriteHeader behavior:")
print(result2.stdout)
if result2.stderr:
	print("Error:", result2.stderr)
EOF
```

Repository: compozy/agh

Length of output: 862

---

🏁 Script executed:

```shell
# Verify Go httptest.ResponseRecorder behavior with invalid status codes
python3 << 'EOF'
import subprocess

# Check http.StatusText behavior for edge cases
code = '''
package main
import (
	"fmt"
	"net/http"
)
func main() {
	fmt.Printf("http.StatusText(0) = %q\\n", http.StatusText(0))
	fmt.Printf("http.StatusText(200) = %q\\n", http.StatusText(200))
	fmt.Printf("http.StatusText(599) = %q\\n", http.StatusText(599))
	fmt.Printf("http.StatusText(600) = %q\\n", http.StatusText(600))
}
'''

result = subprocess.run(['go', 'run', '-'], input=code, capture_output=True, text=True, cwd='/tmp')
print("StatusText validation:")
print(result.stdout)
if result.stderr:
	print("Stderr:", result.stderr)

# Now check ResponseRecorder behavior with invalid codes
code2 = '''
package main
import (
	"fmt"
	"net/http/httptest"
)
func main() {
	// Test what happens with status 0
	rec := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("WriteHeader(0) panics: %v\\n", r)
		} else {
			fmt.Printf("WriteHeader(0) succeeded\\n")
		}
	}()
	rec.WriteHeader(0)
}
'''

result2 = subprocess.run(['go', 'run', '-'], input=code2, capture_output=True, text=True, cwd='/tmp')
print("\nResponseRecorder.WriteHeader behavior with status 0:")
print(result2.stdout)
if result2.stderr:
	print("Stderr:", result2.stderr)

# Test with valid status 599
code3 = '''
package main
import (
	"fmt"
	"net/http/httptest"
)
func main() {
	rec := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("WriteHeader(599) panics: %v\\n", r)
		} else {
			fmt.Printf("WriteHeader(599) succeeded, Code=%d\\n", rec.Code)
		}
	}()
	rec.WriteHeader(599)
}
'''

result3 = subprocess.run(['go', 'run', '-'], input=code3, capture_output=True, text=True, cwd='/tmp')
print("\nResponseRecorder.WriteHeader behavior with status 599:")
print(result3.stdout)
if result3.stderr:
	print("Stderr:", result3.stderr)
EOF
```

Repository: compozy/agh

Length of output: 286

---

🏁 Script executed:

```shell
# Let's verify the Go standard library behavior directly by checking documentation and source
# Create a simple test file to verify the behavior

cat > /tmp/test_status.go << 'EOF'
package main
import (
	"fmt"
	"net/http"
	"net/http/httptest"
)
func main() {
	// Check StatusText values
	fmt.Printf("http.StatusText(0) = %q\n", http.StatusText(0))
	fmt.Printf("http.StatusText(200) = %q\n", http.StatusText(200))
	fmt.Printf("http.StatusText(599) = %q\n", http.StatusText(599))
	fmt.Printf("http.StatusText(600) = %q\n", http.StatusText(600))
	
	// Test ResponseRecorder with status 0
	rec := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("WriteHeader(0) panics: %v\n", r)
		}
	}()
	rec.WriteHeader(0)
}
EOF

cd /tmp && go run test_status.go 2>&1
```

Repository: compozy/agh

Length of output: 208

---

🏁 Script executed:

```shell
# Also check the Gin source behavior with c.JSON
# Look at how Gin's c.JSON handles status codes
rg -A 5 "func.*JSON.*gin.Context" --type go | head -50
```

Repository: compozy/agh

Length of output: 379

---

🏁 Script executed:

```shell
# Check the actual test behavior - run the test to see if it panics
cd / && go test -v internal/api/core -run TestRespondErrorFallbackBranches 2>&1 | head -100
```

Repository: compozy/agh

Length of output: 195

---

🏁 Script executed:

```shell
# Verify that Gin's c.JSON calls WriteHeader and would trigger the panic
# Check Gin's implementation to understand the flow

cat > /tmp/test_gin_json.go << 'EOF'
package main
import (
	"fmt"
	"net/http/httptest"
	"github.com/gin-gonic/gin"
)
func main() {
	// Create a test context with status 0
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("c.JSON with status 0 panics: %v\n", r)
		} else {
			fmt.Printf("c.JSON with status 0 succeeded\n")
		}
	}()
	
	// Try to call JSON with status 0
	ctx.JSON(0, map[string]string{"error": "test"})
}
EOF

cd / && go run /tmp/test_gin_json.go 2>&1
```

Repository: compozy/agh

Length of output: 232

---

🏁 Script executed:

```shell
# Also verify with status 599 that it works
cat > /tmp/test_gin_json_599.go << 'EOF'
package main
import (
	"fmt"
	"net/http/httptest"
	"github.com/gin-gonic/gin"
)
func main() {
	gin.SetMode(gin.TestMode)
	
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("c.JSON with status 599 panics: %v\n", r)
		} else {
			fmt.Printf("c.JSON with status 599 succeeded, Code=%d\n", recorder.Code)
		}
	}()
	
	// Try to call JSON with status 599
	ctx.JSON(599, map[string]string{"error": "test"})
}
EOF

cd / && go run /tmp/test_gin_json_599.go 2>&1
```

Repository: compozy/agh

Length of output: 236

---



</details>

**`status: 0` will panic instead of exercising the fallback.**

Gin's `c.JSON` calls `WriteHeader(code)` internally, and Go's `httptest.ResponseRecorder.WriteHeader` panics for codes outside the valid range (100–599). The test case at line 195 will panic during `RespondError(ctx, 0, ...)` before reaching any assertions. Use status `599` instead—it's valid, produces an empty `http.StatusText()` like 0 would, and correctly tests the `"unknown error"` fallback branch.

<details>
<summary>Proposed test fix</summary>

```diff
-		{name: "unknown error fallback", status: 0, err: nil, mask: false, wantErr: "unknown error", wantStatus: 200},
+		{name: "unknown error fallback", status: 599, err: nil, mask: false, wantErr: "unknown error", wantStatus: 599},
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		{name: "unknown error fallback", status: 599, err: nil, mask: false, wantErr: "unknown error", wantStatus: 599},
		{name: "masked internal fallback", status: 599, err: nil, mask: true, wantErr: "internal server error", wantStatus: 599},
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors_test.go` around lines 195 - 196, The test uses an
invalid status code 0 which causes httptest.ResponseRecorder.WriteHeader to
panic when RespondError(ctx, 0, ...) is called; update the test case (the entry
with name "unknown error fallback") to use a valid status code such as 599
instead of 0 so the test exercises the unknown-error fallback path in
RespondError without panicking and still yields an empty http.StatusText() for
assertion.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Root cause check: reproduced the reported scenario with `go test ./internal/api/core -run TestRespondErrorFallbackBranches -count=1`, and the test passes on the current code.
- Why invalid: `RespondError` already covers the fallback branches correctly here; with Gin plus `httptest.ResponseRecorder`, the `status=0` branch retains the recorder's default HTTP 200 instead of panicking, so the review comment does not describe an actual bug in this test.

## Resolution

- No code change was required because the reported behavior is not reproducible on the current code path.
- The batch still passed the final `make verify` run unchanged for this issue.
