---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: valid
file: internal/cli/cli_integration_test.go
line: 1654
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-0Il0,comment:PRRC_kwDOR5y4QM687org
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Avoid echoing full command payloads in redaction-failure assertions.**

These `t.Fatalf(..., <raw output>)` calls can leak the exact secret into CI logs when the check fails. Keep the assertion message constant/sanitized.

 

<details>
<summary>Suggested change</summary>

```diff
- t.Fatalf("task next output exposed raw claim token: %s", nextOut)
+ t.Fatal("task next output exposed raw claim token")

- t.Fatalf("heartbeat output exposed raw claim token: %s", heartbeatOut)
+ t.Fatal("heartbeat output exposed raw claim token")

- t.Fatalf("complete output exposed raw claim token: %s", completeOut)
+ t.Fatal("complete output exposed raw claim token")
```
</details>

As per coding guidelines, "Raw `claim_token` ... must NEVER appear in logs, ... error payloads ... or memory."


Also applies to: 1688-1689, 1743-1744, 1843-1844, 2091-2092, 2106-2107

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/cli_integration_test.go` around lines 1653 - 1654, The failing
tests currently call t.Fatalf with the raw command output (e.g., t.Fatalf("task
next output exposed raw claim token: %s", nextOut)) which can leak secrets;
update each assertion that prints raw outputs (variables like nextOut, prevOut,
taskOut) to use a constant/sanitized message without interpolating the raw
payload (for example: t.Fatalf("task next output exposed raw claim token") or
include only a fixed redacted marker), and apply this change to all occurrences
referenced (around the checks at lines like the nextOut assertion and the other
listed ranges). Ensure you do not log or format the raw claim_token or full
command payload in any t.Fatalf or similar test failure message.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  Several redaction assertions print the full command output when a raw
  `claim_token` leak is detected. If the assertion ever fires, the failure log
  would reproduce the secret. Replace those failure messages with constant
  sanitized text while keeping the redaction checks.
