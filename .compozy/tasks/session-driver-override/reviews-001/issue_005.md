---
status: resolved
file: internal/config/provider_test.go
line: 478
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581ayy,comment:PRRC_kwDOR5y4QM66RFOI
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert `ErrProviderUnavailable` in override-provider error path.**

On Line 469-478, the test validates message text but not error identity. Add `errors.Is(err, ErrProviderUnavailable)` so wrapped error semantics are guaranteed.

<details>
<summary>✅ Minimal assertion addition</summary>

```diff
 	_, err = cfg.ResolveSessionAgent(agent, "missing")
 	if err == nil {
 		t.Fatal("ResolveSessionAgent() error = nil, want unknown provider failure")
 	}
+	if !errors.Is(err, ErrProviderUnavailable) {
+		t.Fatalf("ResolveSessionAgent() error = %v, want ErrProviderUnavailable", err)
+	}
 	if !strings.Contains(err.Error(), `resolve session agent with provider "missing"`) {
 		t.Fatalf("ResolveSessionAgent() error = %q, want session override context", err.Error())
 	}
```
</details>

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	_, err = cfg.ResolveSessionAgent(agent, "missing")
	if err == nil {
		t.Fatal("ResolveSessionAgent() error = nil, want unknown provider failure")
	}
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("ResolveSessionAgent() error = %v, want ErrProviderUnavailable", err)
	}
	if !strings.Contains(err.Error(), `resolve session agent with provider "missing"`) {
		t.Fatalf("ResolveSessionAgent() error = %q, want session override context", err.Error())
	}
	if !strings.Contains(err.Error(), `unknown provider "missing"`) {
		t.Fatalf("ResolveSessionAgent() error = %q, want unknown provider detail", err.Error())
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/provider_test.go` around lines 469 - 478, The test for
ResolveSessionAgent currently only checks error text; update the assertion to
also assert specific error identity by using errors.Is(err,
ErrProviderUnavailable) (or testing helpers like require.True(t, errors.Is(...))
/ assert.ErrorIs) after the call to cfg.ResolveSessionAgent(agent, "missing") so
the test guarantees wrapped error semantics for ErrProviderUnavailable in
addition to the existing message checks; locate the failing test around
ResolveSessionAgent and add the errors.Is assertion referencing
ErrProviderUnavailable.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the unknown-provider override test currently checks only error text and does not assert the wrapped `ErrProviderUnavailable` identity.
- Fix plan: add an `errors.Is(err, ErrProviderUnavailable)` assertion while keeping the contextual message checks.
