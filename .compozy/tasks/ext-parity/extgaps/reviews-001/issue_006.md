---
status: resolved
file: internal/bridges/registry.go
line: 220
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56__Bl,comment:PRRC_kwDOR5y4QM63zbyB
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's check the file structure and read the relevant section
wc -l internal/bridges/registry.go
```

Repository: compozy/agh

Length of output: 87

---

🏁 Script executed:

```shell
# Read the context around lines 219-220
sed -n '210,230p' internal/bridges/registry.go
```

Repository: compozy/agh

Length of output: 857

---

🏁 Script executed:

```shell
# Find the function containing line 220 and understand the scope
sed -n '180,240p' internal/bridges/registry.go
```

Repository: compozy/agh

Length of output: 2212

---

🏁 Script executed:

```shell
# Search for ErrBridgeInstanceReadOnly definition
rg "ErrBridgeInstanceReadOnly" internal/bridges/registry.go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 182

---

🏁 Script executed:

```shell
# Verify if trimmedID is available in scope at line 220
rg "trimmedID" internal/bridges/registry.go -B 5 -A 5
```

Repository: compozy/agh

Length of output: 7581

---

🏁 Script executed:

```shell
# Check for other bare returns of ErrBridgeInstanceReadOnly without wrapping
rg "return.*ErrBridgeInstanceReadOnly" internal/bridges/registry.go
```

Repository: compozy/agh

Length of output: 94

---

🏁 Script executed:

```shell
# Check if there are other methods with similar unguarded package-source checks
rg "BridgeInstanceSourcePackage" internal/bridges/registry.go -A 1
```

Repository: compozy/agh

Length of output: 147

---

</details>

**Wrap the read-only sentinel with instance context.**

The bare return at line 220 violates the repository's error-wrapping requirement and loses the instance ID from logs and callers. All other error returns in `UpdateInstance` are properly wrapped with context using `fmt.Errorf()`. Wrap this sentinel consistently while preserving it for `errors.Is()` matching.

<details>
<summary>Proposed fix</summary>

```diff
	if instance.Source == BridgeInstanceSourcePackage {
-		return nil, ErrBridgeInstanceReadOnly
+		return nil, fmt.Errorf("bridges: update bridge instance %q: %w", trimmedID, ErrBridgeInstanceReadOnly)
	}
```

</details>

As per coding guidelines, `**/*.go`: Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if instance.Source == BridgeInstanceSourcePackage {
		return nil, fmt.Errorf("bridges: update bridge instance %q: %w", trimmedID, ErrBridgeInstanceReadOnly)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bridges/registry.go` around lines 219 - 220, In UpdateInstance, the
return of the sentinel ErrBridgeInstanceReadOnly is not wrapped with context;
change the bare return that checks if instance.Source ==
BridgeInstanceSourcePackage to wrap the sentinel using fmt.Errorf so callers and
logs include the instance ID (or other identifying info) while preserving
errors.Is() matching (e.g., fmt.Errorf("instance %s: %w", instance.ID,
ErrBridgeInstanceReadOnly)); update the return at the check for instance.Source
== BridgeInstanceSourcePackage accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `UpdateInstance` returns `ErrBridgeInstanceReadOnly` bare for package-managed instances, unlike the other error paths in the method. That loses the instance identifier at the daemon boundary even though `errors.Is` matching can be preserved with wrapping.
- Fix plan: wrap the sentinel with the trimmed instance ID via `fmt.Errorf(...: %w, ErrBridgeInstanceReadOnly)` so logs/callers retain context without breaking `errors.Is`.
- Resolution: wrapped the read-only sentinel with the bridge instance ID while preserving `errors.Is` semantics.
- Verification: passed `go test ./internal/bridges` and `make verify`.
