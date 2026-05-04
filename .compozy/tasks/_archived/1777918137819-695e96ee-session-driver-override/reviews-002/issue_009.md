---
status: resolved
file: internal/session/resume_repair_test.go
line: 271
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPL,comment:PRRC_kwDOR5y4QM6628Dx
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify current nil-context error contract style and assertion style.
rg -n --type=go 'resume repair context is required|legacy provider repair context is required'
rg -n --type=go 'strings\.Contains\(err\.Error\(\)' internal/session/resume_repair_test.go
rg -n --type=go 'errors\.Is\(|errors\.As\(' internal/session/resume_repair_test.go internal/session/resume_repair.go
```

Repository: compozy/agh

Length of output: 860

---



</details>

**Consolidate nil-context tests into table-driven subtests and use sentinel errors for matching.**

These tests violate two coding guidelines:

1. **Error matching**: Must use `errors.Is()` instead of `strings.Contains(err.Error(), ...)`. Requires defining sentinel errors in production code (`internal/session/resume_repair.go`) instead of `errors.New()`.
2. **Test structure**: Must use `t.Run("Should...")` subtests, not top-level test functions.

Consolidate both nil-context scenarios into a single table-driven test with subtests:
```go
func TestRepairRejectsNilContext(t *testing.T) {
  t.Parallel()
  
  tests := []struct {
    name string
    fn   func(context.Context) error
    err  error // sentinel error to match with errors.Is()
  }{
    {
      name: "Should reject nil context in repairInactiveMeta",
      fn:   func(ctx context.Context) error {
        _, err := h.manager.repairInactiveMeta(ctx, filepath.Join(t.TempDir(), "meta.json"), meta)
        return err
      },
      err: ErrRepairContextRequired, // define in production
    },
    // ... similar for RepairLegacyProvider
  }
  
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      err := tt.fn(nil)
      if !errors.Is(err, tt.err) {
        t.Errorf("got %v, want %v", err, tt.err)
      }
    })
  }
}
```

This requires defining sentinel errors in `resume_repair.go`:
```go
var ErrRepairContextRequired = errors.New("resume repair context is required")
var ErrLegacyRepairContextRequired = errors.New("legacy provider repair context is required")
```

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/resume_repair_test.go` around lines 238 - 271, Combine the
two nil-context tests into a single table-driven test (e.g.,
TestRepairRejectsNilContext) that uses t.Run subtests and invokes the two call
sites (manager.repairInactiveMeta and RepairLegacyProvider) via small wrapper
functions, and assert errors using errors.Is rather than string matching; to
support that, add sentinel error variables in production (resume_repair.go)
named ErrRepairContextRequired and ErrLegacyRepairContextRequired and have
repairInactiveMeta and RepairLegacyProvider return those sentinel errors when
ctx is nil so the test can match with errors.Is(err, ErrRepairContextRequired) /
ErrLegacyRepairContextRequired.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: The nil-context tests currently match error text with `strings.Contains`, which is weaker than the repository error-matching contract. I will add same-package sentinel errors in `internal/session/resume_repair.go` and convert the in-scope tests to table-driven `errors.Is` subtests; the production file change is the minimum out-of-scope change required to make the assertions precise.
