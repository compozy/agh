---
status: resolved
file: packages/ui/src/components/sheet.test.tsx
line: 94
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcP,comment:PRRC_kwDOR5y4QM65JoyP
---

# Issue 032: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, locate and examine the sheet.test.tsx file
find . -name "sheet.test.tsx" -type f
```

Repository: compozy/agh

Length of output: 98

---

🏁 Script executed:

```shell
# Once found, read the specific lines mentioned (67-77)
cat -n packages/ui/src/components/sheet.test.tsx | sed -n '60,85p'
```

Repository: compozy/agh

Length of output: 730

---

🏁 Script executed:

```shell
# Search for other instances of console.error stubbing patterns in tests
rg "console\.error\s*=\s*" --type ts --type tsx -B 2 -A 5
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
# Search for console.error stubbing patterns (fix file type)
rg "console\.error\s*=" -B 2 -A 5 packages/ui
```

Repository: compozy/agh

Length of output: 4406

---

🏁 Script executed:

```shell
# Check if there are other test files with similar patterns
rg "console\.(error|warn|log)\s*=\s*\(\)" -B 2 -A 8
```

Repository: compozy/agh

Length of output: 3508

---



</details>

**Use `try/finally` to guarantee `console.error` restoration in throw assertions.**

If the assertion fails, console.error restoration won't execute, leaking the mock into subsequent tests. This pattern appears across multiple component tests (sheet, tooltip, popover, dialog) and needs consistent correction.

<details>
<summary>Suggested fix</summary>

```diff
   it("Should throw when SheetContent is used outside <Sheet>", () => {
     const originalError = console.error;
-    console.error = () => {};
-    expect(() =>
-      render(
-        <SheetContent>
-          <SheetTitle>orphan</SheetTitle>
-        </SheetContent>
-      )
-    ).toThrow(/Sheet\.\* components must be used inside <Sheet>/);
-    console.error = originalError;
+    try {
+      console.error = () => {};
+      expect(() =>
+        render(
+          <SheetContent>
+            <SheetTitle>orphan</SheetTitle>
+          </SheetContent>
+        )
+      ).toThrow(/Sheet\.\* components must be used inside <Sheet>/);
+    } finally {
+      console.error = originalError;
+    }
   });
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    const originalError = console.error;
    try {
      console.error = () => {};
      expect(() =>
        render(
          <SheetContent>
            <SheetTitle>orphan</SheetTitle>
          </SheetContent>
        )
      ).toThrow(/Sheet\.\* components must be used inside <Sheet>/);
    } finally {
      console.error = originalError;
    }
  });
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/sheet.test.tsx` around lines 67 - 77, The test
temporarily overrides console.error (originalError = console.error;
console.error = () => {}) before asserting render of orphan components like
SheetContent/SheetTitle; if the expect().toThrow assertion fails the override
isn’t restored. Wrap the render+expect block in a try/finally so console.error
is always reset to originalError in the finally clause; apply the same
try/finally pattern to the other similar tests (tooltip, popover, dialog) that
set console.error, ensuring you reference the same originalError variable and
restore it in finally.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - This test has the same `console.error` leak hazard as the popover orphan test: restoration is skipped if the assertion fails before the final assignment.
  - Fix by wrapping the temporary override in `try/finally`.
