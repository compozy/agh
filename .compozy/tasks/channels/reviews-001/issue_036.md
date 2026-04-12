---
status: resolved
file: internal/extensiontest/channel_adapter_harness.go
line: 786
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBL-,comment:PRRC_kwDOR5y4QM623eJZ
---

# Issue 036: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, find the file to verify it exists
fd -p "channel_adapter_harness.go" internal/extensiontest/
```

Repository: compozy/agh

Length of output: 104

---

🏁 Script executed:

```shell
# Read the file and check lines around 779-786
wc -l internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 109

---

🏁 Script executed:

```shell
# View the relevant section
sed -n '775,790p' internal/extensiontest/channel_adapter_harness.go
```

Repository: compozy/agh

Length of output: 337

---

🏁 Script executed:

```shell
# Search for Sessions struct and Stop method definition
rg -A 10 "func.*Sessions.*Stop" --type go
```

Repository: compozy/agh

Length of output: 3180

---

🏁 Script executed:

```shell
# Also search for the Sessions type definition to understand its methods
rg -B 5 -A 15 "type.*Sessions" --type go | head -100
```

Repository: compozy/agh

Length of output: 6191

---

🏁 Script executed:

```shell
# Search for the Stop method in session-related files
rg -B 2 -A 10 "func.*\(.*\).*Stop\(" internal/session/ --type go
```

Repository: compozy/agh

Length of output: 7687

---

🏁 Script executed:

```shell
# Also check what type h.Sessions is
rg -B 5 "Sessions" internal/extensiontest/channel_adapter_harness.go | head -50
```

Repository: compozy/agh

Length of output: 713

---



</details>

**Don't ignore `Sessions.Stop` error during cleanup.**

The `Stop` method returns an error that should not be discarded. Ignoring cleanup errors can hide resource leaks (processes, DB handles) that lead to confusing test failures later. Instead, fail the test when cleanup errors occur.

<details>
<summary>Suggested fix</summary>

```diff
 	for _, info := range h.Sessions.List() {
 		if info == nil {
 			continue
 		}
-		_ = h.Sessions.Stop(aghtestutil.Context(t), info.ID)
+		if err := h.Sessions.Stop(aghtestutil.Context(t), info.ID); err != nil {
+			t.Fatalf("Sessions.Stop(%q) error = %v", info.ID, err)
+		}
 	}
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (h *Harness) stopSessions(t testing.TB) {
	t.Helper()
	for _, info := range h.Sessions.List() {
		if info == nil {
			continue
		}
		if err := h.Sessions.Stop(aghtestutil.Context(t), info.ID); err != nil {
			t.Fatalf("Sessions.Stop(%q) error = %v", info.ID, err)
		}
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extensiontest/channel_adapter_harness.go` around lines 779 - 786,
The cleanup loop in Harness.stopSessions currently discards the error from
h.Sessions.Stop; change it to check the returned error and fail the test if
non-nil. In Harness.stopSessions, after skipping nil info and calling
h.Sessions.Stop(aghtestutil.Context(t), info.ID), capture the error and call
t.Fatalf (or t.Fatalff-style message) with the session ID and error (e.g.,
"failed to stop session %s: %v", info.ID, err) so cleanup failures are reported
instead of ignored.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: `Harness.stopSessions` discards `Sessions.Stop` errors during teardown. A failed stop can leave subprocesses or DB-backed session resources alive and make later tests fail for unrelated reasons.
- Root cause: Cleanup errors are ignored with `_ =`.
- Fix plan: Check `Sessions.Stop` and fail the test immediately if teardown cannot stop a live session.
- Resolution: Harness teardown now fails fast on stop errors and the updated harness passed targeted tests plus `make verify`.
