---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/hooks/introspection_test.go
line: 197
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0DsF,comment:PRRC_kwDOR5y4QM6-RRZL
---

# Issue 022: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Use subtests for the new network descriptor matrix.**

This loop adds several independent cases, but one failure stops the rest and loses per-event reporting. Wrap each event assertion in `t.Run("Should ...")` so failures stay isolated and easier to diagnose.
 

<details>
<summary>♻️ Suggested structure</summary>

```diff
 	networkDescriptors := map[HookEvent]string{
 		HookNetworkThreadOpened:     "NetworkThreadOpenedPayload",
 		HookNetworkDirectRoomOpened: "NetworkDirectRoomOpenedPayload",
 		HookNetworkMessagePersisted: "NetworkMessagePersistedPayload",
 		HookNetworkWorkOpened:       "NetworkWorkOpenedPayload",
 		HookNetworkWorkTransitioned: "NetworkWorkTransitionedPayload",
 		HookNetworkWorkClosed:       "NetworkWorkClosedPayload",
 	}
 	for event, wantPayload := range networkDescriptors {
-		descriptor := byEvent[event]
-		if descriptor.Family != HookEventFamilyNetwork ||
-			descriptor.SyncEligible ||
-			descriptor.PayloadSchema != wantPayload ||
-			descriptor.PatchSchema != "NetworkObservationPatch" {
-			t.Fatalf("%s descriptor = %#v, want async network payload=%q", event, descriptor, wantPayload)
-		}
+		event := event
+		wantPayload := wantPayload
+		t.Run("Should describe "+string(event), func(t *testing.T) {
+			t.Parallel()
+
+			descriptor := byEvent[event]
+			if descriptor.Family != HookEventFamilyNetwork ||
+				descriptor.SyncEligible ||
+				descriptor.PayloadSchema != wantPayload ||
+				descriptor.PatchSchema != "NetworkObservationPatch" {
+				t.Fatalf("%s descriptor = %#v, want async network payload=%q", event, descriptor, wantPayload)
+			}
+		})
 	}
```
</details>

As per coding guidelines, "Use `t.Run('Should ...')` subtests with `t.Parallel` as default" and "Use table-driven test layout for Go tests with multiple scenarios".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	networkDescriptors := map[HookEvent]string{
		HookNetworkThreadOpened:     "NetworkThreadOpenedPayload",
		HookNetworkDirectRoomOpened: "NetworkDirectRoomOpenedPayload",
		HookNetworkMessagePersisted: "NetworkMessagePersistedPayload",
		HookNetworkWorkOpened:       "NetworkWorkOpenedPayload",
		HookNetworkWorkTransitioned: "NetworkWorkTransitionedPayload",
		HookNetworkWorkClosed:       "NetworkWorkClosedPayload",
	}
	for event, wantPayload := range networkDescriptors {
		event := event
		wantPayload := wantPayload
		t.Run("Should describe "+string(event), func(t *testing.T) {
			t.Parallel()

			descriptor := byEvent[event]
			if descriptor.Family != HookEventFamilyNetwork ||
				descriptor.SyncEligible ||
				descriptor.PayloadSchema != wantPayload ||
				descriptor.PatchSchema != "NetworkObservationPatch" {
				t.Fatalf("%s descriptor = %#v, want async network payload=%q", event, descriptor, wantPayload)
			}
		})
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/hooks/introspection_test.go` around lines 181 - 197, The loop over
networkDescriptors creates multiple independent assertions that stop on the
first failure; update the test to use table-driven subtests by iterating
networkDescriptors and calling t.Run for each event (e.g., using the map entries
and event key) so each case runs in its own subtest; inside each subtest run the
same assertions against byEvent[event] (checking Family ==
HookEventFamilyNetwork, SyncEligible == false, PayloadSchema == wantPayload,
PatchSchema == "NetworkObservationPatch") and call t.Parallel() at the top of
the subtest to follow the repo guideline.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the network descriptor loop in `TestHooksCatalogIncludesTypedDispatchDescriptors` is a multi-case assertion without subtests, so one failure hides the remaining event coverage and violates the repo’s Go test-shape rule.
- Fix approach: convert the network descriptor matrix into per-event `t.Run("Should ...")` subtests with `t.Parallel()` and keep the current assertions inside each case.
- Verification: fixed in scoped code and validated with fresh `make verify`.
