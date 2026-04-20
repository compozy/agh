---
status: resolved
file: packages/ui/src/components/dialog.test.tsx
line: 104
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcI,comment:PRRC_kwDOR5y4QM65JoyI
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
find . -name "dialog.test.tsx" -type f
```

Repository: compozy/agh

Length of output: 99

---

🏁 Script executed:

```shell
cat -n ./packages/ui/src/components/dialog.test.tsx | head -100
```

Repository: compozy/agh

Length of output: 3764

---

🏁 Script executed:

```shell
grep -n "console\.error" ./packages/ui/src/components/dialog.test.tsx
```

Repository: compozy/agh

Length of output: 169

---



</details>

**Wrap `console.error` restoration in `try/finally` to prevent test leakage.**

If the assertion fails before reaching line 86, `console.error` remains mocked, silently affecting subsequent tests. Use `try/finally` to ensure restoration happens regardless of whether the expected error is thrown.

<details>
<summary>Suggested fix</summary>

```diff
   it("Should throw when DialogContent is rendered outside <Dialog>", () => {
     const originalError = console.error;
-    console.error = () => {};
-    expect(() =>
-      render(
-        <DialogContent>
-          <DialogTitle>orphan</DialogTitle>
-        </DialogContent>
-      )
-    ).toThrow(/Dialog\.\* components must be used inside <Dialog>/);
-    console.error = originalError;
+    try {
+      console.error = () => {};
+      expect(() =>
+        render(
+          <DialogContent>
+            <DialogTitle>orphan</DialogTitle>
+          </DialogContent>
+        )
+      ).toThrow(/Dialog\.\* components must be used inside <Dialog>/);
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
          <DialogContent>
            <DialogTitle>orphan</DialogTitle>
          </DialogContent>
        )
      ).toThrow(/Dialog\.\* components must be used inside <Dialog>/);
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

In `@packages/ui/src/components/dialog.test.tsx` around lines 77 - 87, The test
temporarily replaces console.error before calling render on <DialogContent> with
<DialogTitle> to assert it throws; wrap the console.error assignment and the
expect(...) render call in a try/finally so console.error is restored no matter
what. Specifically, when you mock console.error (originalError = console.error;
console.error = () => {}), run the expect(() => render(...)) inside the try
block and restore console.error = originalError in the finally block to avoid
leaking the mock to other tests; keep the same render, DialogContent,
DialogTitle, and expect usage.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: The orphaned-`DialogContent` test replaces `console.error` and restores it only on the success path. If the expectation throws earlier, the mock leaks into later tests.
- Root cause: Temporary global mutation is not wrapped in cleanup that runs on both pass and failure paths.
- Fix plan: Wrap the `console.error` override in `try/finally` so restoration is guaranteed.

## Resolution

- Wrapped the `console.error` override in `packages/ui/src/components/dialog.test.tsx` with `try/finally` to prevent leakage across tests.
- Verified with `make verify` after all batch changes.
