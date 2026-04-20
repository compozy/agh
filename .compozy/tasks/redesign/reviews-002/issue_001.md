---
status: resolved
file: packages/ui/playwright.config.ts
line: 11
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcB,comment:PRRC_kwDOR5y4QM65JoyA
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Fail fast on invalid `AGH_UI_STORYBOOK_PORT`.**

Line 10 can produce `NaN`, which then contaminates Line 11 URL and webServer configuration.


<details>
<summary>Proposed fix</summary>

```diff
-const STORYBOOK_PORT = Number(process.env.AGH_UI_STORYBOOK_PORT ?? 6007);
+const rawStorybookPort = process.env.AGH_UI_STORYBOOK_PORT ?? "6007";
+const STORYBOOK_PORT = Number(rawStorybookPort);
+if (!Number.isInteger(STORYBOOK_PORT) || STORYBOOK_PORT < 1 || STORYBOOK_PORT > 65535) {
+  throw new Error(`Invalid AGH_UI_STORYBOOK_PORT: ${rawStorybookPort}`);
+}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
const rawStorybookPort = process.env.AGH_UI_STORYBOOK_PORT ?? "6007";
const STORYBOOK_PORT = Number(rawStorybookPort);
if (!Number.isInteger(STORYBOOK_PORT) || STORYBOOK_PORT < 1 || STORYBOOK_PORT > 65535) {
  throw new Error(`Invalid AGH_UI_STORYBOOK_PORT: ${rawStorybookPort}`);
}
export const STORYBOOK_URL = `http://127.0.0.1:${STORYBOOK_PORT}`;
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/playwright.config.ts` around lines 10 - 11, The current
STORYBOOK_PORT initialization may produce NaN and corrupt STORYBOOK_URL; update
the logic around STORYBOOK_PORT to validate the environment value: parse the
AGH_UI_STORYBOOK_PORT (e.g., Number or parseInt), check Number.isInteger and
that the port is in a valid range (1–65535), and if invalid throw an error or
exit early so the process fails fast; then construct STORYBOOK_URL using the
validated port variable (STORYBOOK_PORT) so the webServer config never receives
an invalid URL.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `playwright.config.ts` currently builds `STORYBOOK_URL` and the `webServer` command from `Number(process.env.AGH_UI_STORYBOOK_PORT ?? 6007)`. Invalid input yields `NaN`, which would produce an unusable URL and server port.
- Root cause: The environment variable is parsed without any integer or range validation.
- Fix plan: Validate the raw port value once, fail fast on invalid input, and reuse the validated port everywhere in the config.

## Resolution

- Added `parseStorybookPort` in `packages/ui/playwright.config.ts` and now fail fast on invalid `AGH_UI_STORYBOOK_PORT` before constructing URLs or the web server command.
- Verified with `make verify` after all batch changes.
