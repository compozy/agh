---
status: resolved
file: internal/memory/store.go
line: 88
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745a1,comment:PRRC_kwDOR5y4QM65BAQD
---

# Issue 007: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail
rg -n -C2 'type Backend interface|func \([^)]*\*Store\) (LoadPromptIndex|LoadIndex)\b|var _ Backend = \(\*Store\)\(nil\)' internal/memory
```

Repository: compozy/agh

Length of output: 763

---

🏁 Script executed:

```shell
cat -n internal/memory/types.go | sed -n '86,120p'
```

Repository: compozy/agh

Length of output: 1629

---

🏁 Script executed:

```shell
rg -n 'func \(s \*Store\)' internal/memory/store.go | head -30
```

Repository: compozy/agh

Length of output: 2287

---

🏁 Script executed:

```shell
rg -n 'LoadPromptIndex' internal/memory/
```

Repository: compozy/agh

Length of output: 156

---

🏁 Script executed:

```shell
rg -n 'var _ Backend' internal/memory/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n 'Store.*Backend|Backend.*Store' internal/memory/ | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n 'backend|Backend' internal/memory/store.go | head -20
```

Repository: compozy/agh

Length of output: 104

---

🏁 Script executed:

```shell
rg -n ':= NewStore|= memory.NewStore' --type go | head -10
```

Repository: compozy/agh

Length of output: 878

---

🏁 Script executed:

```shell
rg -n 'func New.*Backend|New.*Store' internal/memory/ | head -20
```

Repository: compozy/agh

Length of output: 1624

---

🏁 Script executed:

```shell
rg -n 'var.*Backend|Backend\s*=' internal/ | grep -v test | head -20
```

Repository: compozy/agh

Length of output: 1637

---

🏁 Script executed:

```shell
rg -n 'func.*Backend\b' internal/ | grep -v test | head -20
```

Repository: compozy/agh

Length of output: 837

---

🏁 Script executed:

```shell
fd -e go -x grep -l 'Backend' {} \; | grep -v test</blockquote>
```

Repository: compozy/agh

Length of output: 211

---

🏁 Script executed:

```shell
rg -n 'memory.Backend|Backend\s*interface' internal/ | grep -v test
```

Repository: compozy/agh

Length of output: 107

---

🏁 Script executed:

```shell
rg -n 'type.*Backend' internal/memory/types.go -A 10
```

Repository: compozy/agh

Length of output: 613

---

🏁 Script executed:

```shell
# Check if Store is ever assigned to Backend or passed where Backend is expected
rg -n 'func.*Backend\s*\)|Backend\s*\)' internal/memory/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there's a wrapper or adapter that implements Backend using Store
rg -n 'type.*struct' internal/memory/ -A 3 | grep -B 3 -A 3 'Store'
```

Repository: compozy/agh

Length of output: 1453

---

🏁 Script executed:

```shell
# Check all files in internal/memory for Backend implementation
ls -la internal/memory/
```

Repository: compozy/agh

Length of output: 1183

---

🏁 Script executed:

```shell
# Verify if Store methods are actually being type-checked against Backend anywhere
rg -n 'LoadIndex|LoadPromptIndex' internal/memory/ -B 2 -A 2
```

Repository: compozy/agh

Length of output: 7071

---



</details>

**Add compile-time interface verification for `Store` to `Backend`.**

The `Backend` interface requires `LoadPromptIndex(scope Scope) (content string, truncated bool, err error)` (types.go:95), but `Store` implements `LoadIndex` instead (store.go:290). Although the signatures are identical, the method name mismatch means `*Store` does not satisfy `Backend`. Add a compile-time assertion and either rename the method or add a shim:

<details>
<summary>Suggested fix</summary>

```diff
+var _ Backend = (*Store)(nil)
+
 // List is the backend-aligned alias for Scan.
 func (s *Store) List(scope Scope) ([]Header, error) {
 	return s.Scan(scope)
 }
+
+// LoadPromptIndex is the backend-aligned alias for LoadIndex.
+func (s *Store) LoadPromptIndex(scope Scope) (content string, truncated bool, err error) {
+	return s.LoadIndex(scope)
+}
```
</details>

This satisfies the coding guideline: `Use compile-time interface verification: var _ Interface = (*Type)(nil)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
var _ Backend = (*Store)(nil)

// List is the backend-aligned alias for Scan.
func (s *Store) List(scope Scope) ([]Header, error) {
	return s.Scan(scope)
}

// LoadPromptIndex is the backend-aligned alias for LoadIndex.
func (s *Store) LoadPromptIndex(scope Scope) (content string, truncated bool, err error) {
	return s.LoadIndex(scope)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 85 - 88, The Store type does not
currently satisfy the Backend interface because Backend declares
LoadPromptIndex(scope Scope) (content string, truncated bool, err error) while
Store implements LoadIndex with the same signature; add a compile-time assertion
var _ Backend = (*Store)(nil) and fix the mismatch by either renaming
Store.LoadIndex to LoadPromptIndex or adding a small shim method
LoadPromptIndex(scope Scope) (string, bool, error) that simply delegates to
LoadIndex; ensure the assertion is placed near the Store type and that the
shim/rename preserves the original implementation and signature.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Backend` advertises `LoadPromptIndex`, but `Store` only exposes `LoadIndex`, and there is no compile-time assertion tying `*Store` to the interface.
  - That leaves the backend contract able to drift silently. I will add a backend-aligned shim plus a compile-time assertion and validate the interface surface in tests.

## Resolution

- Added `var _ Backend = (*Store)(nil)` and a backend-aligned `LoadPromptIndex(...)` shim on `Store`.
- Added regression coverage that exercises `LoadPromptIndex` through the `Backend` interface.
