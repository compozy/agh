---
status: resolved
file: packages/ui/src/components/search-input.test.tsx
line: 18
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcL,comment:PRRC_kwDOR5y4QM65JoyL
---

# Issue 029: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Typing assertion should validate accumulated value, not last character**

This test currently expects `"b"` on the second call, which does not reflect the intended `next` string semantics. It should assert `"ab"` (and ideally avoid pinning `value=""` unless the test also re-renders with updated state).



<details>
<summary>Proposed fix</summary>

```diff
-    render(<SearchInput value="" onChange={handle} placeholder="Search workspaces" />);
+    render(<SearchInput onChange={handle} placeholder="Search workspaces" />);
@@
-    expect(handle).toHaveBeenNthCalledWith(2, "b");
+    expect(handle).toHaveBeenNthCalledWith(2, "ab");
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/search-input.test.tsx` around lines 8 - 18, The
test for SearchInput incorrectly asserts the second onChange call receives "b"
instead of the accumulated value "ab"; update the assertions in
search-input.test.tsx to expect handle toHaveBeenNthCalledWith(2, "ab") and
ensure the test reflects the component's next-string semantics (either stop
pinning value="" or make the test render a controlled wrapper that updates value
on each onChange so the emitted values are accumulated). Target symbols:
SearchInput component, the test's handle mock, and the user.type call.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The current test pins `value=\"\"` without rerendering controlled state, so the DOM input never accumulates characters and the second callback only receives `"b"`.
  - Rework the test to exercise the component’s next-string semantics against actual accumulated input once uncontrolled usage is restored.
