---
status: resolved
file: extensions/bridges/discord/provider.go
line: 2063
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57LwmE,comment:PRRC_kwDOR5y4QM64DQzj
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's locate and examine the file
find . -name "provider.go" -path "*/discord/*" | head -5
```

Repository: compozy/agh

Length of output: 95

---

🏁 Script executed:

```shell
# Once we have the file, let's look at the specific function
sed -n '2050,2070p' extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 616

---

🏁 Script executed:

```shell
# Let's also check if there are tests for this function
rg -l "isNotInitializedRPCError" --type go
```

Repository: compozy/agh

Length of output: 663

---

🏁 Script executed:

```shell
# Check how this function is being used
rg "isNotInitializedRPCError" -A 2 -B 2 --type go
```

Repository: compozy/agh

Length of output: 9712

---

🏁 Script executed:

```shell
# Check if discord has a test file and what tests it contains for this function
cat extensions/bridges/discord/provider_test.go | grep -A 20 "TestIsNotInitializedRPCError\|isNotInitializedRPCError"
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Let's also check what rpcCodeNotInitialized is and understand the RPC error structure
rg "rpcCodeNotInitialized" --type go | head -20
```

Repository: compozy/agh

Length of output: 2436

---

🏁 Script executed:

```shell
# Let's look for the subprocess package to understand RPCError structure
rg -A 5 "type RPCError" --type go
```

Repository: compozy/agh

Length of output: 408

---

🏁 Script executed:

```shell
# Check if there are any other bridges that use string fallback like discord
rg -A 5 "func isNotInitializedRPCError" extensions/bridges --type go
```

Repository: compozy/agh

Length of output: 3222

---



</details>

**Remove the string fallback and rely solely on the RPC error code.**

The `strings.Contains()` fallback violates the coding guideline: "Use errors.Is() and errors.As() for error matching — never compare error strings". This fallback can cause false positives (any error containing "not initialized" matches regardless of type) and is unnecessary—all other bridge implementations (telegram, teams, slack, linear, gchat, github) use only `errors.As()` with the code field check.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider.go` around lines 2056 - 2063, The
isNotInitializedRPCError function should not perform a string-based fallback;
remove the strings.Contains() path and rely only on errors.As() to detect RPC
errors by their Code() value: in isNotInitializedRPCError, keep the interface{
Code() int } extraction via errors.As(err, &rpcErr) and return rpcErr.Code() ==
rpcCodeNotInitialized, and otherwise return false so only typed RPC errors with
the matching code are considered.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `isNotInitializedRPCError()` still falls back to `strings.Contains(err.Error(), "not initialized")` after the typed RPC-code check.
  - That violates the repo rule against string-based error matching and can misclassify unrelated errors that happen to contain the same text.
  - Planned fix: remove the string fallback and cover the typed-only behavior with a focused unit test.
  - Resolution: `isNotInitializedRPCError()` now relies solely on typed RPC-code extraction via `errors.As`, and the unit coverage now includes a typed positive case plus a plain-string negative case.
  - Verification: `go test ./extensions/bridges/discord -count=1` and `make verify` both passed after the fix.
