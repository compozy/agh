---
status: resolved
file: internal/memory/types.go
line: 96
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5745a_,comment:PRRC_kwDOR5y4QM65BAQR
---

# Issue 010: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Inspect the exported backend contract and the current memory-store method signatures.
rg -n -C2 --type=go 'type Backend interface' internal/memory/types.go
echo
rg -n -C2 --type=go 'func \(.*\)\s+(List|Read|Write|Delete|LoadPromptIndex)\(' internal/memory
```

Repository: compozy/agh

Length of output: 2045

---

🏁 Script executed:

```shell
# Find all implementations of the Backend interface
rg -n 'type.*Backend' internal/memory --type=go -A1 | head -30
```

Repository: compozy/agh

Length of output: 172

---

🏁 Script executed:

```shell
# Find all method implementations on *Store that implement Backend
rg -n 'func \(.*\*Store\)' internal/memory/store.go --type=go | grep -E '(List|Read|Write|Delete|LoadPromptIndex|Search|Reindex)'
```

Repository: compozy/agh

Length of output: 742

---

🏁 Script executed:

```shell
# Find all usages of Backend interface and its methods
rg -n '\.List\(|\.Read\(|\.Write\(|\.Delete\(|\.LoadPromptIndex\(' internal/memory --type=go | head -40
```

Repository: compozy/agh

Length of output: 3910

---

🏁 Script executed:

```shell
# Find where Backend is stored/passed as interface{}
rg -n 'Backend' internal/memory/types.go --type=go | head -20
```

Repository: compozy/agh

Length of output: 169

---

🏁 Script executed:

```shell
# Check the full Backend interface definition including LoadPromptIndex
cat -n internal/memory/types.go | sed -n '88,96p'
```

Repository: compozy/agh

Length of output: 571

---

🏁 Script executed:

```shell
# Find all usages of Backend as interface type (not just direct method calls)
rg -n '\bBackend\b' internal --type=go | grep -v test | head -30
```

Repository: compozy/agh

Length of output: 2884

---

🏁 Script executed:

```shell
# Find if Backend is used in daemon, API, or CLI layers
rg -n '\bBackend\b' --type=go | grep -E '(daemon|api|cli|layers)' | head -20
```

Repository: compozy/agh

Length of output: 2205

---

🏁 Script executed:

```shell
# Search explicitly for memory.Backend interface usage
rg -n 'memory\.Backend' --type=go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search for Backend interface parameter/field declarations (not just methods)
rg -n 'Backend\s+(interface|Backend)' internal/memory --type=go
```

Repository: compozy/agh

Length of output: 107

---

🏁 Script executed:

```shell
# Check what the Store interface looks like and if Backend is used
rg -n 'type Store struct' internal/memory/store.go --type=go -A10
```

Repository: compozy/agh

Length of output: 379

---

🏁 Script executed:

```shell
# Search in daemon, consolidation, and other layers for Backend usage
find internal -type d -name "daemon" -o -name "api" -o -name "consolidation" | head -5
rg -n '\bBackend\b' internal/memory/consolidation --type=go | head -15
```

Repository: compozy/agh

Length of output: 113

---

🏁 Script executed:

```shell
# Search for Backend as function parameter or return type
rg -n 'func.*Backend' internal --type=go -B2 | head -40
```

Repository: compozy/agh

Length of output: 2190

---

🏁 Script executed:

```shell
# Search for Backend in struct fields
rg -n '\bBackend\s+Backend' internal --type=go
```

Repository: compozy/agh

Length of output: 177

---

🏁 Script executed:

```shell
# Check consolidation package for usage
rg -n '\bBackend' internal/memory/consolidation --type=go -B1 -A1 | head -40
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for who uses the Store and if it's passed as Backend interface
rg -n 'NewStore\|\.Backend' internal/memory --type=go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Search specifically for memory.Backend as parameter or field in other files
rg -n 'Backend' internal/memory/types.go --type=go -A40 | tail -50
```

Repository: compozy/agh

Length of output: 1701

---

🏁 Script executed:

```shell
# Check if Backend is used in exported functions or API boundaries
rg -n '(func|type|var).*Backend\s' internal/memory --type=go -B1 -A3 | grep -v test
```

Repository: compozy/agh

Length of output: 451

---

🏁 Script executed:

```shell
# Search for any exported functions that might accept Backend parameter
rg -n 'func \(' internal/memory/store.go --type=go | grep -v ':' | head -10
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for daemon or API integration with memory package
rg -n 'memory\.' internal/daemon --type=go | head -20
rg -n 'memory\.' internal/api --type=go | head -20
```

Repository: compozy/agh

Length of output: 4111

---

🏁 Script executed:

```shell
# Search for where Backend interface might be used as an interface type parameter
rg -n 'Backend' internal/api/udsapi/server.go --type=go -B2 -A2
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check consolidation package which seems related
rg -n 'Backend\|Service' internal/memory/consolidation --type=go | head -40
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for Store or Backend being passed to daemon/api functions
rg -n 'NewStore\|WithMemory' internal/api/udsapi/server.go --type=go -B2 -A5
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the actual Store interface/type definition more carefully
cat -n internal/memory/store.go | head -50
```

Repository: compozy/agh

Length of output: 1516

---

🏁 Script executed:

```shell
# Look for where Store is injected into daemon or API
rg -n 'memoryStore\|memory.Store' internal/daemon/daemon.go --type=go | head -20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there are other implementations of Backend interface
rg -n 'func \(.*\) List\(scope Scope\)' internal --type=go
```

Repository: compozy/agh

Length of output: 136

---

🏁 Script executed:

```shell
# Check the full consolidation service which might use Backend
rg -n 'type Service' internal/memory/consolidation --type=go -A10 | head -40
```

Repository: compozy/agh

Length of output: 1530

---

🏁 Script executed:

```shell
# Verify if the proposed diff would break any existing callers in tests or implementations
rg -n '\.List\(|\.Read\(|\.Write\(|\.Delete\(|\.LoadPromptIndex\(' internal/memory/store_test.go --type=go | wc -l
```

Repository: compozy/agh

Length of output: 57

---

🏁 Script executed:

```shell
# Check if any code already tries to use Backend as interface type (not just Store)
rg -n 'var.*Backend|Backend\s*=|Backend\)' internal --type=go | grep -v 'environment.Backend' | grep -v test
```

Repository: compozy/agh

Length of output: 4570

---

🏁 Script executed:

```shell
# Final check: verify that the Store is indeed passed as *Store, not as Backend interface
rg -n 'WithMemoryStore|NewStore' internal/api/udsapi --type=go -B2 -A2
```

Repository: compozy/agh

Length of output: 3921

---

🏁 Script executed:

```shell
# Check daemon usage pattern as well
rg -n 'memStore|memory.Store' internal/daemon/daemon.go --type=go -B1 -A1 | head -30
```

Repository: compozy/agh

Length of output: 543

---



</details>

**Thread `context.Context` through the full `Backend` contract.**

`Search` and `Reindex` accept `context.Context`, but `List`, `Read`, `Write`, `Delete`, and `LoadPromptIndex` do not. Since this interface is exported and documents crossing daemon/API/CLI layer boundaries, the whole surface should be consistent with the guideline: "Use context.Context as first argument to functions crossing runtime boundaries." This refactoring should happen now before additional implementations depend on the current signature.

<details>
<summary>Proposed API shape</summary>

```diff
 type Backend interface {
-	List(scope Scope) ([]Header, error)
-	Read(scope Scope, filename string) ([]byte, error)
-	Write(scope Scope, filename string, content []byte) error
-	Delete(scope Scope, filename string) error
+	List(ctx context.Context, scope Scope) ([]Header, error)
+	Read(ctx context.Context, scope Scope, filename string) ([]byte, error)
+	Write(ctx context.Context, scope Scope, filename string, content []byte) error
+	Delete(ctx context.Context, scope Scope, filename string) error
 	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
 	Reindex(ctx context.Context, opts ReindexOptions) (ReindexResult, error)
-	LoadPromptIndex(scope Scope) (content string, truncated bool, err error)
+	LoadPromptIndex(ctx context.Context, scope Scope) (content string, truncated bool, err error)
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
// Backend captures the memory backend surface used by daemon, API, and CLI layers.
type Backend interface {
	List(ctx context.Context, scope Scope) ([]Header, error)
	Read(ctx context.Context, scope Scope, filename string) ([]byte, error)
	Write(ctx context.Context, scope Scope, filename string, content []byte) error
	Delete(ctx context.Context, scope Scope, filename string) error
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	Reindex(ctx context.Context, opts ReindexOptions) (ReindexResult, error)
	LoadPromptIndex(ctx context.Context, scope Scope) (content string, truncated bool, err error)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/types.go` around lines 87 - 96, The Backend interface is
inconsistent: Search and Reindex already accept context.Context but List, Read,
Write, Delete, and LoadPromptIndex do not; update the Backend interface so all
methods take context.Context as the first parameter (change List(scope Scope) to
List(ctx context.Context, scope Scope), Read(scope Scope, filename string) to
Read(ctx context.Context, scope Scope, filename string), Write(scope Scope,
filename string, content []byte) to Write(ctx context.Context, scope Scope,
filename string, content []byte), Delete(scope Scope, filename string) to
Delete(ctx context.Context, scope Scope, filename string), and
LoadPromptIndex(scope Scope) to LoadPromptIndex(ctx context.Context, scope
Scope) (and keep Search(ctx context.Context, ...) and Reindex(ctx
context.Context, ...) as-is); then update all implementations and call sites to
pass the ctx as the first argument and adjust any tests or mock types (refer to
the Backend interface and methods List, Read, Write, Delete, LoadPromptIndex,
Search, Reindex for locations to change).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The `memory.Backend` interface currently has no active consumers as an interface type anywhere in `internal/`; the live code paths use concrete `*Store` values.
  - Refactoring every backend method to take `context.Context` would force unrelated API and test churn outside this batch without fixing a demonstrated runtime defect.
  - The concrete issues in this batch are addressed more directly by fixing `Store`/`Backend` drift (issue 007), preserving write/delete success semantics after committed mutations (issue 008), and propagating actual cancellation in session prompt handling (issue 011).
