---
status: resolved
file: web/src/systems/network/components/network-create-channel-dialog.tsx
line: 102
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIej,comment:PRRC_kwDOR5y4QM66CAlJ
---

# Issue 017: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Mark the purpose input as required on the control itself.**

Right now the field is only "required" by surrounding state, so screen readers will not announce that requirement and native form validation will not help if this dialog gets reused elsewhere. Adding `required` (and ideally `aria-required`) fixes that.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-create-channel-dialog.tsx` around
lines 90 - 102, Add the required attribute (and aria-required="true") to the
Textarea control so assistive tech and native form validation recognize the
field as mandatory; update the Textarea with id "network-channel-purpose" (the
control using value={draft.purpose} and onChange={event =>
onPurposeChange(event.target.value)}) to include required and
aria-required="true" attributes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: the purpose textarea is required by surrounding submit logic, but the control itself is not marked `required`. That prevents assistive technology and native form validation from announcing the field correctly.
- Fix plan: add `required` and `aria-required="true"` to the purpose textarea and extend the existing component test file to assert the attributes. The test lives next to the component outside the listed scope, so that update will be kept minimal and documented here.
- Resolution: added `required` and `aria-required=\"true\"` to the purpose textarea and extended the dialog test to assert both attributes.
- Verification: `bun run test:raw src/routes/_app/-network.test.tsx src/systems/network/components/network-create-channel-dialog.test.tsx`, `make web-lint`, `make web-typecheck`, and `make verify`
