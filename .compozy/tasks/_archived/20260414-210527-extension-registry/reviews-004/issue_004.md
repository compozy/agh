---
status: resolved
file: internal/cli/skill_marketplace.go
line: 188
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564868,comment:PRRC_kwDOR5y4QM63p4Aw
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle temp-dir cleanup failures instead of discarding them.**

This defer swallows `os.RemoveAll` errors, so failed cleanup leaves stale staging dirs behind and makes troubleshooting harder.


<details>
<summary>Suggested fix</summary>

```diff
 	defer func() {
-		_ = os.RemoveAll(tempRoot)
+		if removeErr := os.RemoveAll(tempRoot); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
+			err = errors.Join(err, fmt.Errorf("cli: remove temporary install directory %q: %w", tempRoot, removeErr))
+		}
 	}()
```
</details>

As per coding guidelines, `**/*.go`: `Never ignore errors with _ — every error must be handled or have a written justification`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_marketplace.go` around lines 186 - 188, The defer
currently swallows the os.RemoveAll(tempRoot) error; change the defer to capture
and handle the error instead of assigning to _. Replace the anonymous defer with
a check like: call os.RemoveAll(tempRoot) inside the defer, assign its return to
err, and if err != nil log or report it (use the package's logger or fmt.Errorf/
log.Printf consistently with the file) so failed cleanup is visible; reference
tempRoot, os.RemoveAll and the defer anonymous function when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the deferred `os.RemoveAll(tempRoot)` explicitly discards cleanup failures during marketplace skill installs.
- Evidence: [`internal/cli/skill_marketplace.go`](internal/cli/skill_marketplace.go) lines 186-188 assign the error to `_`, which violates the repo rule against ignoring errors.
- Fix plan: capture cleanup failures in the named return error so install callers see temp-dir cleanup problems.
- Resolution: Deferred cleanup now joins temp-dir removal failures into the returned error, with a regression test covering the joined error path. Verified with package tests and `make verify`.
