---
status: resolved
file: internal/session/provider_lifecycle_test.go
line: 66
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11w,comment:PRRC_kwDOR5y4QM663fCG
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# 1) Locate err.Error() string matching in this test.
rg -nP --type=go 'strings\.Contains\s*\(\s*err\.Error\(\)' internal/session/provider_lifecycle_test.go

# 2) Discover sentinel/typed errors available for stronger assertions.
rg -nP --type=go 'var\s+Err[A-Za-z0-9_]+\s*=' internal/session internal/config internal/workspace

# 3) Inspect wrapped-error usage in session manager paths to confirm Is/As viability.
rg -nP --type=go 'func \(m \*Manager\)\s+(Create|Status|Resume)\b|fmt\.Errorf\(".*%w"|errors\.(Is|As)\(' internal/session -C2
```

Repository: compozy/agh

Length of output: 41794

---

🏁 Script executed:

```shell
# Read the provider_lifecycle_test.go file to understand the context
head -n 220 internal/session/provider_lifecycle_test.go | tail -n 160
```

Repository: compozy/agh

Length of output: 5482

---

🏁 Script executed:

```shell
# Search for sentinel error definitions in session package
rg -n 'var\s+Err[A-Za-z0-9_]+\s*=' internal/session --type=go | head -30

# Look at what manager.Status, manager.Create, and manager.Resume return for provider errors
rg -A5 'func.*Provider.*error' internal/session --type=go | head -50
```

Repository: compozy/agh

Length of output: 420

---

🏁 Script executed:

```shell
# Search for all Err* definitions in session
rg -n '^var\s+Err' internal/session --type=go

# Look at environment.Provider method to understand what error is returned
rg -B5 -A10 'func.*Provider.*error' internal/session/environment.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for var Err in session package
fd '\.go$' internal/session -type f -exec grep -l 'var Err' {} \;

# Look at the error definitions file
cat internal/session/errors.go
```

Repository: compozy/agh

Length of output: 285

---

🏁 Script executed:

```shell
# Search for error variable declarations
rg 'var\s+Err\w+' internal/session -A1

# Look at the entire provider_lifecycle_test.go file to see all errors
cat -n internal/session/provider_lifecycle_test.go | head -80
```

Repository: compozy/agh

Length of output: 2857

---

🏁 Script executed:

```shell
# Search for Err definitions in the entire session package
find internal/session -name "*.go" -type f | xargs grep 'var\s\+Err'

# Look at where providers come from and what error is returned
rg -B3 -A3 'ResolveProvider|Provider.*error' internal/session/manager_start.go | head -40

# Check the environment package for provider-related errors
rg 'Provider.*error|ErrProvider' internal/session/environment.go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at manager_start.go where Create is implemented
sed -n '50,100p' internal/session/manager_start.go

# Look for where the provider error comes from
rg -B5 -A5 'environment.*Provider' internal/session/manager_start.go

