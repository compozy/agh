---
status: resolved
file: extensions/bridges/discord/provider.go
line: 2351
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTPp,comment:PRRC_kwDOR5y4QM64dqFo
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "Check where not-initialized RPC errors are produced:"
rg -nP --type=go 'NewRPCError\(\s*(rpcCodeNotInitialized|-32003)\b'

echo
echo "Check for string-based matching of initialization errors:"
rg -nP --type=go 'Not initialized'
```

Repository: compozy/agh

Length of output: 5929

---

🏁 Script executed:

```shell
# Read the isNotInitializedRPCError function from discord/provider.go
sed -n '2320,2360p' extensions/bridges/discord/provider.go
```

Repository: compozy/agh

Length of output: 969

---

🏁 Script executed:

```shell
# Check if there are tests for isNotInitializedRPCError in discord
rg -n 'isNotInitializedRPCError' extensions/bridges/discord/
```

Repository: compozy/agh

Length of output: 668

---

🏁 Script executed:

```shell
# Verify if the function signature and behavior are tested
rg -B5 -A10 'isNotInitializedRPCError' extensions/bridges/discord/provider_test.go | head -50
```

Repository: compozy/agh

Length of output: 718

---

🏁 Script executed:

```shell
# Find all test usages of isNotInitializedRPCError in discord tests
rg -B2 -A2 'isNotInitializedRPCError' extensions/bridges/discord/provider_test.go
```

Repository: compozy/agh

Length of output: 379

---

🏁 Script executed:

```shell
# Check if there are any references to rpcCodeErr type definition in discord tests
rg -B5 'type rpcCodeErr' extensions/bridges/discord/provider_test.go
```

Repository: compozy/agh

Length of output: 131

---

🏁 Script executed:

```shell
# Get the full rpcCodeErr type definition including its Code() method
rg -A5 'type rpcCodeErr struct' extensions/bridges/discord/provider_test.go
```

Repository: compozy/agh

Length of output: 206

---



</details>

**Remove string-based RPC error matching fallback in `isNotInitializedRPCError`.**

The function currently falls back to string comparison (`strings.EqualFold(rpcErr.Message, "Not initialized")`), which is brittle and violates the coding guideline: "never compare error strings." Production code consistently uses the error code (`rpcCodeNotInitialized`), and multiple bridges use different message variants ("Not initialized", "not ready", etc.), making the message fallback unreliable. Keep matching strictly code-based via the `Code()` interface check.

<details>
<summary>🔧 Proposed fix</summary>

```diff
 func isNotInitializedRPCError(err error) bool {
 	if err == nil {
 		return false
 	}
 	type rpcCodeError interface {
 		Code() int
 	}
 	var codeErr rpcCodeError
 	if errors.As(err, &codeErr) && codeErr.Code() == rpcCodeNotInitialized {
 		return true
 	}
 	var rpcErr *subprocess.RPCError
-	if !errors.As(err, &rpcErr) {
-		return false
-	}
-	return rpcErr.Code == rpcCodeNotInitialized ||
-		strings.EqualFold(strings.TrimSpace(rpcErr.Message), "Not initialized")
+	return errors.As(err, &rpcErr) && rpcErr.Code == rpcCodeNotInitialized
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if err == nil {
		return false
	}
	type rpcCodeError interface {
		Code() int
	}
	var codeErr rpcCodeError
	if errors.As(err, &codeErr) && codeErr.Code() == rpcCodeNotInitialized {
		return true
	}
	var rpcErr *subprocess.RPCError
	return errors.As(err, &rpcErr) && rpcErr.Code == rpcCodeNotInitialized
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider.go` around lines 2336 - 2351, The
function isNotInitializedRPCError currently falls back to brittle string
matching on subprocess.RPCError.Message; remove that string-based check and
return only based on the code-based checks: keep the errors.As check for the
Code() interface (rpcCodeError) and the check against rpcCodeNotInitialized, and
if errors.As into *subprocess.RPCError, only use rpcErr.Code ==
rpcCodeNotInitialized to decide true/false; delete the strings.EqualFold(...
"Not initialized") fallback and ensure the function returns false when no code
match is found (no other error-string comparisons).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `isNotInitializedRPCError` still falls back to matching `subprocess.RPCError.Message == "Not initialized"`, which is brittle and inconsistent with the code-based path already present in the same helper. The fix is to remove the string fallback and tighten the discord tests so only RPC code-based matches succeed.
