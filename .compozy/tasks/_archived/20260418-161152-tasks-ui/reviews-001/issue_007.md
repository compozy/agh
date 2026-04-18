---
status: resolved
file: internal/api/core/tasks_surface_internal_test.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lbv,comment:PRRC_kwDOR5y4QM65B8fM
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
fd -t f "tasks_surface_internal_test.go" -x wc -l {}
```

Repository: compozy/agh

Length of output: 109

---

🏁 Script executed:

```shell
cat -n internal/api/core/tasks_surface_internal_test.go | head -100
```

Repository: compozy/agh

Length of output: 4020

---

🏁 Script executed:

```shell
sed -n '50-120p' internal/api/core/tasks_surface_internal_test.go
```

Repository: compozy/agh

Length of output: 106

---

🏁 Script executed:

```shell
grep -n "t.Run\|t.Parallel\|handlers\." internal/api/core/tasks_surface_internal_test.go | head -30
```

Repository: compozy/agh

Length of output: 921

---

🏁 Script executed:

```shell
cat -n internal/api/core/tasks_surface_internal_test.go | sed -n '51p;115p;144p;174p;215p'
```

Repository: compozy/agh

Length of output: 518

---

🏁 Script executed:

```shell
cat -n internal/api/core/tasks_surface_internal_test.go | sed -n '20p;22p;24p'
```

Repository: compozy/agh

Length of output: 271

---

🏁 Script executed:

```shell
grep -A 2 "type workspaceServiceStub" internal/api/core/tasks_surface_internal_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
grep -n "workspaceServiceStub" internal/api/core/tasks_surface_internal_test.go
```

Repository: compozy/agh

Length of output: 275

---

🏁 Script executed:

```shell
sed -n '190,220p' internal/api/core/tasks_surface_internal_test.go
```

Repository: compozy/agh

Length of output: 967

---



</details>

**Move handler initialization into each subtest or make the stub return an error.**

The workspace stub closes over the parent test's `*testing.T` and calls `t.Fatalf()` directly. When subtests run in parallel, failures in the stub are reported against the parent test context instead of the subtest that triggered them, breaking test isolation. Either initialize the handlers inside each `t.Run()` block or make the stub return an error and assert in the subtest.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_surface_internal_test.go` around lines 20 - 27, The
workspaceServiceStub passed into BaseHandlers (see BaseHandlers and
workspaceServiceStub with its get method) closes over the parent *testing.T and
calls t.Fatalf, which breaks isolation when subtests run in parallel; fix by
moving the handlers initialization into each t.Run subtest so each subtest
constructs its own BaseHandlers and stub bound to that subtest's t, or change
the stub's get implementation to return an error instead of calling t.Fatalf and
assert that error inside the specific subtest—ensure you update references to
the stub/get and remove direct t.Fatalf calls from code run outside the subtest.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The current `workspaceServiceStub` uses the parent test’s `t.Fatalf` through a closure that is shared by parallel subtests. That can attribute failures to the wrong test context and makes the helper harder to reuse safely. I’ll bind handler/stub setup to each subtest instead of closing over the parent `*testing.T`.