# Check session package files to find all defined errors
ls -la internal/session/*.go | grep -v test | head -20
```

Repository: compozy/agh

Length of output: 3385

---

🏁 Script executed:

```shell
# Look at the environment.go file where Provider resolution happens
sed -n '75,95p' internal/session/environment.go

# Search for where provider resolution error occurs
rg -B3 -A10 'Provider.*not.*found|provider.*unavailable' internal/session --type=go

# Check internal/config for provider errors
rg -B2 -A2 'ErrProvider' internal/config --type=go
```

Repository: compozy/agh

Length of output: 2798

---

🏁 Script executed:

```shell
# Look at how Create/Status/Resume use ResolveProvider
rg -B3 -A5 'ResolveProvider' internal/session/manager_start.go

# Check the actual error wrapping in manager_start.go around line 250-280
sed -n '240,280p' internal/session/manager_start.go

# Verify how the provider error flows through the code
rg -B5 -A5 'environment.Provider|ResolveProvider' internal/session/environment.go | head -40
```

Repository: compozy/agh

Length of output: 2172

---

🏁 Script executed:

```shell
# Check the imports in provider_lifecycle_test.go
head -20 internal/session/provider_lifecycle_test.go

# Look at the config package structure
ls -la internal/config/*.go | head -10

# Check if ErrProviderUnavailable is exported and accessible
grep -n 'ErrProviderUnavailable\|var Err' internal/config/provider.go | head -20

# Verify error wrapping in config.ResolveProvider
sed -n '1,100p' internal/config/provider.go | tail -50
```

Repository: compozy/agh

Length of output: 2806

---

🏁 Script executed:

```shell
# Check how config package is imported elsewhere in session tests
grep -r 'import.*config' internal/session --include='*.go' | head -5

# Check the error chain when provider is unavailable
sed -n '200,230p' internal/config/provider.go

# Look at how other session tests handle config errors
grep -B3 -A3 'ErrProviderUnavailable\|aghconfig' internal/session --include='*.go'
```

Repository: compozy/agh

Length of output: 1025

---

🏁 Script executed:

```shell
# Check imports in provider_lifecycle_test.go more carefully
sed -n '1,15p' internal/session/provider_lifecycle_test.go

# See if config/aghconfig is imported
grep -n 'aghconfig\|config' internal/session/provider_lifecycle_test.go | head -10

# Look at the harness to see how config is accessed
grep -A20 'func newHarness' internal/session/provider_lifecycle_test.go | head -25
```

Repository: compozy/agh

Length of output: 321

---

🏁 Script executed:

```shell
# Find where newHarness is defined
rg -n 'func newHarness' internal/session/provider_lifecycle_test.go

# Look at the complete test harness definition
rg -B5 -A30 'type.*arness|func newHarness' internal/session/provider_lifecycle_test.go

# Check the test file that might have the harness definition
find internal/session -name '*_test.go' -exec grep -l 'type.*arness' {} \;
```

Repository: compozy/agh

Length of output: 88

---

🏁 Script executed:

```shell
# Look at the harness definition in manager_test.go
rg -A30 'type.*arness struct' internal/session/manager_test.go

# See how it's created
rg -A20 'func newHarness' internal/session/manager_test.go | head -30

# Check what cfg field contains
grep -n 'cfg\s*' internal/session/manager_test.go | head -10
```

Repository: compozy/agh

Length of output: 2081

---

🏁 Script executed:

```shell
# Check what error the Manager.Create method actually returns when provider is invalid
rg -B10 -A10 'func.*prepareCreate|func.*prepareSessionAgent' internal/session/manager_start.go | head -60

# Look at where provider error occurs - in ResolveSessionAgent
rg -B5 -A10 'ResolveSessionAgent' internal/config/agent.go | head -40

# Look at the test more closely to understand the error context
sed -n '49,75p' internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 1402

---

🏁 Script executed:

```shell
# Look at ResolveSessionAgent in config/agent.go
sed -n '200,280p' internal/config/agent.go

# Look at what error is wrapped when provider is invalid
rg -B5 -A5 'ResolveSessionAgent' internal/session/manager_start.go
```

Repository: compozy/agh

Length of output: 2731

---



</details>

**Use typed error assertions with `errors.Is()` instead of `err.Error()` string matching.**

These string checks are brittle and couple tests to message wording. Replace with `errors.Is(err, aghconfig.ErrProviderUnavailable)` first; optional detail assertions may be kept if strictly needed for context validation.

This applies to lines 64–66, 171–176, and 209–214.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/provider_lifecycle_test.go` around lines 64 - 66, The test
currently asserts error details by matching substrings on err.Error() after
calling Create(), which is brittle; replace these string comparisons with typed
error assertions using errors.Is(err, aghconfig.ErrProviderUnavailable) for the
Create() error checks (and the other similar assertions in the same test file),
and only keep optional substring checks for extra context if necessary. Update
imports to include "errors" and use errors.Is(err,
aghconfig.ErrProviderUnavailable) in place of string-matching (for the Create()
call and the other two occurrences that check for "missing-provider"), ensuring
the test fails on the correct typed error rather than message wording.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- The provider-failure assertions in `provider_lifecycle_test.go` currently rely on string matching for provider-unavailable cases even though the error chain preserves `aghconfig.ErrProviderUnavailable`.
- Strengthening those assertions with `errors.Is` keeps the tests aligned with the actual contract while retaining string checks only for the extra session/agent context that has no sentinel.
- Resolved by asserting `errors.Is(err, aghconfig.ErrProviderUnavailable)` in the create/resume provider-failure paths while preserving the existing context-string checks.
