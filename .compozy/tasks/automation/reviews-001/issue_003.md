---
status: resolved
file: internal/api/core/errors.go
line: 94
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0I,comment:PRRC_kwDOR5y4QM623e7T
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In current Go versions, what is the recommended way to preserve both a sentinel error and the original cause in the error chain: multiple `%w` with `fmt.Errorf` or `errors.Join`?`

💡 Result:

In current Go versions (1.20+), the recommended way to preserve both a sentinel error and the original cause in the error chain is multiple %w with fmt.Errorf, such as fmt.Errorf("context: %w: %w", sentinel, cause). This creates a wrapped error tree where errors.Is and errors.As can find both the sentinel and the cause. errors.Join is designed for combining multiple independent errors (e.g., validation failures), not for adding a sentinel to a single cause. While errors.Join(sentinel, cause) technically preserves both via Unwrap []error and allows Is/As inspection, it is not the standard pattern for this use case. Official documentation emphasizes fmt.Errorf %w for wrapping with context while preserving the original error.

Citations:

- 1: https://pkg.go.dev/errors@go1.26rc3
- 2: https://pkg.go.dev/errors
- 3: https://go.dev/src/errors/join.go
- 4: https://go.dev/blog/go1.13-errors
- 5: https://pkg.go.dev/errors@master

---

🏁 Script executed:

```shell
# Check the Go version being used
cat go.mod | head -5
```

Repository: compozy/agh

Length of output: 109

---

🏁 Script executed:

```shell
# Read the specific file to verify the code snippet
cat -n internal/api/core/errors.go | sed -n '85,100p'
```

Repository: compozy/agh

Length of output: 598

---

🏁 Script executed:

```shell
# Search for usage of NewAutomationValidationError to understand the impact
rg 'NewAutomationValidationError' --type go -A 2 -B 2
```

Repository: compozy/agh

Length of output: 9531

---



</details>

**Use `fmt.Errorf("%w: %w", ErrAutomationValidation, err)` to preserve the original error in the error chain.**

The current code uses `%v` for the inner error, which converts it to a string representation instead of wrapping it in the error chain. This breaks downstream `errors.Is()` and `errors.As()` inspection. Since the project targets Go 1.25.0, use multiple `%w` directives to preserve both the sentinel and the cause: `fmt.Errorf("%w: %w", ErrAutomationValidation, err)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/errors.go` around lines 89 - 94, The
NewAutomationValidationError function wraps an incoming error but currently uses
"%v" which loses the original error in the chain; change the fmt.Errorf call in
NewAutomationValidationError to use two %w verbs so both ErrAutomationValidation
and the provided err are wrapped (use fmt.Errorf("%w: %w",
ErrAutomationValidation, err)) to preserve error unwrapping via
errors.Is/errors.As.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `NewAutomationValidationError` currently uses `%v` for the underlying cause, which keeps the sentinel but drops the original error from the unwrap chain. I will switch to wrapping both errors so `errors.Is`/`errors.As` can match the sentinel and the root cause, then add a focused regression test.
