---
status: resolved
file: internal/mcp/auth/pkce.go
line: 33
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59lV1z,comment:PRRC_kwDOR5y4QM67Ri1K
---

# Issue 018: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Locate and examine the pkce.go file
find . -name "pkce.go" -path "*/mcp/auth/*" | head -20
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
# Read the content of the file with line numbers
if [ -f "internal/mcp/auth/pkce.go" ]; then
  cat -n "internal/mcp/auth/pkce.go"
fi
```

Repository: compozy/agh

Length of output: 2360

---



</details>

**Wrap error returns to add operation-specific context at lines 32 and 46.**

Both `newPKCEPair` and `newState` return errors from `randomURLToken` without wrapping them. Wrap these errors with `fmt.Errorf` to distinguish verifier and state generation failures for callers.

<details>
<summary>Proposed changes</summary>

```diff
 func newPKCEPair(random io.Reader) (PKCEPair, error) {
    if random == nil {
        random = rand.Reader
    }
    verifier, err := randomURLToken(random, pkceVerifierBytes)
    if err != nil {
-		return PKCEPair{}, err
+		return PKCEPair{}, fmt.Errorf("mcp auth: generate PKCE verifier: %w", err)
    }
    sum := sha256.Sum256([]byte(verifier))
    return PKCEPair{
        Verifier:  verifier,
        Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
        Method:    "S256",
    }, nil
 }

 func newState(random io.Reader) (string, error) {
    if random == nil {
        random = rand.Reader
    }
-	return randomURLToken(random, stateBytes)
+	state, err := randomURLToken(random, stateBytes)
+	if err != nil {
+		return "", fmt.Errorf("mcp auth: generate oauth state: %w", err)
+	}
+	return state, nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func newPKCEPair(random io.Reader) (PKCEPair, error) {
	if random == nil {
		random = rand.Reader
	}
	verifier, err := randomURLToken(random, pkceVerifierBytes)
	if err != nil {
		return PKCEPair{}, fmt.Errorf("mcp auth: generate PKCE verifier: %w", err)
	}
	sum := sha256.Sum256([]byte(verifier))
	return PKCEPair{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(sum[:]),
		Method:    "S256",
	}, nil
}

func newState(random io.Reader) (string, error) {
	if random == nil {
		random = rand.Reader
	}
	state, err := randomURLToken(random, stateBytes)
	if err != nil {
		return "", fmt.Errorf("mcp auth: generate oauth state: %w", err)
	}
	return state, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/mcp/auth/pkce.go` around lines 30 - 33, newPKCEPair and newState are
returning raw errors from randomURLToken which loses operation context; update
the error returns in newPKCEPair (where verifier is generated using
randomURLToken with pkceVerifierBytes) and in newState (where state is generated
with stateBytes) to wrap the returned error using fmt.Errorf with a descriptive
message like "generate PKCE verifier: %w" and "generate state: %w" respectively
so callers can distinguish verifier vs state generation failures (ensure fmt is
imported).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `newPKCEPair` and `newState` return raw `randomURLToken` errors. The shared random-token helper already wraps the low-level read failure, but callers cannot distinguish verifier generation from OAuth state generation. Wrap both call sites with operation-specific context.
