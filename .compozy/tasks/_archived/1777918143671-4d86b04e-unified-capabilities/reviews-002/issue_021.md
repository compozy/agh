---
status: resolved
file: internal/network/envelope_integration_test.go
line: 122
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyyh,comment:PRRC_kwDOR5y4QM654NpX
---

# Issue 021: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep the `capability` fixture as fixed wire JSON.**

Unlike the other cases, this input is now produced by the current `Envelope` marshaler. That means field/tag/omitempty drift in the encoder will be reflected in the fixture itself, so this test stops protecting wire compatibility for `kind="capability"`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/envelope_integration_test.go` around lines 106 - 122, The
test fixture for the "capability" case is being produced by mustEnvelopeBytes(…,
Envelope{…}) which uses the current marshaler and thus allows encoder drift;
change the test to use a fixed wire JSON blob instead of mustEnvelopeBytes.
Replace the raw: mustEnvelopeBytes(t, Envelope{...}) in the test case named
"capability" so that it loads or embeds the canonical JSON bytes (a literal JSON
string or a read-from-fixture helper) representing the exact on-the-wire
envelope for ProtocolV0/KindCapability (i.e., refer to the test case name
"capability", the use of mustEnvelopeBytes, the Envelope struct and
KindCapability to locate the code) so the fixture remains stable and does not
reflect encoder tag/omitempty changes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The `capability` fixture is currently built with `mustEnvelopeBytes(...)`, which reuses the production marshaler and lets encoder/tag/omitempty drift rewrite the fixture itself. That weakens the round-trip test's wire-compatibility coverage for `kind="capability"`.
  I will replace that case with a fixed canonical JSON blob so the fixture stays stable even if the encoder changes.
  Fixed and verified with targeted package tests plus `make verify`.
