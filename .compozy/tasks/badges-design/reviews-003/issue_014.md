---
status: pending
file: web/src/systems/session/adapters/session-api.ts
line: 155
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-O8FD,comment:PRRC_kwDOR5y4QM68JGPw
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use typed adapter errors instead of raw `Error` in `repairSession`.**

Line 152 and Line 154 throw raw `Error`, which makes adapter-level error handling less structured and inconsistent with system adapter contracts. Please switch these branches to typed session adapter errors.


As per coding guidelines `web/src/systems/*/adapters/*-api.ts`: "Typed error classes in adapters — never throw raw errors".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/adapters/session-api.ts` around lines 150 - 155, The
two raw Error throws in repairSession should be replaced with the adapter's
typed errors: throw a SessionNotFoundError (or similarly named not-found adapter
error) when response.status === 404, and throw a SessionAdapterError (or the
generic adapter error class used across systems) for the default failure branch
using defaultApiErrorMessage(response, error) as the message or as properties;
update imports to bring in the correct typed error classes (e.g.,
SessionNotFoundError, SessionAdapterError) and ensure you pass the
id/response/error context to those constructors instead of new Error.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
