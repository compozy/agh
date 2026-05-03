---
provider: coderabbit
pr: "90"
round: 2
round_created_at: 2026-05-03T03:57:53.330715Z
status: resolved
file: packages/site/public/install.sh
line: 63
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KWjq,comment:PRRC_kwDOR5y4QM69Zj0S
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Tighten resolved tag validation to avoid malformed tag acceptance.**

`v[0-9]*` is broader than needed and can accept unexpected suffixes. Use the same strict tag class as your provenance policy to fail fast.


<details>
<summary>Suggested patch</summary>

```diff
   case "$resolved_tag" in
-    v[0-9]*)
+    v[0-9][A-Za-z0-9._-]*)
       printf '%s\n' "$resolved_tag"
       ;;
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/public/install.sh` around lines 56 - 63, The case that checks
resolved_tag uses the loose pattern v[0-9]* which allows unwanted suffixes;
update the validation in the install script to use the stricter tag pattern used
by the provenance policy (e.g., require v<major>.<minor>.<patch> and optional
canonical suffix rules) so malformed tags are rejected—modify the case for
resolved_tag and the associated fail call (which reports resolved_url) to use
that stricter regex/pattern and ensure the success branch only prints tags that
fully match the policy.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `resolve_latest_release_tag()` still accepts any `resolved_tag` matching `v[0-9]*`, which is broader than the provenance identity policy already enforced later in the script. That loose pattern can accept malformed tag suffixes instead of failing fast during `latest` resolution.
- Fix approach: tighten the accepted tag class in the installer to match the release provenance policy and update the site contract test so the stricter validation is locked in.
- Resolution: the installer’s `latest` tag resolver now accepts the same tag shape enforced by the cosign provenance identity policy, and the public installer contract test locks that behavior in.
- Verification: `bunx vitest run packages/site/lib/public-install-contract.test.ts`, `sh -n packages/site/public/install.sh`, `make verify`.
