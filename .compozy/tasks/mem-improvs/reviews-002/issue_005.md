---
status: resolved
file: magefile.go
line: 88
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575RS4,comment:PRRC_kwDOR5y4QM65BgwN
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
rg -n --type=go '@latest|gotest\.tools/gotestsum@'
```

Repository: compozy/agh

Length of output: 313

---

🏁 Script executed:

```shell
#!/bin/bash
# View the code context around the mentioned lines in magefile.go
sed -n '82,92p' magefile.go
echo "---"
sed -n '92,102p' magefile.go
echo "---"
sed -n '530,545p' magefile.go
echo "---"
sed -n '72,78p' magefile.go
echo "---"
sed -n '476,484p' magefile.go
```

Repository: compozy/agh

Length of output: 1551

---

🏁 Script executed:

```shell
#!/bin/bash
# Check the complete function signature and usage of runCommandInDirWithEnv
rg -B 2 -A 5 'func runCommandInDirWithEnv'
```

Repository: compozy/agh

Length of output: 384

---



</details>

**Pin all `@latest` tool versions and wrap errors with context in race-enabled command.**

1. `@latest` makes CI/test behavior non-reproducible and breaks builds unexpectedly. Pin:
   - `gotest.tools/gotestsum@latest` at lines 87, 97, 480
   - `golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest` at line 75

2. `runRaceEnabledGoCommand` (lines 532–534) violates Go guidelines by returning raw errors without wrapped context. It must call `fmt.Errorf("context: %w", err)` instead of returning the bare error.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@magefile.go` around lines 86 - 88, Pin the tool module versions and add
wrapped error context: replace all occurrences of
"gotest.tools/gotestsum@latest" (used near the run invocation and the other two
occurrences) with a specific version (e.g. gotest.tools/gotestsum@<version>) and
replace
"golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest" with
a fixed version as well; then in the function runRaceEnabledGoCommand change any
bare error returns to wrapped errors using fmt.Errorf with %w (e.g., return
fmt.Errorf("running race-enabled go command %v: %w", args, err)) so callers
receive contextualized errors.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `magefile.go` still shells out to `@latest` tool paths for `modernize`/`gotestsum`, which makes the build non-reproducible, and `runRaceEnabledGoCommand` returns the raw subprocess error without caller context.
- Impact: CI and local verification can change underneath the branch, and race-command failures lose the call-site context needed for diagnosis.
- Fix plan: pin the tool invocations to explicit module versions and wrap `runRaceEnabledGoCommand` failures with command context. The error-wrapping portion overlaps with issue 006 and will be satisfied by the same code change.
- Resolution: pinned `gotest.tools/gotestsum` to `v1.13.0`, pinned the gopls modernize tool to `v0.21.1`, and wrapped `runRaceEnabledGoCommand` failures with the invoked args for diagnosis.
- Verification: `go test -tags mage .`; `make verify`
