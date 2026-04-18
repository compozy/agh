---
status: resolved
file: internal/api/core/tasks_surface_internal_test.go
line: 20
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lbu,comment:PRRC_kwDOR5y4QM65B8fL
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, locate and read the test file
fd -t f "tasks_surface_internal_test.go" | head -1 | xargs wc -l
```

Repository: compozy/agh

Length of output: 107

---

🏁 Script executed:

```shell
# Read the specific lines mentioned in the review
fd -t f "tasks_surface_internal_test.go" | head -1 | xargs cat -n | sed -n '1,30p'
```

Repository: compozy/agh

Length of output: 1057

---

🏁 Script executed:

```shell
# Also read around line 191-195
fd -t f "tasks_surface_internal_test.go" | head -1 | xargs cat -n | sed -n '185,220p'
```

Repository: compozy/agh

Length of output: 1314

---

🏁 Script executed:

```shell
# Check all usages of gin.SetMode in the codebase
rg "gin\.SetMode" -t go -A 2 -B 2
```

Repository: compozy/agh

Length of output: 6283

---

🏁 Script executed:

```shell
# Check if gin.SetMode is called with different modes in parallel tests
rg "gin\.SetMode" -t go --no-heading | sort | uniq -c
```

Repository: compozy/agh

Length of output: 1018

---

🏁 Script executed:

```shell
# Look at the actual gin package to understand SetMode behavior
# Check if there are any mutex protections or if it's truly unprotected
rg "func SetMode" -t go -A 5
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the workspace service stub usage more carefully
cat -n internal/api/core/tasks_surface_internal_test.go | sed -n '16,40p'
```

Repository: compozy/agh

Length of output: 1140

---

🏁 Script executed:

```shell
# Check if the Workspaces stub is actually called in the test flow
cat -n internal/api/core/tasks_surface_internal_test.go | sed -n '30,100p'
```

Repository: compozy/agh

Length of output: 3065

---

🏁 Script executed:

```shell
# Check if there's a TestMain or init function that sets gin mode globally
rg "TestMain\|func init" -t go internal/api/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look for any tests that use SetMode outside of t.Parallel() context
rg -B5 "gin\.SetMode" -t go internal/api/core/test_helpers_test.go
```

Repository: compozy/agh

Length of output: 161

---

🏁 Script executed:

```shell
# Check the helper test file more carefully
cat -n internal/api/core/test_helpers_test.go | head -50
```

Repository: compozy/agh

Length of output: 1427

---

🏁 Script executed:

```shell
# Check if the workspace stub is actually called and would cause issues
cat -n internal/api/core/tasks_surface_internal_test.go | sed -n '200,220p'
```

Repository: compozy/agh

Length of output: 770

---

🏁 Script executed:

```shell
# Check if gin.SetMode really needs to be called in every test or if it can be centralized
rg "TestMain" -t go internal/
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check what ParseTaskInboxQuery does - does it validate lane first?
rg "func ParseTaskInboxQuery" -t go -A 20
```

Repository: compozy/agh

Length of output: 1506

---

🏁 Script executed:

```shell
# Check the taskInboxDomainQuery implementation to see when workspace is called
rg "func.*taskInboxDomainQuery" -t go -A 30
```

Repository: compozy/agh

Length of output: 1915

---

🏁 Script executed:

```shell
# Look for gin package source or documentation about SetMode thread-safety
# Let's check if gin is mentioned in go.mod to understand the version
cat go.mod | grep gin
```

Repository: compozy/agh

Length of output: 135

---

🏁 Script executed:

```shell
# Check if there are any synchronization primitives around gin.SetMode or if tests ever conflict
rg "TestMode|ReleaseMode" -t go -B2 -A2 | head -60
```

Repository: compozy/agh

Length of output: 3293

---

🌐 Web query:

`gin-gonic SetMode thread safety race condition go`

💡 Result:

gin-gonic's SetMode function is thread-safe in recent versions of Gin. It uses sync/atomic operations to update the global mode variables: atomic.StoreInt32 for ginMode (int32) and atomic.Value.Store for modeName. This fix was implemented to address data race warnings reported in parallel tests, as seen in GitHub issue `#1180` and commit 7d14792 (merged around 2018, present in v1.10.0+ and master as of 2026). Older versions (pre-fix, e.g., v1.10.0 source shows non-atomic direct assignments) had data race issues when called concurrently, like in t.Parallel tests with -race flag. Best practice: Call SetMode early in main or init, before creating the router, as it's intended for single initialization, not concurrent calls. Concurrent calls from application code are not recommended, even if atomic now. No ongoing race conditions specific to SetMode in recent sources or issues.

Citations:

- 1: https://github.com/gin-gonic/gin/issues/1180
- 2: https://github.com/gin-gonic/gin/commit/7d147928ee232fce156ea7ce8ae6329e148aeb41
- 3: https://github.com/gin-gonic/gin/blob/master/mode.go
- 4: https://github.com/gin-gonic/gin/blob/v1.10.0/mode.go
- 5: https://github.com/gin-gonic/gin/blob/v1.12.0/mode.go

---



</details>

The parent `*testing.T` captured in the workspace service stub closure can report failures to the wrong test context when called from parallel subtests. Consider passing the subtest's context through the stub interface instead of closing over the parent test.

Also, the workspace service stub at lines 195–200 is unreachable in this test because the query at line 207 contains neither `scope` nor `workspace` parameters; `taskInboxDomainQuery` only calls the workspace lookup when both conditions are met. To test the workspace-not-found path, either provide matching query parameters or remove the unused stub setup.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_internal_test.go` around lines 16 - 20, The
workspace service stub in TestExpandedTaskQueryParsingAndDomainConversion closes
over the parent *testing.T (causing parallel-subtest reporting issues) and is
unreachable because taskInboxDomainQuery only triggers workspace lookup when
both scope and workspace params are present; either (A) change the stub
interface used by the test to accept a context or a *testing.T from the subtest
(pass the subtest t into the stub when calling
TestExpandedTaskQueryParsingAndDomainConversion or its subtests) so failures are
reported to the correct test, or (B) remove the unused workspace stub and
instead add matching query parameters (scope and workspace) to the test input so
taskInboxDomainQuery exercises the workspace-not-found path; locate the stub and
the call site by the symbols workspace service stub (closure) and
taskInboxDomainQuery in TestExpandedTaskQueryParsingAndDomainConversion to
implement the chosen fix.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The report mixes two observations, but both point to real test issues in this file. The shared workspace stub closes over the parent `*testing.T`, which is poor isolation for parallel subtests, and the workspace-not-found stub in the validation test is unreachable with the current request. I’ll restructure the test helpers per subtest and add a reachable workspace-resolution failure case.
