---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: extensions/bridges/telegram/provider_test.go
line: 272
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGS,comment:PRRC_kwDOR5y4QM69ZeEt
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the specific missing-secret failure here.**

This passes on any non-nil error, so a regression back to the generic invalid-secret path would still look green.


<details>
<summary>💡 Proposed test tightening</summary>

```diff
-	if err := verifyWebhookSecret(context.Background(), req, nil, ""); err == nil {
-		t.Fatal("verifyWebhookSecret(missing configured secret) error = nil, want non-nil")
+	if err := verifyWebhookSecret(context.Background(), req, nil, ""); err == nil ||
+		!strings.Contains(err.Error(), "webhook secret is required") {
+		t.Fatalf("verifyWebhookSecret(missing configured secret) error = %v, want missing-secret error", err)
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
	if err := verifyWebhookSecret(context.Background(), req, nil, ""); err == nil ||
		!strings.Contains(err.Error(), "webhook secret is required") {
		t.Fatalf("verifyWebhookSecret(missing configured secret) error = %v, want missing-secret error", err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/telegram/provider_test.go` around lines 270 - 272, The
current test only checks for a non-nil error from verifyWebhookSecret; tighten
it to assert the specific missing-secret failure by asserting the returned error
matches the expected sentinel or contains the expected message. Update the
assertion in provider_test.go where verifyWebhookSecret is called to either use
errors.Is(err, expectedErr) if there is a package-level sentinel (e.g.,
ErrMissingSecret) or use a string containment check (e.g., require.True/if
!strings.Contains(err.Error(), "missing configured secret") then t.Fatalf) to
ensure the error is specifically the configured-secret-missing case rather than
any non-nil error.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the missing-secret case in `TestVerifyWebhookSecret` currently accepts any non-nil error, so a regression back to the generic invalid-secret path would still pass.
- Fix plan: assert the specific missing-secret error text returned by `verifyWebhookSecret`.
