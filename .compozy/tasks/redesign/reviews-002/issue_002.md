---
status: resolved
file: packages/ui/playwright.config.ts
line: 49
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcD,comment:PRRC_kwDOR5y4QM65JoyD
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In Playwright Test config, when both top-level `use` and `projects[].use` define the same option (e.g., `viewport`), does `projects[].use` override the top-level value?`

💡 Result:

Yes, projects[].use overrides the top-level use for the same option, such as viewport. The project-specific use configuration is merged with the top-level use, where project values take precedence (override) for overlapping keys. This follows standard JavaScript object merge behavior (shallow merge), as evidenced by common config patterns where projects specify use: { ...devices['Desktop Chrome'] } to set specific viewport, browserName, etc., overriding any top-level defaults. Official docs describe top-level use as base/shared options and project use as specific, with examples using spread syntax (...) to inherit and override selectively. For instance, setting viewport in top-level and redefining it in project will use the project's value.

Citations:

- 1: https://playwright.dev/docs/test-projects
- 2: https://playwright.dev/docs/test-use-options
- 3: https://playwright.dev/docs/api/class-testconfig
- 4: https://playwright.dev/docs/api/class-testproject

---

🏁 Script executed:

```shell
cd packages/ui && head -60 playwright.config.ts | cat -n
```

Repository: compozy/agh

Length of output: 2291

---



</details>

**Remove the `use` property from the chromium project to inherit top-level configuration.**

Line 48 spreads `devices["Desktop Chrome"]`, which overrides the carefully configured top-level `use` settings (viewport: 1280x800, colorScheme: "dark", reducedMotion: "reduce", etc.) via shallow merge. This causes screenshot baselines to differ from the intended configuration. Since Playwright inherits top-level `use` settings when a project doesn't specify its own, removing this line allows the project to use the custom viewport and other settings already defined at the config level.

<details>
<summary>Proposed fix</summary>

```diff
   projects: [
     {
       name: "chromium",
-      use: { ...devices["Desktop Chrome"] },
     },
   ],
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  projects: [
    {
      name: "chromium",
    },
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/playwright.config.ts` around lines 45 - 49, Remove the
per-project override that spreads devices["Desktop Chrome"] so the "chromium"
project inherits the top-level Playwright configuration; locate the projects
array and the chromium project object (name: "chromium") and delete the use: {
...devices["Desktop Chrome"] } entry (or remove the use property entirely) so
the project relies on the top-level use settings (viewport, colorScheme,
reducedMotion, etc.).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: The top-level `use` block already spreads `devices["Desktop Chrome"]` and then overrides viewport, color scheme, reduced motion, and related screenshot settings. The per-project `use: { ...devices["Desktop Chrome"] }` shallow-merges over those overrides and resets several values.
- Root cause: The chromium project redundantly reintroduces the device preset, overriding the intended shared visual-test defaults.
- Fix plan: Remove the project-level `use` override so the chromium project inherits the validated top-level configuration.

## Resolution

- Removed the redundant chromium project `use` override so the project inherits the shared screenshot settings from the top-level Playwright config.
- Verified with `make verify` after all batch changes.
