---
status: resolved
file: extensions/bridges/gchat/provider.go
line: 1840
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Odc4,comment:PRRC_kwDOR5y4QM64G4Y3
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Also require `email_verified` on Pub/Sub bearer tokens.**

This path only checks issuer, audience, and `claims.Email`. Google’s Pub/Sub push-auth guidance explicitly says to validate both the expected service-account email and that `email_verified` is `true` after signature/audience verification; otherwise an unverified email claim still passes here. ([cloud.google.com](https://cloud.google.com/pubsub/docs/authenticate-push-subscriptions?utm_source=openai))

<details>
<summary>Suggested fix</summary>

```diff
 	if !issuerMatches(claims.Issuer, strings.TrimSpace(cfg.pubsubIssuer), "accounts.google.com", "https://accounts.google.com") {
 		return fmt.Errorf("gchat: pubsub bearer issuer %q did not match expected Google issuer", claims.Issuer)
 	}
+	if !claims.EmailVerified {
+		return errors.New("gchat: pubsub bearer email is not verified")
+	}
 	if !strings.EqualFold(strings.TrimSpace(claims.Email), strings.TrimSpace(cfg.pubsubServiceAccountEmail)) {
 		return fmt.Errorf("gchat: pubsub bearer email %q did not match expected service account %q", claims.Email, cfg.pubsubServiceAccountEmail)
 	}
 	return nil
```

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/gchat/provider.go` around lines 1835 - 1840, Add a check
that the Pub/Sub JWT's email is verified: after the existing issuerMatches and
email equality checks (in the same validation path using issuerMatches,
claims.Email and cfg.pubsubServiceAccountEmail), verify that
claims.EmailVerified is true (or non-nil and true if using a pointer) and return
a descriptive error like "gchat: pubsub bearer email %q is not verified" if not;
ensure you normalize/trimsame strings as you already do for the email comparison
and place the check after signature/audience verification so unverified email
claims are rejected.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `verifyPubSubBearerToken` validates signature, audience, issuer, and service-account email, but it does not reject tokens whose `email_verified` claim is false.
  - Root cause: the validation path accepts an unverified email claim as long as the email string matches.
  - Outcome: required `claims.EmailVerified` to be true and extended `extensions/bridges/gchat/provider_test.go` with a negative test for an unverified Pub/Sub token. That extra test file was outside the listed batch code files but was required to validate the production fix. Verified with `go test ./extensions/bridges/discord ./extensions/bridges/gchat` and `make verify`.
