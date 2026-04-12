---
status: resolved
file: internal/channels/delivery_projection_test.go
line: 450
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tkbd,comment:PRRC_kwDOR5y4QM624L_J
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Assert the expected validation error, not just `err != nil`.**

Lines 419, 437, and 448 only prove that *some* validation failed. These cases will still pass if `Validate()` starts returning an unrelated error, which weakens the regression signal for the exact invariant under test.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/delivery_projection_test.go` around lines 416 - 450, Update
the three failing tests to assert the specific validation error instead of only
checking err != nil: in ShouldRejectSnapshotWhenLastAckedExceedsLastSent assert
that snapshot.Validate() returns an error whose message contains the specific
phrase/validation identifier for last-acked > last-sent (use ErrorContains or
ErrorAs against the validation error value used in Validate); in
ShouldRejectMissingSnapshot assert that newResumeRequest(snapshot).Validate()
returns the missing-snapshot validation error (check error message or error type
for "missing snapshot"); and in ShouldRejectMismatchedSnapshotDeliveryID assert
that req.Validate() returns the mismatched-delivery-id validation error (check
for the delivery id mismatch message/type). Reference the calls to
snapshot.Validate() and req.Validate() in the tests and replace the generic
nil-checks with ErrorContains/ErrorAs assertions against the exact validation
error text or exported error variable used in the code.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  The three validation tests currently accept any non-nil error, which weakens the regression signal and can hide unrelated failures. `DeliverySnapshot.Validate()` and `DeliveryRequest.Validate()` already return stable, specific validation messages for these invariants.
  Resolved in `internal/channels/delivery_projection_test.go` by asserting the specific validation messages for the last-acked invariant, missing snapshot, and mismatched delivery ID cases. Verified with `go test ./internal/channels -count=1` and the final `make verify` pass.
