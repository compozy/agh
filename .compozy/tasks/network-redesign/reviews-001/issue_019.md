---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 291
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59QGuv,comment:PRRC_kwDOR5y4QM661FLl
---

# Issue 019: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Using `step` as the React key may cause issues with duplicate execution outline steps.**

If two steps have identical text, React will warn about duplicate keys and may incorrectly reuse DOM nodes.


<details>
<summary>🔧 Use index-based key for list items</summary>

```diff
-              {readStringList(capability, "execution_outline").map(step => (
+              {readStringList(capability, "execution_outline").map((step, stepIndex) => (
                   <p
                     className="text-[12px] leading-5 text-[color:var(--color-text-secondary)]"
-                    key={step}
+                    key={stepIndex}
                   >
                     {step}
                   </p>
                 ))}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
                {readStringList(capability, "execution_outline").map((step, stepIndex) => (
                  <p
                    className="text-[12px] leading-5 text-[color:var(--color-text-secondary)]"
                    key={stepIndex}
                  >
                    {step}
                  </p>
                ))}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-workspace-shell.tsx` around lines
284 - 291, The list rendering in network-workspace-shell.tsx uses the step
string as the React key (inside the map over readStringList(capability,
"execution_outline")), which can produce duplicate keys when steps repeat;
update the map to use a stable per-item index-based key (e.g.,
key={`${index}-${step}` or just key={index}) to guarantee uniqueness and avoid
React warnings and DOM reuse issues for identical step text.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `execution_outline` comes from dynamic payload data and can legitimately contain duplicate step text. Using the raw step string as the React key can therefore emit duplicate-key warnings and confuse reconciliation.
- Fix plan: use a stable per-item key that remains unique for repeated step text and add a route-level rendering test with duplicate outline steps to prove the warning disappears. That test lives in the existing network route test file outside the listed scope, so the expansion will be minimal and documented here.
- Resolution: changed execution outline keys to include the item index and added route-level coverage that renders duplicate step text without React duplicate-key warnings.
- Verification: `bun run test:raw src/routes/_app/-network.test.tsx src/systems/network/components/network-create-channel-dialog.test.tsx`, `make web-lint`, `make web-typecheck`, and `make verify`
