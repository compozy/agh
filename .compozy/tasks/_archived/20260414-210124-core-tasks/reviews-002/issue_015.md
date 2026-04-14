---
status: resolved
file: internal/extension/manager_test.go
line: 762
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lgw,comment:PRRC_kwDOR5y4QM63o2Qx
---

# Issue 015: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify string-based error matching in the changed reload tests.
rg -nP 'strings\.Contains\(\s*err\.Error\(\)\s*,\s*"[^"]+"\s*\)' internal/extension/manager_test.go
```

Repository: compozy/agh

Length of output: 613

---

🏁 Script executed:

```shell
#!/bin/bash
# Step 1: Find the Reload() method definition and its error returns
rg -A 10 'func.*Reload\(' internal/extension/ | head -30
```

Repository: compozy/agh

Length of output: 674

---

🏁 Script executed:

```shell
#!/bin/bash
# Step 2: Check if ErrorContains helper exists in testutil
rg -r 'ErrorContains|ErrorAs' internal/testutil/ internal/api/testutil/ 2>/dev/null | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Step 3: Check error definitions in manager.go - look for custom error types or sentinel errors
rg -B 2 -A 2 'errors\.(New|Is|As)|fmt\.Errorf|var.*Error' internal/extension/manager.go | head -40
```

Repository: compozy/agh

Length of output: 1058

---

🏁 Script executed:

```shell
#!/bin/bash
# Step 4: Get context around the specific error messages to see what functions return them
rg -B 5 '"manager is required"' internal/extension/
rg -B 5 '"registry is required"' internal/extension/
```

Repository: compozy/agh

Length of output: 1472

---



</details>

**Refactor errors to use sentinel values instead of string matching.**

Lines 760 and 781 use `strings.Contains(err.Error(), "...")` to match errors, which violates the coding guidelines. Errors returned by `Reload()`, `Start()`, `Stop()`, and related methods are currently created with `errors.New("extension: ...")`, making them unsuitable for `errors.Is()` matching.

Define these errors as package-level sentinels in `manager.go`:
```go
var (
    ErrContextRequired   = errors.New("extension: context is required")
    ErrManagerRequired   = errors.New("extension: manager is required")
    ErrRegistryRequired  = errors.New("extension: registry is required")
    // ... others as needed
)
```

Then update the test assertions to use `errors.Is()`:
```go
if err := nilManager.Reload(testutil.Context(t)); !errors.Is(err, ErrManagerRequired) {
    t.Fatalf("nil manager Reload() error = %v, want %v", err, ErrManagerRequired)
}
```

Apply the same refactor to other string-based error checks in the file (lines 271, 1015, 1019).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/manager_test.go` around lines 760 - 762, Tests currently
assert errors by matching substrings of err.Error() (e.g., in manager_test.go
around Reload/Start/Stop checks); define package-level sentinel errors in
manager.go (e.g., ErrContextRequired, ErrManagerRequired, ErrRegistryRequired,
etc.) and replace the string-based errors.New(...) returns in the manager
methods with those sentinel variables, then update the tests (checks around
Reload, Start, Stop and other mentioned lines) to use errors.Is(err,
ErrManagerRequired) (or the appropriate sentinel) instead of strings.Contains,
ensuring all relevant error returns and test assertions reference the new
sentinel symbols like ErrManagerRequired, ErrContextRequired,
ErrRegistryRequired and the methods Reload/Start/Stop.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The touched tests in `internal/extension/manager_test.go` currently verify manager failures by matching substrings in `err.Error()`, and those call sites are backed by ad hoc `errors.New(...)` values in `internal/extension/manager.go`.
  Root cause: the manager exposes several repeated error conditions without stable sentinel values, forcing brittle string matching in tests and callers.
  Planned fix: introduce minimal package-level sentinel errors in `internal/extension/manager.go`, return or wrap them from the relevant manager methods, and update the touched tests to use `errors.Is(...)`. This requires a minimal supporting production change outside the scoped file.

## Resolution

- Added stable sentinel errors in `internal/extension/manager.go` for the repeated manager failure conditions and updated the touched tests in `internal/extension/manager_test.go` to assert them with `errors.Is(...)`.
- This required a minimal supporting production change outside the scoped test file, as noted in triage.
