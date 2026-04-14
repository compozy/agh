---
status: resolved
file: internal/api/udsapi/server_test.go
line: 158
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565HzD,comment:PRRC_kwDOR5y4QM63qGaY
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# 1) Check if udsapi defines sentinel/typed errors for constructor validation.
rg -n --type=go -C3 'var\s+Err\w+\s*=\s*errors\.New|type\s+\w+Error\s+struct' internal/api/udsapi

# 2) Inspect New(...) validation returns to see if errors are wrapped for errors.Is().
rg -n --type=go -C3 'session manager is required|task service is required|observer is required|workspace resolver is required|fmt\.Errorf\(' internal/api/udsapi/server.go

# 3) Find current error assertion style in this test file.
rg -n --type=go -C2 'err\.Error\(\)|strings\.Contains|errors\.Is|errors\.As' internal/api/udsapi/server_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# First, verify the file exists and get basic info
find . -name "server_test.go" -path "*/udsapi/*" 2>/dev/null | head -20
```

Repository: compozy/agh

Length of output: 91

---

🏁 Script executed:

```shell
# If the above finds the file, examine lines around 157-158
if [ -f "internal/api/udsapi/server_test.go" ]; then
  echo "=== Lines 150-170 of server_test.go ===" 
  sed -n '150,170p' internal/api/udsapi/server_test.go
fi
```

Repository: compozy/agh

Length of output: 611

---

🏁 Script executed:

```shell
# Check the New() function implementation in server.go
if [ -f "internal/api/udsapi/server.go" ]; then
  echo "=== Searching for New() constructor ===" 
  rg -n "^func New" internal/api/udsapi/server.go -A 20
fi
```

Repository: compozy/agh

Length of output: 685

---

🏁 Script executed:

```shell
# Search for error definitions in the udsapi package
if [ -d "internal/api/udsapi" ]; then
  echo "=== Looking for error definitions ===" 
  rg -n "var Err|type.*Error" internal/api/udsapi/ -t go
fi
```

Repository: compozy/agh

Length of output: 92

---

🏁 Script executed:

```shell
# Get the full New() function
sed -n '253,310p' internal/api/udsapi/server.go
```

Repository: compozy/agh

Length of output: 1615

---

🏁 Script executed:

```shell
# Get the test cases to see what wantErr values are
sed -n '100,160p' internal/api/udsapi/server_test.go
```

Repository: compozy/agh

Length of output: 1679

---

🏁 Script executed:

```shell
# Check the entire udsapi directory for any error type definitions
ls -la internal/api/udsapi/
```

Repository: compozy/agh

Length of output: 1440

---

🏁 Script executed:

```shell
# Search more broadly for error types in udsapi
find internal/api/udsapi -name "*.go" -type f -exec basename {} \;
```

Repository: compozy/agh

Length of output: 375

---

🏁 Script executed:

```shell
# Check if errors are wrapped with fmt.Errorf anywhere in the file
rg -n "fmt\.Errorf.*session manager is required|fmt\.Errorf.*task service is required" internal/api/udsapi/server.go
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Use sentinel errors instead of string matching for New() validation errors.**

The test currently matches error text with `strings.Contains(err.Error(), ...)`, which violates the coding guideline: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

The New() function in server.go returns plain errors.New() for validation failures. To fix this:

1. Define sentinel errors in server.go:
   ```go
   var (
       ErrSessionManagerRequired = errors.New("udsapi: session manager is required")
       ErrTaskServiceRequired    = errors.New("udsapi: task service is required")
       ErrObserverRequired       = errors.New("udsapi: observer is required")
       ErrWorkspaceResolverRequired = errors.New("udsapi: workspace resolver is required")
   )
   ```

2. Return these sentinel errors from New() instead of creating new errors each time.

3. Update test assertions to use `errors.Is(err, ErrSessionManagerRequired)` instead of string matching.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/server_test.go` around lines 157 - 158, Replace
string-based error matching with sentinel errors: define exported package-level
sentinel errors (ErrSessionManagerRequired, ErrTaskServiceRequired,
ErrObserverRequired, ErrWorkspaceResolverRequired) in server.go and have
New(...) return these sentinel errors for the respective validation failures
instead of constructing new errors each call; then update the test in
server_test.go to assert using errors.Is(err, ErrSessionManagerRequired) (and
the other sentinels) rather than strings.Contains(err.Error(), ...), locating
the logic around the New() call and the validation branches that currently
return errors.New.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The test matches constructor failures with `strings.Contains`, which conflicts with the repo rule to use `errors.Is`/`errors.As` instead of error-string matching. Fixing this requires a minimal adjacent production change in `internal/api/udsapi/server.go`: introduce constructor sentinel errors and assert with `errors.Is` in the scoped test file.
  Resolution: Added exported constructor sentinel errors in `internal/api/udsapi/server.go` and updated the scoped tests to assert them with `errors.Is`.
