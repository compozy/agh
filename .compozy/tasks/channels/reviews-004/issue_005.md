---
status: resolved
file: internal/daemon/daemon_test.go
line: 3148
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tt64,comment:PRRC_kwDOR5y4QM624Xyw
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate delivery payloads before recording or ACKing them.**

`internal/channels/delivery_types.go` already gives you `DeliveryRequest.Validate()` for resume/snapshot and ID-consistency checks. Skipping it here means the helper can happily ACK malformed `channels/deliver` payloads, which makes daemon-side protocol bugs look green in integration tests.



<details>
<summary>Possible fix</summary>

```diff
 	case "channels/deliver":
 		var params channelspkg.DeliveryRequest
 		if err := json.Unmarshal(req.Params, &params); err != nil {
-			return false, err
+			return false, fmt.Errorf("decode channels/deliver request: %w", err)
+		}
+		if err := params.Validate(); err != nil {
+			return false, fmt.Errorf("validate channels/deliver request: %w", err)
 		}
 		if err := h.recordDelivery(params); err != nil {
 			return false, err
 		}
 
 		ack := channelspkg.DeliveryAck{
 			DeliveryID: strings.TrimSpace(params.Event.DeliveryID),
 			Seq:        params.Event.Seq,
 		}
+		if err := ack.ValidateFor(params.Event); err != nil {
+			return false, fmt.Errorf("build channels/deliver ack: %w", err)
+		}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3120 - 3148, The test helper
currently unmarshals a channelspkg.DeliveryRequest into params and proceeds to
record and ACK it without validating; call params.Validate() (the existing
method on DeliveryRequest from internal/channels/delivery_types.go) immediately
after json.Unmarshal and before
h.recordDelivery/sendResult/sendDelayedDeliveryResult, return the validation
error (or fail the request) if it returns non-nil so malformed deliveries are
rejected rather than ACKed; keep the subsequent logic (ack construction,
scenario branches, and calls to h.recordDelivery, h.sendDelayedDeliveryResult,
h.sendResult) unchanged except gated by the successful Validate() call.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The `channels/deliver` helper path in `internal/daemon/daemon_test.go` unmarshals `channelspkg.DeliveryRequest` and proceeds directly to recording and ACK generation.
  - `internal/channels/delivery_types.go` already defines `DeliveryRequest.Validate()`, and skipping it allows malformed payloads to be recorded and acknowledged by the helper, which can hide daemon-side protocol bugs in tests.
  - Fix approach: validate the decoded delivery request before recording or acknowledging it, return contextual validation errors, and add a helper-level regression test that malformed deliveries are rejected.

## Resolution

- Added `DeliveryRequest.Validate()` to the helper `channels/deliver` path immediately after decode, with contextual error wrapping.
- Added a regression test that malformed resume deliveries are rejected before any marker write or ACK output occurs.
- Verified with `go test ./internal/daemon` and `make verify`.
