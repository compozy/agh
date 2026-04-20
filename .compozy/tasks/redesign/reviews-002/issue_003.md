---
status: resolved
file: packages/ui/scripts/serve-storybook.ts
line: 6
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcF,comment:PRRC_kwDOR5y4QM65JoyF
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate port input before starting the server.**

Line 5 accepts any `Number(...)` result; invalid env/arg values can become `NaN` (or out-of-range) and break startup.


<details>
<summary>Proposed fix</summary>

```diff
-const portArg = Number(process.argv[3] ?? process.env.AGH_UI_STORYBOOK_PORT ?? 6007);
+const rawPort = process.argv[3] ?? process.env.AGH_UI_STORYBOOK_PORT ?? "6007";
+const portArg = Number(rawPort);
+if (!Number.isInteger(portArg) || portArg < 1 || portArg > 65535) {
+  console.error(`[serve-storybook] invalid port: ${rawPort}`);
+  process.exit(1);
+}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
const rawPort = process.argv[3] ?? process.env.AGH_UI_STORYBOOK_PORT ?? "6007";
const portArg = Number(rawPort);
if (!Number.isInteger(portArg) || portArg < 1 || portArg > 65535) {
  console.error(`[serve-storybook] invalid port: ${rawPort}`);
  process.exit(1);
}
const hostArg = process.argv[4] ?? "127.0.0.1";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/scripts/serve-storybook.ts` around lines 5 - 6, portArg is
currently set with Number(...) which can produce NaN or out-of-range values;
update the logic that defines portArg (and keep hostArg) to parse and validate
the port: use parseInt on process.argv[3] ?? process.env.AGH_UI_STORYBOOK_PORT
?? "6007", check Number.isInteger and that the value is between 1 and 65535, and
if invalid either fallback to 6007 (and log a warning) or exit with an error;
ensure the validated numeric port is used wherever portArg is referenced (e.g.,
server start code) so startup won't proceed with NaN or invalid ports.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `serve-storybook.ts` parses the port with `Number(...)` and passes it directly to `Bun.serve`. Invalid CLI/env input can become `NaN` or an out-of-range value and break startup in a non-obvious way.
- Root cause: The script trusts unvalidated external input for its listening port.
- Fix plan: Parse the raw port string once, verify it is an integer within `1..65535`, and exit with a clear error when invalid.

## Resolution

- Added explicit numeric/range validation for the Storybook server port in `packages/ui/scripts/serve-storybook.ts` and exit early with a clear error on invalid input.
- Verified with `make verify` after all batch changes.
