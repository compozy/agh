---
status: resolved
file: extensions/bridges/github/provider_test.go
line: 1054
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTP2,comment:PRRC_kwDOR5y4QM64dqF6
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "Inspect webhook signature + routing flow in provider implementation..."
rg -n -C6 'func verifyGitHubWebhookSignature|func selectGitHubIssueConfig|func \(.*\) handleWebhookRequest|webhookSecret|X-Hub-Signature-256' extensions/bridges/github/provider.go

echo
echo "Look for any-candidate signature acceptance loops:"
rg -n -C5 'for _, .* := range candidates|ConstantTimeCompare|hmac|sha256' extensions/bridges/github/provider.go
```

Repository: compozy/agh

Length of output: 5390

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Examine handleWebhookRequest and how signature verification is used ==="
rg -n -C10 'handleWebhookRequest' extensions/bridges/github/provider.go | head -80

echo
echo "=== Look for where verifyGitHubWebhookSignature is called ==="
rg -n -B5 -A10 'verifyGitHubWebhookSignature' extensions/bridges/github/provider.go | grep -A15 'verifyGitHubWebhookSignature('

echo
echo "=== Examine the test context around line 1050 ==="
sed -n '1010,1080p' extensions/bridges/github/provider_test.go
```

Repository: compozy/agh

Length of output: 4636

---



</details>

**Fix test name-behavior mismatch and address signature-validation binding gap**

The test name `TestGitHubProviderReconcileRejectsSharedWebhookPaths` contradicts the assertions (line 1050-1054), which now expect `configError == nil`—accepting shared paths.

More critically, the current signature validation (line 1648 in `verifyGitHubWebhookSignature`) accepts ANY candidate's secret, but routing (line 1562 in `selectGitHubIssueConfig`) selects by repository name from the webhook payload. This means a payload signed with Instance A's secret can be routed to Instance B if the `repository.full_name` field matches Instance B—Instance B's secret is never verified against the signature.

Either:
1. Rename the test to reflect acceptance of shared paths, OR
2. Restore rejection of shared webhook paths, OR  
3. Change signature verification to re-validate against the selected instance's secret after routing

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/github/provider_test.go` around lines 1050 - 1054, The
test name TestGitHubProviderReconcileRejectsSharedWebhookPaths is inconsistent
with assertions that expect configs[i].configError == nil and there is a real
security gap: verifyGitHubWebhookSignature currently accepts any candidate
secret but selectGitHubIssueConfig routes by repository.full_name, so an event
signed with Instance A's secret can be delivered to Instance B without
re-checking its secret. Fix by (preferably) updating the webhook handling flow:
after selectGitHubIssueConfig selects the target config (by
repository.full_name), re-run signature verification against that selected
config's secret (use the same verification logic in verifyGitHubWebhookSignature
but with the selectedConfig.Secret) before accepting/routing the payload; also
update tests (either rename TestGitHubProviderReconcileRejectsSharedWebhookPaths
to reflect acceptance, or change tests to assert rejection if you restore
shared-path rejection) so they match the new behavior and add a unit test that
ensures a payload signed with A's secret is rejected for B.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The production routing gap is real. `serveWebhookHTTP` verifies the signature against any config sharing the path, then `selectGitHubIssueConfig`/`selectGitHubReviewConfig` routes by payload repository name, so a payload signed with one instance secret can be routed to another instance on the same webhook path. Fix requires a minimal production change in `extensions/bridges/github/provider.go` to re-verify the selected config's secret after routing, plus test updates in `provider_test.go`.
