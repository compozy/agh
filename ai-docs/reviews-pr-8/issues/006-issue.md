# Issue 6 - Review Thread Comment

**File:** `internal/skills/testdata/hooks/driver.sh:29`
**Date:** 2026-04-08 11:02:41 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the mismatch is real, but the proposed payload-default change would break the existing fixture contract and several hook tests that intentionally use `HOOK_TEST_OUTPUT`. The fix makes `value` an explicit case while preserving the current output semantics.

## Body

_⚠️ Potential issue_ | _🔴 Critical_

**Default output mode is broken (`value` never matches).**

`HOOK_TEST_OUTPUT_MODE` defaults to `value`, but the `case` branch is keyed as `payload`, so the default path falls into `*` and returns the wrong output.

<details>
<summary>🐛 Proposed fix</summary>

```diff
-case "${HOOK_TEST_OUTPUT_MODE:-value}" in
-	payload)
+case "${HOOK_TEST_OUTPUT_MODE:-value}" in
+	value|payload)
 		printf '%s' "$payload"
 		;;
```

</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
case "${HOOK_TEST_OUTPUT_MODE:-value}" in
	value|payload)
		printf '%s' "$payload"
		;;
	env)
		printf '%s' "${HOOK_TEST_CUSTOM_ENV:-}"
		;;
	combined)
		printf '%s|%s' "$payload" "${HOOK_TEST_CUSTOM_ENV:-}"
		;;
	*)
		printf '%s' "${HOOK_TEST_OUTPUT:-}"
		;;
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/testdata/hooks/driver.sh` around lines 17 - 29, The case for
HOOK_TEST_OUTPUT_MODE currently expects "payload" but the default expansion uses
"value", so the default path falls to the "*" branch and returns the wrong
variable; update the case patterns to treat "value" as an alias for "payload"
(for example add a branch matching "value|payload") or change the default
expansion to "payload", ensuring the payload-printing branch (the existing
payload branch) runs for the default case.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Resolve

Thread ID: `PRRT_kwDOR5y4QM55lKhi`

```bash
gh api graphql -f query='mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}' -F id=PRRT_kwDOR5y4QM55lKhi
```

---

_Generated from PR review - CodeRabbit AI_
