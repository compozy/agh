---
status: resolved
file: internal/api/httpapi/handlers_test.go
line: 941
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R107,comment:PRRC_kwDOR5y4QM663fBB
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid using the same production helper as the expected-value oracle.**

Line 938 computes `expectedProviders` with the same helper used by the handler, so this assertion can miss regressions in provider-option generation. Assert behavior independently from fixture inputs.

<details>
<summary>🧪 Suggested independent assertions</summary>

```diff
-	expectedProviders := core.SessionProviderOptionPayloadsFromConfig(&resolved.Config)
-	if !slices.Equal(response.Providers, expectedProviders) {
-		t.Fatalf("providers = %#v, want %#v", response.Providers, expectedProviders)
-	}
+	providerNames := make([]string, 0, len(response.Providers))
+	for _, item := range response.Providers {
+		providerNames = append(providerNames, item.Name)
+	}
+	if !slices.Contains(providerNames, "alpha") {
+		t.Fatalf("providers = %#v, want to include %q", response.Providers, "alpha")
+	}
+	if !slices.IsSorted(providerNames) {
+		t.Fatalf("provider names must be sorted: %#v", providerNames)
+	}
```
</details>
As per coding guidelines, "MUST test meaningful business logic, not trivial operations" and "Ensure tests verify behavior outcomes, not just function calls".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	providerNames := make([]string, 0, len(response.Providers))
	for _, item := range response.Providers {
		providerNames = append(providerNames, item.Name)
	}
	if !slices.Contains(providerNames, "alpha") {
		t.Fatalf("providers = %#v, want to include %q", response.Providers, "alpha")
	}
	if !slices.IsSorted(providerNames) {
		t.Fatalf("provider names must be sorted: %#v", providerNames)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/handlers_test.go` around lines 938 - 941, The test
currently calls core.SessionProviderOptionPayloadsFromConfig(&resolved.Config)
to derive expectedProviders, which mirrors the production helper and can hide
regressions; change the test to assert independently from that helper by
constructing the expected provider payloads directly from the test fixture
values (or hard‑coding the expected slice) and then compare to
response.Providers (check count, provider IDs/names and key fields rather than
reusing SessionProviderOptionPayloadsFromConfig), e.g., build the expected
[]SessionProviderOptionPayload based on resolved.Config fixture values and
assert equality/individual fields against response.Providers to validate
behavior.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- The test currently derives `expectedProviders` through `core.SessionProviderOptionPayloadsFromConfig`, which is the same helper the handler uses.
- That couples the assertion to the implementation and can hide regressions in provider filtering or ordering, so the expected provider set needs to be asserted independently from the fixture inputs.
- Resolved by asserting the exact provider-name list directly from the workspace fixture instead of reusing the production helper, then verifying the full batch with `make verify`.
