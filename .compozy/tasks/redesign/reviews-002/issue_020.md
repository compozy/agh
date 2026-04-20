---
status: resolved
file: packages/ui/src/components/empty.test.tsx
line: 42
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58EmKI,comment:PRRC_kwDOR5y4QM65P-uk
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify there are no optional-chained querySelector assertions that can pass with undefined.
# Expected after fix: no matches.
rg -nP 'expect\([^)]*\?\.[^)]*\)\.not\.toBeNull\(\)' packages/ui/src/components/empty.test.tsx
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
cat -n packages/ui/src/components/empty.test.tsx | sed -n '25,55p'
```

Repository: compozy/agh

Length of output: 1840

---

🌐 Web query:

`Jest toBeNull matcher undefined vs null behavior`

💡 Result:

The Jest toBeNull matcher strictly checks for the value null using exact equality (Object.is). It fails for undefined. - expect(null).toBeNull passes. - expect(undefined).toBeNull fails. Use toBeUndefined to specifically check for undefined. For either null or undefined, use toBeFalsy or expect(value == null).toBe(true). Examples from official docs: test('null', => { const n = null; expect(n).toBeNull; // passes expect(n).not.toBeUndefined; // passes }); test('undefined', => { let u; expect(u).toBeUndefined; // passes expect(u).not.toBeNull; // passes });

Citations:

- 1: https://jestjs.io/docs/using-matchers
- 2: https://jestjs.io/docs/en/using-matchers
- 3: https://jest-bot.github.io/jest/docs/using-matchers.html
- 4: https://runebook.dev/en/articles/jest/expect/tobenull
- 5: https://jestjs.io/docs/expect

---



</details>

**Prevent false-positive icon assertions when slot lookup fails.**

On Line 42, `iconSlot?.querySelector("svg")` can return `undefined` when `iconSlot` is `null`. Since `undefined` is not equal to `null`, `expect(undefined).not.toBeNull()` passes, creating a false positive even if `data-slot="empty-icon"` is missing.

Line 51 is already protected by the assertion on Line 50 that validates `iconSlot` first, so the optional chaining there is safe.

<details>
<summary>💡 Proposed fix</summary>

```diff
   it("Should fall back to a default icon when none is provided", () => {
     const { container } = render(<Empty title="Nothing here" />);
     const iconSlot = container.querySelector('[data-slot="empty-icon"]');
+    expect(iconSlot).not.toBeNull();
     expect(iconSlot?.querySelector("svg")).not.toBeNull();
   });
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    const iconSlot = container.querySelector('[data-slot="empty-icon"]');
    expect(iconSlot).not.toBeNull();
    expect(iconSlot?.querySelector("svg")).not.toBeNull();
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/empty.test.tsx` around lines 41 - 42, The test can
produce false-positive results because it asserts
expect(iconSlot?.querySelector("svg")).not.toBeNull() when iconSlot may be null;
change the test so you first assert iconSlot is present (e.g.,
expect(iconSlot).not.toBeNull()) and only then query for the SVG (using
iconSlot.querySelector("svg")), or else change the expectation to check for
truthiness of iconSlot before asserting about the SVG; update the assertions
around the iconSlot variable in empty.test.tsx (the iconSlot lookup and
subsequent SVG assertion) to ensure the slot existence is validated prior to
inspecting its children.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `expect(iconSlot?.querySelector("svg")).not.toBeNull()` can pass when `iconSlot` is `undefined`, so the test can report success even if the icon slot is missing.
- Root cause: Optional chaining is used before establishing that the queried slot exists.
- Fix plan: Assert the icon slot exists first, then assert on its SVG child without relying on the false-positive path.

## Resolution

- Updated `packages/ui/src/components/empty.test.tsx` to assert the icon slot exists before checking the fallback SVG.
- Verified with `make verify` after all batch changes.
