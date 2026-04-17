---
status: resolved
file: internal/bundles/service_test.go
line: 443
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57znol,comment:PRRC_kwDOR5y4QM645ilH
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the wrapped cause in `ShouldWrapBundleResourceListError`.**

This subtest only checks the message, so `fmt.Errorf("...: %v", err)` would still pass. Keep the injected error in a variable and add `errors.Is(err, resourceErr)` so the test actually protects `%w` behavior.


<details>
<summary>💡 Suggested test tightening</summary>

```diff
-		store := newMemoryStore()
-		store.listBundleResourcesHook = func() ([]resources.Record[BundleResourceSpec], error) {
-			return nil, errors.New("resource store offline")
-		}
+		store := newMemoryStore()
+		resourceErr := errors.New("resource store offline")
+		store.listBundleResourcesHook = func() ([]resources.Record[BundleResourceSpec], error) {
+			return nil, resourceErr
+		}
 		service := newMarketingService(store, WithLogger(discardBundleTestLogger()))
 
 		_, err := service.ListActivations(testutil.Context(t))
 		if err == nil {
 			t.Fatal("ListActivations() error = nil, want non-nil")
 		}
+		if !errors.Is(err, resourceErr) {
+			t.Fatalf("ListActivations() error = %v, want wrapped %v", err, resourceErr)
+		}
 		if !strings.Contains(err.Error(), "list bundle resources for activations") {
 			t.Fatalf("ListActivations() error = %v, want wrapped bundle resource context", err)
 		}
```
</details>
As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)" and "Focus on critical paths: workflow execution, state management, error handling".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		store := newMemoryStore()
		resourceErr := errors.New("resource store offline")
		store.listBundleResourcesHook = func() ([]resources.Record[BundleResourceSpec], error) {
			return nil, resourceErr
		}
		service := newMarketingService(store, WithLogger(discardBundleTestLogger()))

		_, err := service.ListActivations(testutil.Context(t))
		if err == nil {
			t.Fatal("ListActivations() error = nil, want non-nil")
		}
		if !errors.Is(err, resourceErr) {
			t.Fatalf("ListActivations() error = %v, want wrapped %v", err, resourceErr)
		}
		if !strings.Contains(err.Error(), "list bundle resources for activations") {
			t.Fatalf("ListActivations() error = %v, want wrapped bundle resource context", err)
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/bundles/service_test.go` around lines 430 - 443, The test should
assert the wrapped cause rather than only matching the string: define the
injected error (e.g., resourceErr := errors.New("resource store offline")), have
newMemoryStore().listBundleResourcesHook return that variable, call
service.ListActivations(...) and then use errors.Is(err, resourceErr) (or
errors.As) in addition to the existing message check to ensure the
implementation uses %w when wrapping; references: newMemoryStore,
listBundleResourcesHook, newMarketingService, ListActivations.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `ShouldWrapBundleResourceListError` only checks the rendered message, so it would still pass if the implementation stopped wrapping the underlying cause with `%w`.
- Fix plan: keep the injected resource error in a variable and assert `errors.Is(err, resourceErr)` alongside the existing context check.
- Resolution: the subtest now seeds a non-empty activation list, preserves the injected `resourceErr`, and asserts `errors.Is` so `%w` wrapping is enforced.
- Verification: `go test ./internal/bundles ./internal/environment/daytona ./internal/extension ./internal/tools` and `make verify` passed on 2026-04-17.
