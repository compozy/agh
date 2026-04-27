---
status: resolved
file: web/src/systems/agent/components/agent-sessions-list.tsx
line: 148
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59tCVo,comment:PRRC_kwDOR5y4QM67bPPX
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**`0s` is currently rendered as missing data.**

Line 146 treats `0` as invalid (`seconds <= 0`), so newly started sessions display `—` instead of `0s`.



<details>
<summary>💡 Proposed fix</summary>

```diff
 function formatDuration(seconds: number | undefined | null): string {
-  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds <= 0) return "—";
+  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds < 0) return "—";
   const total = Math.round(seconds);
   if (total < 60) return `${total}s`;
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
function formatDuration(seconds: number | undefined | null): string {
  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds < 0) return "—";
  const total = Math.round(seconds);
  if (total < 60) return `${total}s`;
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/agent-sessions-list.tsx` around lines 145 -
148, formatDuration currently treats 0 as invalid because the guard checks
seconds <= 0, causing newly started sessions to show "—"; change the validation
in formatDuration to only reject negative or non-number values (e.g., use
seconds < 0 or explicitly test for negative numbers) or add a special-case that
returns "0s" when seconds === 0, so zero seconds render as "0s" instead of "—"
(update the function named formatDuration accordingly).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `formatDuration` rejects `seconds <= 0`, so a valid `0` elapsed duration renders the missing-data placeholder.
  - Zero elapsed seconds is a legitimate state for a newly started session and should render `0s`.
  - The fix is to reject only negative/non-finite/non-number values and add a focused regression assertion for the `0s` display.

## Resolution

- Updated `formatDuration` in `web/src/systems/agent/components/agent-sessions-list.tsx` to reject only negative, non-finite, or non-number values.
- Added a regression test proving `elapsed_seconds: 0` renders as `0s`.
- Verified with targeted Vitest and full `make verify`.
