---
status: resolved
file: internal/dashboard/api.go
line: 263
severity: medium
author: claude-reviewer
---

# Issue 121: statusCodeForReadError relies on fragile error string matching



## Review Comment

`statusCodeForReadError` at line 263 determines the HTTP status code by inspecting error message strings:

```go
func statusCodeForReadError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "not running") {
		return http.StatusConflict
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}
```

This approach is brittle because:
1. Any refactoring of error messages in the backend will silently change the HTTP status codes returned to clients.
2. Unrelated errors that happen to contain "not found" or "not running" in their message will be misclassified.
3. The default fallback to `http.StatusBadRequest` (400) is incorrect for server-side errors -- an internal failure should return 500.
4. Line 271 redundantly calls `strings.ToLower(err.Error())` when `message` (already lowercased) is available from line 267.

**Suggested fix:** Define sentinel errors or error types in the backend package and use `errors.Is()` or `errors.As()` for classification. This aligns with the project's coding style guidelines ("Use `errors.Is()` and `errors.As()` for error matching; do not compare error strings").

```go
var (
    ErrNotFound   = errors.New("not found")
    ErrNotRunning = errors.New("not running")
)

func statusCodeForReadError(err error) int {
    if err == nil {
        return http.StatusOK
    }
    if errors.Is(err, ErrNotFound) {
        return http.StatusNotFound
    }
    if errors.Is(err, ErrNotRunning) {
        return http.StatusConflict
    }
    return http.StatusInternalServerError
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in `internal/dashboard/api.go`: `statusCodeForReadError` lowercases and inspects error strings for `"not running"` and `"not found"`, and it falls back to `400` for all other failures. The dashboard backend can provide typed/wrapped read errors, so this is an actionable correctness fix rather than a purely stylistic cleanup.
