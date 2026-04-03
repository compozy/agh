---
status: resolved
file: internal/cli/daemon.go
line: 1312
severity: medium
author: claude-reviewer
---

# Issue 109: userCommandError uses string prefix check instead of error wrapping



## Review Comment

The `userCommandError` function at line 1312-1319 uses string prefix checking to avoid double-wrapping errors:

```go
func userCommandError(err error) error {
    if err == nil {
        return nil
    }
    if strings.HasPrefix(err.Error(), "error: ") {
        return err
    }
    return fmt.Errorf("error: %s", err.Error())
}
```

This approach has two problems:

1. **It breaks the error chain**: Using `fmt.Errorf("error: %s", err.Error())` instead of `fmt.Errorf("error: %w", err)` loses the original error's type information. Callers cannot use `errors.Is()` or `errors.As()` to match the original error. The project's CLAUDE.md coding style explicitly requires: "Use `errors.Is()` and `errors.As()` for error matching; do not compare error strings."

2. **String prefix checking is fragile**: If an upstream error message happens to start with "error: ", it will be passed through without the `userCommandError` prefix, which may or may not be the right behavior.

Suggested fix: Use `%w` for wrapping and introduce a sentinel type to detect already-wrapped errors:

```go
type userError struct{ inner error }
func (e *userError) Error() string { return "error: " + e.inner.Error() }
func (e *userError) Unwrap() error { return e.inner }

func userCommandError(err error) error {
    if err == nil {
        return nil
    }
    var ue *userError
    if errors.As(err, &ue) {
        return err
    }
    return &userError{inner: err}
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in `internal/cli/daemon.go`: `userCommandError` currently re-creates the error text with `fmt.Errorf("error: %s", err.Error())`, which strips the original error chain, and it suppresses re-wrapping via a string-prefix check. This is directly contrary to the repository guidance on `errors.Is`/`errors.As` and is straightforward to fix with a small wrapper type that preserves `Unwrap`.
