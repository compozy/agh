---
status: resolved
file: internal/api/udsapi/extensions_additional_test.go
line: 145
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZa,comment:PRRC_kwDOR5y4QM623eZr
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify test naming/style in this changed file.
fd -i 'extensions_additional_test.go' | while read -r f; do
  echo "== $f =="
  echo "-- top-level tests --"
  rg -n '^\s*func\s+Test' "$f"
  echo "-- subtests using Should... pattern --"
  rg -n 't\.Run\("Should' "$f" || true
done
```

Repository: compozy/agh

Length of output: 442

---



</details>

**Convert all tests to table-driven subtests with `t.Run("Should...")` naming.**

Lines 14-117 use individual test functions instead of the required table-driven pattern with subtests. Line 119-145 uses table-driven tests but with descriptive names (`"nil"`, `"not found"`, etc.) instead of `Should...` pattern. Per coding guidelines, all Go tests must use `t.Run("Should...")` subtests as default.

Additionally, negative-path assertions (lines 73-81, 113-116, 149-152) only check HTTP status codes; add specific error message validation using the assertion helpers to strengthen regression detection.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/extensions_additional_test.go` around lines 14 - 145,
These tests use standalone Test... functions and non-"Should..." names rather
than table-driven t.Run subtests and also lack error-message assertions on
negative paths; refactor TestListExtensionsHandlerReturnsExtensions,
TestInstallExtensionHandlerValidatesAndCreates,
TestEnableDisableExtensionHandlers and the existing table in
TestExtensionStatusCodeMappingsAndApproveSession into table-driven subtests
using t.Run("Should ...") entries (e.g., for success and each negative case) and
move per-case setup into each subtest, and for the negative-path checks (the
missing path/checksum and blank name requests and the error cases checked via
extensionStatusCode) add assertions that inspect the response body/error message
using the existing decodeJSONResponse or your test assertion helpers in addition
to verifying HTTP status codes so failures assert on both status and message.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `extensions_additional_test.go` mixes standalone tests, non-`Should...` subtest names, and weak negative-path assertions, which conflicts with the repo’s Go test conventions.
- Fix approach: refactor the file into table-driven `t.Run("Should...")` cases and fold the stronger error-message assertions from issue 003 into the same pass.
