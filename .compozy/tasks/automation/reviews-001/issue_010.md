---
status: resolved
file: internal/automation/extension_test.go
line: 28
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0O,comment:PRRC_kwDOR5y4QM623e7a
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Assert the ext-prefix failure explicitly.**

Line 38 only checks for “some error”, so this case will still pass if `Validate("trigger_fire")` starts failing for an unrelated reason. Please assert the specific error/message for non-`ext.` events, and rename the subtests to the required `Should...` form while touching this table.  


As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases" and "MUST have specific error assertions (ErrorContains, ErrorAs)."


Also applies to: 32-46

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/extension_test.go` around lines 14 - 28, Update the
table-driven tests in internal/automation/extension_test.go so each test name
follows the "Should..." pattern (e.g., "Should reject built-in event names") and
for the case where Event is "session.stopped" (and generally any non-`ext.`
event) replace the loose wantErr check with a specific assertion that the
validation failure contains the ext-prefix error (use ErrorContains/ErrorAs
helper) when calling Validate("trigger_fire") on the ExtensionTriggerRequest;
refer to the table entries and the ExtensionTriggerRequest struct to locate and
modify the failing subtests.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The current extension-request table test only checks for “some error” on non-`ext.` events, so unrelated failures could satisfy the case. I will rename the subtests to the local `Should...` form and assert the specific ext-prefix validation message.
