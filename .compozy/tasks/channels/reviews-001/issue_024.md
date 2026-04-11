---
status: resolved
file: internal/daemon/daemon_test.go
line: 3215
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLs,comment:PRRC_kwDOR5y4QM623eJF
---

# Issue 024: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Propagate the marker file close error.**

`appendMarkerLine` can currently report success even if `Close()` fails while flushing the appended line. Please return the close error instead of dropping it with `_`.

<details>
<summary>Possible fix</summary>

```diff
-func appendMarkerLine(path string, line string) error {
+func appendMarkerLine(path string, line string) (err error) {
 	target := strings.TrimSpace(path)
 	if target == "" {
 		return nil
 	}
 	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
 		return err
 	}
 	file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
 	if err != nil {
 		return err
 	}
 	defer func() {
-		_ = file.Close()
+		if closeErr := file.Close(); err == nil && closeErr != nil {
+			err = closeErr
+		}
 	}()
 	_, err = fmt.Fprintf(file, "%s\n", strings.TrimSpace(line))
 	return err
}
```
</details>

As per coding guidelines, "`**/*.go`: Never ignore errors with `_` — every error must be handled or have a written justification`".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_test.go` around lines 3207 - 3215, The
appendMarkerLine function currently ignores file.Close() error; change it to
capture the close error and return it (or wrap it) if non-nil so failures during
flush are propagated. Specifically, after opening the file via os.OpenFile and
writing with fmt.Fprintf, store the result of file.Close() (instead of
discarding with `_`), check that error and return it (or if fmt.Fprintf already
returned a non-nil error, return that first or combine/wrap both) so the
function does not report success when Close() fails.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: `appendMarkerLine` returns the write error from `fmt.Fprintf`, but it currently discards `file.Close()` errors. A buffered write/flush failure on close could therefore be reported as success.
- Root cause: The deferred close path ignores its error with `_ = file.Close()`.
- Fix plan: Convert the helper to a named return and propagate a close error when the write itself succeeded, preserving write-error precedence.
- Resolution: `appendMarkerLine` now propagates close errors correctly and the updated helper passed targeted tests and `make verify`.
