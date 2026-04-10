---
status: resolved
file: cmd/agh-codegen/main_test.go
line: 34
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QU5-,comment:PRRC_kwDOR5y4QM620Aps
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

rg -n -C3 'func checkJSONFile|stale; run codegen|Contains\(err\.Error\(\)|errors\.Is\(|errors\.As\('
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
#!/bin/bash
# Find and examine the checkJSONFile function and test
fd -e go cmd/agh-codegen | head -20
```

Repository: compozy/agh

Length of output: 420

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at main_test.go lines around 32-34
sed -n '25,45p' cmd/agh-codegen/main_test.go
```

Repository: compozy/agh

Length of output: 810

---

🏁 Script executed:

```shell
#!/bin/bash
# Find checkJSONFile function definition
rg -n 'func checkJSONFile' cmd/agh-codegen/
```

Repository: compozy/agh

Length of output: 135

---

🏁 Script executed:

```shell
#!/bin/bash
# Check for existing sentinel errors in the file
rg -n 'var.*Err|errors\.New' cmd/agh-codegen/ | head -30
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Examine the checkJSONFile function implementation
sed -n '124,145p' cmd/agh-codegen/main.go
```

Repository: compozy/agh

Length of output: 656

---

🏁 Script executed:

```shell
#!/bin/bash
# Look for all error patterns in main.go to understand current error handling
rg -n 'return.*Errorf|return.*Error|fmt\.Errorf|errors\.' cmd/agh-codegen/main.go
```

Repository: compozy/agh

Length of output: 1436

---



</details>

**Use `errors.Is()` instead of string matching for error assertions**

At line 33, `strings.Contains(err.Error(), "stale")` couples the test to message text and violates the coding guideline requiring `errors.Is()` or `errors.As()` for error matching.

Define a sentinel error in `cmd/agh-codegen/main.go`:

```go
var ErrStaleGeneratedFile = errors.New("generated file is stale")
```

Update the `checkJSONFile()` function to wrap this sentinel at line 142:

```go
if !reflect.DeepEqual(gotCanonical, wantCanonical) {
    return fmt.Errorf("%s: %w", path, ErrStaleGeneratedFile)
}
```

Then update the test assertion:

```go
if !errors.Is(err, ErrStaleGeneratedFile) {
    t.Fatalf("checkJSONFile() error = %v, want ErrStaleGeneratedFile", err)
}
```

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@cmd/agh-codegen/main_test.go` around lines 32 - 34, Tests are matching error
text instead of using errors.Is; define a sentinel var ErrStaleGeneratedFile =
errors.New("generated file is stale") (in the same package as checkJSONFile),
update checkJSONFile to wrap that sentinel when the canonical JSON differs
(e.g., return fmt.Errorf("%s: %w", path, ErrStaleGeneratedFile) inside the
mismatch branch), and change the test assertion to use errors.Is(err,
ErrStaleGeneratedFile) instead of strings.Contains(err.Error(), "stale") to
assert the specific error.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is accurate. `TestCheckJSONFileRejectsContentDifferences` currently matches a stale-file condition via `strings.Contains(err.Error(), "stale")`, which is brittle and contrary to the workspace rule to use `errors.Is()`/`errors.As()` for error matching.
  - Root cause: `checkJSONFile()` exposes the stale-generated-file condition only as formatted text, so the test has no typed error to assert on.
  - Fix approach: introduce a sentinel stale-file error in `cmd/agh-codegen/main.go`, wrap it from `checkJSONFile()`, and update the test to assert with `errors.Is`.
  - Resolution: implemented in `cmd/agh-codegen/main.go` and `cmd/agh-codegen/main_test.go`, then verified with focused package tests plus `make verify`.
