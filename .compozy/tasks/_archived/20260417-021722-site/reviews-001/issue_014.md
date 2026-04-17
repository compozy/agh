---
status: resolved
file: packages/site/components/docs/doc-page-masthead.tsx
line: 17
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAP,comment:PRRC_kwDOR5y4QM64gE5R
---

# Issue 014: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Edge case: Empty string parts in slug could produce "undefined" text.**

If a slug contains an empty string part (e.g., from a malformed URL), `part[0]?.toUpperCase()` returns `undefined`, and concatenating with `part.slice(1)` produces the literal string `"undefined"`.


<details>
<summary>🛡️ Proposed defensive fix</summary>

```diff
 function toLabel(value?: string) {
-  if (!value) {
+  if (!value || value.length === 0) {
     return "Overview";
   }

   return value
     .split("-")
-    .map(part => part[0]?.toUpperCase() + part.slice(1))
+    .filter(part => part.length > 0)
+    .map(part => part[0].toUpperCase() + part.slice(1))
     .join(" ");
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
function toLabel(value?: string) {
  if (!value || value.length === 0) {
    return "Overview";
  }

  return value
    .split("-")
    .filter(part => part.length > 0)
    .map(part => part[0].toUpperCase() + part.slice(1))
    .join(" ");
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/docs/doc-page-masthead.tsx` around lines 8 - 17, The
toLabel function can produce "undefined" when a slug contains empty parts;
change it to defensively skip or handle empty segments by filtering out
zero-length parts before mapping (or checking part.length and returning an empty
string for that segment) so that only non-empty parts are transformed with
part[0].toUpperCase() + part.slice(1) and then joined; update the toLabel
implementation to use this check/filter to avoid concatenating "undefined" for
malformed slugs.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `toLabel()` can emit the literal string `"undefined"` when a segment contains consecutive dashes, because empty split parts are still mapped through `part[0]?.toUpperCase() + part.slice(1)`.
  - Root cause: the slug-to-label formatter does not filter zero-length parts before capitalization.
  - Fix plan: skip empty parts before mapping and add a `DocPageMasthead` regression test that covers malformed dashed input.
  - Resolution: `toLabel()` now filters empty slug parts before capitalization, and `doc-page-masthead.test.tsx` includes a malformed dashed-segment regression test.
  - Verification: `packages/site` `bun run test`, `bun run typecheck`, and `bun run build` passed.
