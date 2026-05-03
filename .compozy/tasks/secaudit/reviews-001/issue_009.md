---
provider: coderabbit
pr: "90"
round: 1
round_created_at: 2026-05-03T03:31:47.363113Z
status: resolved
file: packages/site/public/install.sh
line: 5
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KSGY,comment:PRRC_kwDOR5y4QM69ZeE0
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Restrict the trusted cosign identity to release tags only.**

Allowing `refs/heads/main` here weakens the provenance policy for a release installer. This entrypoint should trust only tag-based release runs, otherwise a non-release `release.yml` execution on `main` is accepted by the same verifier.

 

<details>
<summary>Suggested fix</summary>

```diff
-COSIGN_CERT_IDENTITY_REGEXP='^https://github\.com/compozy/agh/\.github/workflows/release\.yml@refs/(heads/main|tags/v[0-9][A-Za-z0-9._-]*)$'
+COSIGN_CERT_IDENTITY_REGEXP='^https://github\.com/compozy/agh/\.github/workflows/release\.yml@refs/tags/v[0-9][A-Za-z0-9._-]*$'
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
COSIGN_CERT_IDENTITY_REGEXP='^https://github\.com/compozy/agh/\.github/workflows/release\.yml@refs/tags/v[0-9][A-Za-z0-9._-]*$'
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/public/install.sh` at line 5, Update the
COSIGN_CERT_IDENTITY_REGEXP in install.sh to remove acceptance of
refs/heads/main and only match tag-based release refs (e.g., refs/tags/v...);
locate the COSIGN_CERT_IDENTITY_REGEXP variable definition and adjust its regex
so it no longer includes the |refs/(heads/main) alternative and only allows the
refs/tags/v[0-9]... pattern for release tags.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the installer currently accepts provenance certificates for both release tags and `refs/heads/main`, which weakens the release-installer trust policy by allowing non-tag workflow runs to satisfy the same verifier.
- Fix plan: restrict the trusted identity regexp to release tags only and update the public installer contract test.
