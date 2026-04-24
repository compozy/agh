---
status: resolved
file: internal/daemon/daemon_test.go
line: 2075
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RYXn,comment:PRRC_kwDOR5y4QM6622vD
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this test body in a `t.Run("Should...")` subtest.**

The behavior being checked is good, but this test case should follow the enforced subtest naming pattern.



<details>
<summary>✅ Suggested structure</summary>

```diff
 func TestFakeSessionManagerDeleteTracksDeleteIndependently(t *testing.T) {
 	t.Parallel()
-
-	manager := &fakeSessionManager{
-		infos: []*session.Info{{ID: "sess-a"}, {ID: "sess-b"}},
-	}
-
-	if err := manager.Delete(testutil.Context(t), "sess-a"); err != nil {
-		t.Fatalf("Delete() error = %v", err)
-	}
-
-	if got, want := len(manager.deleteCalls), 1; got != want {
-		t.Fatalf("len(deleteCalls) = %d, want %d", got, want)
-	}
-	if got, want := manager.deleteCalls[0], "sess-a"; got != want {
-		t.Fatalf("deleteCalls[0] = %q, want %q", got, want)
-	}
-	if got := len(manager.stopCalls); got != 0 {
-		t.Fatalf("len(stopCalls) = %d, want 0", got)
-	}
-	if got, want := len(manager.infos), 1; got != want {
-		t.Fatalf("len(infos) = %d, want %d", got, want)
-	}
-	if got, want := manager.infos[0].ID, "sess-b"; got != want {
-		t.Fatalf("infos[0].ID = %q, want %q", got, want)
-	}
+	t.Run("ShouldTrackDeleteIndependentlyFromStop", func(t *testing.T) {
+		t.Parallel()
+
+		manager := &fakeSessionManager{
+			infos: []*session.Info{{ID: "sess-a"}, {ID: "sess-b"}},
+		}
+
+		if err := manager.Delete(testutil.Context(t), "sess-a"); err != nil {
+			t.Fatalf("Delete() error = %v", err)
+		}
+
+		if got, want := len(manager.deleteCalls), 1; got != want {
+			t.Fatalf("len(deleteCalls) = %d, want %d", got, want)
+		}
+		if got, want := manager.deleteCalls[0], "sess-a"; got != want {
+			t.Fatalf("deleteCalls[0] = %q, want %q", got, want)
+		}
+		if got := len(manager.stopCalls); got != 0 {
+			t.Fatalf("len(stopCalls) = %d, want 0", got)
+		}
+		if got, want := len(manager.infos), 1; got != want {
+			t.Fatalf("len(infos) = %d, want %d", got, want)
+		}
+		if got, want := manager.infos[0].ID, "sess-b"; got != want {
+			t.Fatalf("infos[0].ID = %q, want %q", got, want)
+		}
+	})
 }
```
</details>

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestFakeSessionManagerDeleteTracksDeleteIndependently(t *testing.T) {
	t.Parallel()

	t.Run("ShouldTrackDeleteIndependentlyFromStop", func(t *testing.T) {
		t.Parallel()

		manager := &fakeSessionManager{
			infos: []*session.Info{{ID: "sess-a"}, {ID: "sess-b"}},
		}

		if err := manager.Delete(testutil.Context(t), "sess-a"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if got, want := len(manager.deleteCalls), 1; got != want {
			t.Fatalf("len(deleteCalls) = %d, want %d", got, want)
		}
		if got, want := manager.deleteCalls[0], "sess-a"; got != want {
			t.Fatalf("deleteCalls[0] = %q, want %q", got, want)
		}
		if got := len(manager.stopCalls); got != 0 {
			t.Fatalf("len(stopCalls) = %d, want 0", got)
		}
		if got, want := len(manager.infos), 1; got != want {
			t.Fatalf("len(infos) = %d, want %d", got, want)
		}
		if got, want := manager.infos[0].ID, "sess-b"; got != want {
			t.Fatalf("infos[0].ID = %q, want %q", got, want)
		}
	})
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 2048 - 2075, Wrap the existing
TestFakeSessionManagerDeleteTracksDeleteIndependently body in a t.Run(...)
subtest using the "Should..." naming pattern (e.g. t.Run("Should delete tracks
independently", func(t *testing.T) { ... })), moving t.Parallel() inside that
subtest and keeping the same assertions that exercise fakeSessionManager, its
Delete method, manager.deleteCalls, manager.stopCalls and manager.infos so
behavior is unchanged but the test now follows the required subtest naming
convention.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/daemon/daemon_test.go` still has a top-level test body for `TestFakeSessionManagerDeleteTracksDeleteIndependently`, while the repository rule for Go tests is `t.Run("Should...")` subtests.
  - This is a style-and-consistency fix only; behavior should remain unchanged.
  - Planned fix: wrap the assertions in a named subtest and move the inner `t.Parallel()` there.

## Resolution

- Wrapped `TestFakeSessionManagerDeleteTracksDeleteIndependently` in `t.Run("ShouldTrackDeleteIndependentlyFromStop", ...)` and moved the subtest-local `t.Parallel()` inside the named case.
- Preserved the existing assertions and fake-session behavior; this change is structural only.
- Verified with `make verify` (exit `0`).
