---
status: resolved
file: internal/api/core/network_test.go
line: 128
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TblT,comment:PRRC_kwDOR5y4QM624BCR
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Replace the no-op assertion branch (`deadline == 0`).**

`deadline` is hardcoded to a non-zero constant, so this condition never validates behavior. It weakens the test signal.


<details>
<summary>Suggested cleanup</summary>

```diff
-	deadline := int64(1775823000)
@@
-		if statusPayload.Network.KindMetrics[0].Sent != 4 || deadline == 0 {
+		if statusPayload.Network.KindMetrics[0].Sent != 4 || statusPayload.Network.KindMetrics[0].Kind != string(network.KindSay) {
 			t.Fatalf("kind metrics = %#v", statusPayload.Network.KindMetrics)
 		}
```
</details>


Also applies to: 232-233

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_test.go` around lines 127 - 128, The test contains
a no-op branch that checks if deadline == 0 which never occurs because deadline
is hardcoded; remove that branch and replace it with a concrete assertion that
deadline equals the expected value (either the hardcoded constant 1775823000 or
compute expectedDeadline := fixedNow.Unix() and assert equality). Update both
occurrences where variables fixedNow and deadline are used so the test asserts
the correct non-zero deadline rather than skipping verification.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: The `deadline == 0` branch is dead code because the test hardcodes a non-zero deadline, so that assertion does not validate any runtime behavior.
- Fix plan: Replace the dead branch with a concrete assertion on the converted payload fields so the test exercises real metadata preservation.
