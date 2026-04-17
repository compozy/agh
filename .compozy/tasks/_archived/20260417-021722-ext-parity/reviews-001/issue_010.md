---
status: resolved
file: internal/api/contract/contract.go
line: 58
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQL,comment:PRRC_kwDOR5y4QM64dqGT
---

# Issue 010: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
rg -n -C3 'ProviderStateJSON|provider_state_json|SessionEnvironmentPayload|ProviderState' --type go
```

Repository: compozy/agh

Length of output: 50367

---

🏁 Script executed:

```shell
cat -n internal/api/contract/contract.go | head -80
```

Repository: compozy/agh

Length of output: 4082

---

🏁 Script executed:

```shell
rg -n "SessionEnvironmentPayload" --type go -A 15
```

Repository: compozy/agh

Length of output: 4041

---

🏁 Script executed:

```shell
rg -n "provider_state_json\|ProviderStateJSON" internal/api --type go -B 3 -A 3
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Remove or redact `ProviderStateJSON` before serialization in the session API.**

The conversion function `SessionEnvironmentPayloadFromMeta()` in `internal/api/core/conversions.go` (line 66) copies provider state verbatim to the API response without filtering. This exposes sandbox identifiers and internal backend state (e.g., `{"sandbox_id":"..."}`) to all session API clients. Either remove the field entirely from `SessionEnvironmentPayload` or redact sensitive keys before serialization.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/contract.go` around lines 49 - 58,
SessionEnvironmentPayload currently exposes ProviderStateJSON; update the
conversion in SessionEnvironmentPayloadFromMeta() to avoid returning raw
provider state by either removing ProviderStateJSON from the
SessionEnvironmentPayload struct or sanitizing it before assignment —
specifically parse the provider state JSON in
SessionEnvironmentPayloadFromMeta(), remove sensitive keys such as "sandbox_id"
(and any other internal backend identifiers), and only set ProviderStateJSON to
the redacted JSON (or leave it nil/omitted) so internal identifiers are not sent
to session API clients; ensure the change references the
SessionEnvironmentPayload type and the SessionEnvironmentPayloadFromMeta()
function so all callers remain consistent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The API conversion path currently copies `store.SessionEnvironmentMeta.ProviderState` verbatim into `SessionEnvironmentPayload.ProviderStateJSON` in `internal/api/core/conversions.go`. That exposes provider-private runtime state to clients. Fix requires a minimal out-of-scope production change in `internal/api/core/conversions.go` plus coverage updates in the scoped API tests.
