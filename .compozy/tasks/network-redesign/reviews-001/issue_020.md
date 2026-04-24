---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 495
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59QGvA,comment:PRRC_kwDOR5y4QM661FL-
---

# Issue 020: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Composite key may collide if label-value pairs are duplicated.**

Using `${field.label}-${field.value}` as a key could produce duplicates if two fields share the same label and value.


<details>
<summary>🔧 Use index to guarantee uniqueness</summary>

```diff
-      {fields.map(field => (
-        <div className="space-y-1" key={`${field.label}-${field.value}`}>
+      {fields.map((field, index) => (
+        <div className="space-y-1" key={index}>
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
    <div className="space-y-3">
      {fields.map((field, index) => (
        <div className="space-y-1" key={index}>
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-workspace-shell.tsx` around lines
493 - 495, The key generation in the fields.map callback uses a composite
`${field.label}-${field.value}` which can collide; update the JSX in
network-workspace-shell.tsx where fields.map is used (the map callback creating
the <div key=...>) to use a stable unique identifier instead—preferably a
dedicated property like field.id if available, otherwise fall back to the
iteration index (e.g., use field.id or index) so keys are guaranteed unique and
React list reconciliation remains correct.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: `NetworkDetailFieldList` is fed by `summarizeChannelWireFields()` and `summarizePeerWireFields()`, and the current field producers generate unique labels for every rendered entry in a given list. With the present data model, `${field.label}-${field.value}` remains unique for reachable payloads in this component.
- Resolution: no code change. The current field producers already keep reachable detail entries unique for this component.
- Verification: `bun run test:raw src/routes/_app/-network.test.tsx src/systems/network/components/network-create-channel-dialog.test.tsx`, `make web-lint`, `make web-typecheck`, and `make verify`
