---
status: resolved
file: internal/daemon/harness_detached_work.go
line: 523
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMw,comment:PRRC_kwDOR5y4QM65IPEE
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don’t include mutable `Reentry` state in the idempotency match.**

`expected` is built with `Reentry == nil`, but persisted runs can later gain `Reentry` metadata after processing. With the current full-struct equality check, the same submission key will stop deduping and start failing with “metadata does not match submission”.

<details>
<summary>One local fix</summary>

```diff
     current, err := decodeDetachedHarnessRunMetadata(run.Metadata)
     if err != nil {
         return err
     }
+    current.Reentry = nil
     if current != expected {
         return fmt.Errorf(
             "%w: detached harness run %q metadata does not match submission %q",
             taskpkg.ErrValidation,
             run.ID,
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func validateDetachedHarnessRunMatch(
    run taskpkg.Run,
    req normalizedDetachedHarnessSubmitRequest,
    origin taskpkg.Origin,
    expected detachedHarnessRunMetadata,
) error {
    if run.Origin != origin {
        return fmt.Errorf(
            "%w: detached harness run %q origin %q/%q does not match submission origin %q/%q",
            taskpkg.ErrValidation,
            run.ID,
            run.Origin.Kind,
            run.Origin.Ref,
            origin.Kind,
            origin.Ref,
        )
    }
    current, err := decodeDetachedHarnessRunMetadata(run.Metadata)
    if err != nil {
        return err
    }
    current.Reentry = nil
    if current != expected {
        return fmt.Errorf(
            "%w: detached harness run %q metadata does not match submission %q",
            taskpkg.ErrValidation,
            run.ID,
            req.SubmissionKey,
        )
    }
    return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_detached_work.go` around lines 494 - 523, The
equality check in validateDetachedHarnessRunMatch is failing because persisted
run metadata can gain a mutable Reentry after processing; change the comparison
to ignore the Reentry field when comparing decoded current (from
decodeDetachedHarnessRunMetadata(run.Metadata)) to expected (built with Reentry
== nil) — e.g., clear current.Reentry (or perform a field-by-field compare
excluding Reentry) before comparing, and keep the same error message using
run.ID and req.SubmissionKey when the remaining immutable metadata differs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `validateDetachedHarnessRunMatch` compares the full decoded detached-run metadata struct, including `Reentry`.
  - `Reentry` is intentionally mutable state populated after processing, so a later idempotent resubmission of the same work can fail even when all immutable submission fields still match.
  - I will ignore `Reentry` during the equality check and add regression coverage that resubmission still dedupes after reentry metadata is written.
