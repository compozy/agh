---
status: resolved
file: internal/channels/delivery_projection_test.go
line: 483
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tbvd,comment:PRRC_kwDOR5y4QM624BPL
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's check if the file exists and examine the specific lines mentioned
wc -l internal/channels/delivery_projection_test.go
```

Repository: compozy/agh

Length of output: 104

---

🏁 Script executed:

```shell
# Check lines around 481-483
sed -n '475,490p' internal/channels/delivery_projection_test.go
```

Repository: compozy/agh

Length of output: 769

---

🏁 Script executed:

```shell
# Check lines around 623-625
sed -n '617,630p' internal/channels/delivery_projection_test.go
```

Repository: compozy/agh

Length of output: 703

---

🏁 Script executed:

```shell
# Find the definition of the sentinel errors mentioned
rg "ErrDeliveryNotFound|ErrDeliveryQueueSaturated" internal/channels/ -A 2 -B 2
```

Repository: compozy/agh

Length of output: 6025

---



</details>

**Use `errors.Is()` for sentinel error assertions.**

Direct equality checks with sentinel errors make tests brittle if `projectEventLocked` or `enqueueEventLocked` ever wrap errors with context. The codebase elsewhere in this same file already uses `errors.Is()` for these errors; these locations should be consistent.

<details>
<summary>Required changes</summary>

```diff
-	if _, ok, err := broker.projectEventLocked(nil, DeliveryProjectionEvent{Type: "agent_message"}); err != ErrDeliveryNotFound || ok {
+	if _, ok, err := broker.projectEventLocked(nil, DeliveryProjectionEvent{Type: "agent_message"}); !errors.Is(err, ErrDeliveryNotFound) || ok {
 		t.Fatalf("projectEventLocked(nil) = (%v, %v), want ErrDeliveryNotFound and ok=false", err, ok)
 	}
```

```diff
-	if err := broker.enqueueEventLocked(fullRoute, &activeDelivery{deliveryID: "del-c"}, start); err != ErrDeliveryQueueSaturated {
+	if err := broker.enqueueEventLocked(fullRoute, &activeDelivery{deliveryID: "del-c"}, start); !errors.Is(err, ErrDeliveryQueueSaturated) {
 		t.Fatalf("enqueueEventLocked(full route) error = %v, want ErrDeliveryQueueSaturated", err)
 	}
```

</details>

Per coding guidelines: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if _, ok, err := broker.projectEventLocked(nil, DeliveryProjectionEvent{Type: "agent_message"}); !errors.Is(err, ErrDeliveryNotFound) || ok {
		t.Fatalf("projectEventLocked(nil) = (%v, %v), want ErrDeliveryNotFound and ok=false", err, ok)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/delivery_projection_test.go` around lines 481 - 483,
Replace the direct sentinel error equality check in the test for
projectEventLocked with an errors.Is() check: change the condition that
currently compares err != ErrDeliveryNotFound to use !errors.Is(err,
ErrDeliveryNotFound) (and ensure the errors package is imported); apply the same
pattern wherever enqueueEventLocked or projectEventLocked sentinel errors are
compared to ErrDeliveryNotFound in this file to match existing usage.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  The current assertions compare sentinel errors directly even though the surrounding suite already uses `errors.Is` for the same broker sentinels. These tests are brittle if the broker later wraps those errors with context. The fix is to switch the affected sentinel checks in this file to `errors.Is`.
  Resolved in `internal/channels/delivery_projection_test.go` by converting the sentinel comparisons to `errors.Is`, then verified with `go test ./internal/channels -count=1` and `make verify`.
