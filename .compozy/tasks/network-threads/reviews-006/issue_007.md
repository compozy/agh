---
provider: coderabbit
pr: "105"
round: 6
round_created_at: 2026-05-06T03:03:04.040959Z
status: resolved
file: internal/network/helpers_test.go
line: 212
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_1nZN,comment:PRRC_kwDOR5y4QM6-TX8a
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the actor-mismatch sentinel here.**

A non-opener hitting `OpenWork` is a permission/lifecycle failure, not a generic field-validation failure. Locking this test to `ErrInvalidField` makes it harder for callers to distinguish malformed envelopes from actor mismatch.

<details>
<summary>Suggested expectation</summary>

```diff
-	if _, err := OpenWork(withDirectSurface(openErrEnv), time.Time{}); !errors.Is(err, ErrInvalidField) {
-		t.Fatalf("OpenWork(non-opener) error = %v, want ErrInvalidField", err)
+	if _, err := OpenWork(withDirectSurface(openErrEnv), time.Time{}); !errors.Is(err, ErrWorkActorNotAllowed) {
+		t.Fatalf("OpenWork(non-opener) error = %v, want ErrWorkActorNotAllowed", err)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if _, err := OpenWork(withDirectSurface(openErrEnv), time.Time{}); !errors.Is(err, ErrWorkActorNotAllowed) {
		t.Fatalf("OpenWork(non-opener) error = %v, want ErrWorkActorNotAllowed", err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/network/helpers_test.go` around lines 210 - 212, The test currently
asserts OpenWork(withDirectSurface(openErrEnv), ...) returns ErrInvalidField,
but this case represents an actor/permission mismatch; update the assertion to
expect the actor-mismatch sentinel instead (use errors.Is(err, ErrActorMismatch)
or the project's defined ErrActorMismatch constant) when calling OpenWork with
withDirectSurface(openErrEnv) and time.Time{}; keep the rest of the call
(OpenWork, withDirectSurface, openErrEnv) unchanged so callers can distinguish
actor-mismatch from field-validation errors.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  The suggested sentinel change does not match the code path under test. This test calls `OpenWork(withDirectSurface(openErrEnv), ...)` where `openErrEnv.Kind == KindTrace`; `OpenWork()` rejects that input immediately because only `say` and `capability` may open work, returning `ErrInvalidField` before any participant or actor validation can happen. `ErrWorkActorNotAllowed` is exercised on the active-work path through `ApplyWorkEnvelope(...)` and is already covered in `internal/network/lifecycle_test.go` for real actor-mismatch scenarios. Changing this assertion would weaken the opener-kind contract rather than improve it.
  Analysis complete; no code change was required. The batch still passed fresh `make verify`.
