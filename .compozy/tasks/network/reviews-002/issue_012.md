---
status: resolved
file: internal/network/lifecycle.go
line: 44
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56T_fF,comment:PRRC_kwDOR5y4QM624toT
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/network/lifecycle.go | head -250
```

Repository: compozy/agh

Length of output: 9554

---



</details>

**Wrap propagated errors with operation context.**

Six error returns lack wrapping, violating the guideline: "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`". This degrades debuggability and creates inconsistency with other wrapped returns in the same file.

<details>
<summary>Proposed patch</summary>

```diff
@@
 	if err := ValidateSpace(i.Space); err != nil {
-		return err
+		return fmt.Errorf("validate interaction space: %w", err)
 	}
@@
 	if err := i.State.Validate(); err != nil {
-		return err
+		return fmt.Errorf("validate interaction state: %w", err)
 	}
@@
 	if err := interaction.Validate(); err != nil {
-		return Interaction{}, err
+		return Interaction{}, fmt.Errorf("open interaction validation failed: %w", err)
 	}
@@
 	if err := interaction.Validate(); err != nil {
-		return LifecycleResult{}, err
+		return LifecycleResult{}, fmt.Errorf("validate current interaction: %w", err)
 	}
@@
 		body, err := env.DecodeBody()
 		if err != nil {
-			return LifecycleResult{}, err
+			return LifecycleResult{}, fmt.Errorf("decode receipt body: %w", err)
 		}
@@
 		body, err := env.DecodeBody()
 		if err != nil {
-			return LifecycleResult{}, err
+			return LifecycleResult{}, fmt.Errorf("decode trace body: %w", err)
 		}
```
</details>

Affects lines: 42-44, 57-58, 122-124, 151-153, 195-198, 205-208

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/lifecycle.go` around lines 42 - 44, Return values that
directly propagate errors (e.g., the ValidateSpace(i.Space) check) must be
wrapped with operation context using fmt.Errorf("...: %w", err); update every
direct "return err" in internal/network/lifecycle.go (including the six
occurrences reported around lines 42-44, 57-58, 122-124, 151-153, 195-198,
205-208) to wrap the underlying error with a short descriptive message of the
failing operation (for example: fmt.Errorf("validating space %q: %w", i.Space,
err) or similar for each operation), so callers get explicit context while
preserving the original error with %w.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: Several lifecycle validation and decode paths return raw errors without operation context, which violates the project’s wrapped-error rule and makes failures harder to diagnose.
- Fix plan: Wrap each propagated error with concise lifecycle context while preserving `errors.Is` behavior through `%w`.
