# Issue 5 - Review Thread Comment

**File:** `internal/skills/provenance.go:40`
**Date:** 2026-04-08 12:09:55 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the provenance hash only covered `SKILL.md`, which left hook scripts and other payload files outside the integrity envelope. The fix adds deterministic directory hashing for the installed skill payload, updates install-time sidecar generation to use it, and keeps `VerifyHash` aligned with the full payload hash instead of the markdown file alone.

## Body

_⚠️ Potential issue_ | _🟠 Major_

**Verify the full installed skill payload, not just `SKILL.md`.**

`ComputeHash`/`VerifyHash` only cover the markdown file. Hooks and MCP declarations can execute auxiliary files from the same skill directory, so a modified script or helper binary will still look “verified” as long as `SKILL.md` stays unchanged. Please hash a manifest / directory contents, or at least every declared command target, before treating marketplace provenance as an integrity check.

Also applies to: 87-111

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/provenance.go` around lines 36 - 40, ComputeHash currently
only hashes SKILL.md; update it (and the corresponding VerifyHash logic) to
compute a deterministic hash of the full installed skill payload instead: gather
SKILL.md plus every file referenced by hooks and MCP declarations (or walk the
skill directory and build a sorted manifest of file paths and their contents),
concatenate in a stable order and compute the SHA-256 over that manifest; then
change VerifyHash to validate against that full-payload hash rather than just
SKILL.md. Ensure you update the functions named ComputeHash and VerifyHash (and
any callers between lines ~87-111 that verify hashes) to accept or derive the
list/manifest of files so integrity covers declared command targets and
auxiliary files.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55mbaf`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55mbaf
```

---

_Generated from PR review - CodeRabbit AI_
