---
status: resolved
file: internal/observe/helpers_test.go
line: 366
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCa,comment:PRRC_kwDOR5y4QM61T6H1
---

# Issue 018: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Make the fake resolver verify the requested workspace reference.**

Right now this double returns the same `ResolvedWorkspace` for every input, so these tests would still pass if the production code started calling `Resolve` with a root path or empty string again. Recording the expected ref/ID and failing on mismatches will keep the new contract protected.


As per coding guidelines, "Check dependent package APIs before writing integration code or tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/observe/helpers_test.go` around lines 349 - 366, The
fakeObserveWorkspaceResolver should validate the incoming workspace reference
instead of always returning the same ResolvedWorkspace: add an expectedRef (or
expectedID) field to fakeObserveWorkspaceResolver and in both Resolve and
ResolveOrRegister compare the passed string argument to that expected value,
returning an error (or a test-friendly mismatch error) if they differ; otherwise
return r.resolved and r.err as before — update tests to set expectedRef when
constructing the fake so calls that pass empty/root refs will fail.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The fake resolver currently returns the same workspace regardless of the input reference.
  - That means the tests would keep passing if production code accidentally stopped using the workspace ID they are meant to protect.
  - I will make the fake assert the expected workspace reference and update the relevant tests to set it explicitly.
