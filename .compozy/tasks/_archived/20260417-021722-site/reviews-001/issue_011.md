---
status: resolved
file: packages/site/app/global.css
line: 30
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAC,comment:PRRC_kwDOR5y4QM64gE49
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In CSS custom properties, is `--font-display: var(--font-display), "Playfair Display", serif;` a cyclic/self-referential declaration that makes the custom property invalid at computed-value time?`

💡 Result:

No, --font-display: var(--font-display), "Playfair Display", serif; is a cyclic/self-referential declaration. The custom property --font-display directly references itself via var(--font-display) in its own value, creating a dependency cycle (self-edge in the dependency graph). Per the CSS Custom Properties spec, this makes the custom property invalid at computed-value time, computing to the guaranteed-invalid value.

Citations:

- 1: https://www.w3.org/TR/css-variables-1/
- 2: https://w3.org/TR/css-variables-1
- 3: https://stackoverflow.com/questions/55717060/css-custom-properties-dependencies-not-resolving-why-also-did-i-discover-undo
- 4: https://stackoverflow.com/questions/51660196/unable-to-overwrite-css-variable-with-its-own-value

---



</details>

**Remove the self-referential `var(--font-display)` from this property.**

`--font-display: var(--font-display), ...` creates a cyclic custom-property declaration. Per the CSS Custom Properties spec, this makes the property invalid at computed-value time, resulting in a guaranteed-invalid value. Elements relying on `--font-display` will not receive the intended font.

The other font tokens (`--font-sans`, `--font-mono`) correctly reference different variables. Ensure `--font-display` references a corresponding variable (e.g., `var(--font-playfair)` or similar) or remove the variable reference entirely and use the font name directly.

<details>
<summary>🧰 Tools</summary>

<details>
<summary>🪛 Stylelint (17.7.0)</summary>

[error] 28-28: Expected "BlinkMacSystemFont" to be "blinkmacsystemfont" (value-keyword-case)

(value-keyword-case)

---

[error] 27-27: Unexpected unknown at-rule "@theme" (scss/at-rule-no-unknown)

(scss/at-rule-no-unknown)

</details>

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/app/global.css` around lines 27 - 30, The --font-display custom
property is self-referential (--font-display: var(--font-display), "Playfair
Display", serif) which creates a cyclic declaration and invalidates the value;
update the declaration inside the `@theme` inline block to remove the
self-reference and instead reference the correct source variable (e.g.,
var(--font-playfair)) or use the font name directly (e.g., "Playfair Display",
serif) so that --font-display resolves to a valid value.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The display-font theme token and the Next font variable currently share the same custom-property name, which makes the display-font mapping self-referential/ambiguous at the stylesheet boundary.
  - Root cause: `packages/site/app/layout.tsx` injects Playfair into `--font-display`, while `packages/site/app/global.css` also defines the theme token `--font-display` in terms of itself.
  - Fix plan: rename the Next font source variable to a distinct name (for example `--font-playfair`), map the theme token to that variable, and add a stylesheet regression test. This requires a minimal out-of-scope edit to `packages/site/app/layout.tsx` to fix the root cause cleanly.
  - Resolution: renamed the Next font source variable to `--font-playfair`, mapped the theme token to that distinct variable, and added `packages/site/app/global.test.ts` to lock the contract.
  - Verification: `packages/site` `bun run test`, `bun run typecheck`, and `bun run build` passed. Root `make verify` still fails outside this batch in untouched files `web/src/styles.test.ts` / `packages/ui/src/tokens.css` because the test expects `--radius: 0.5rem` while the token source defines `0.7rem`.
