---
status: resolved
file: magefile_test.go
line: 109
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575b1b,comment:PRRC_kwDOR5y4QM65BvYS
---

# Issue 004: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify current string-based error assertions and whether typed sentinel assertions exist.

rg -n --type=go 'strings\.Contains\(\s*err\.Error\(\)' magefile_test.go
rg -n --type=go '\berrRaceEnabledGoCommand\b' magefile.go magefile_test.go
rg -n --type=go '\berrors\.(Is|As)\(' magefile_test.go
```

Repository: compozy/agh

Length of output: 203

---

🏁 Script executed:

```shell
#!/bin/bash
# Check the implementation of runRaceEnabledGoCommand in magefile.go
rg -n --type=go -A 5 'func runRaceEnabledGoCommand' magefile.go
```

Repository: compozy/agh

Length of output: 333

---

🏁 Script executed:

```shell
#!/bin/bash
# Get full test context around lines 104-109
sed -n '95,115p' magefile_test.go
```

Repository: compozy/agh

Length of output: 650

---



</details>

**Use `errors.Is()` and `errors.As()` instead of string matching for error assertions.**

Lines 104 and 107 use `strings.Contains(err.Error(), ...)` to validate error messages, which violates the coding guideline: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

Refactor to use typed error assertions:
1. Add a sentinel error in `magefile.go`: `var errRaceEnabledGoCommand = errors.New("run race-enabled go command")`
2. Update the error return in `runRaceEnabledGoCommand` to wrap the sentinel: `fmt.Errorf("%w %v: %w", errRaceEnabledGoCommand, args, err)`
3. Replace string assertions with `errors.Is(err, errRaceEnabledGoCommand)` to verify the sentinel is in the error chain
4. Use `errors.As(err, &exec.ExitError{})` to validate the underlying command failure

This approach is more robust, fails fast on logic changes, and aligns with Go error handling best practices.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@magefile_test.go` around lines 104 - 109, Add a sentinel error and switch the
test to use errors.Is/As: declare var errRaceEnabledGoCommand = errors.New("run
race-enabled go command") in magefile.go, change runRaceEnabledGoCommand to wrap
that sentinel when returning errors (e.g., use fmt.Errorf with %w to include
args and the underlying err), then in the test replace the strings.Contains
checks with errors.Is(err, errRaceEnabledGoCommand) to assert the sentinel is in
the chain and use errors.As(err, new(*exec.ExitError)) (or a typed variable) to
assert the underlying failure is an exec.ExitError.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `magefile_test.go` currently matches wrapped errors by string content, which is brittle and violates the project guidance to use `errors.Is`/`errors.As` for error matching.
  - The root cause is that `runRaceEnabledGoCommand` does not expose a stable typed sentinel in its error chain, so the minimal production fix requires a small change in `magefile.go` even though that file is outside the listed code-file scope.
  - Completed fix: added `errRaceEnabledGoCommand` in `magefile.go`, wrapped it from `runRaceEnabledGoCommand`, and updated the mage test to assert `errors.Is` plus `errors.As(*exec.ExitError)` instead of matching strings.
  - Scope note: `magefile.go` was touched because the typed error chain required by the review cannot be added from the test alone.
  - Verification: `make verify` passed after the change set.
