---
status: resolved
file: internal/cli/skill_marketplace.go
line: 94
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrWy,comment:PRRC_kwDOR5y4QM62twcA
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/cli/skill_marketplace.go | sed -n '80,100p'
```

Repository: compozy/agh

Length of output: 863

---

🏁 Script executed:

```shell
cat -n internal/cli/skill_marketplace.go | sed -n '370,380p'
```

Repository: compozy/agh

Length of output: 338

---

🏁 Script executed:

```shell
cat -n internal/cli/skill_marketplace.go | sed -n '408,420p'
```

Repository: compozy/agh

Length of output: 689

---



</details>

**Don't discard cleanup errors with `_`.**

These three locations silently drop error returns from cleanup operations:
- Lines 85, 376: `archive.Data.Close()` and `gzipReader.Close()` in defer blocks
- Line 413: `file.Close()` in error path (while line 416 properly handles `file.Close()` in success path)

Per coding guidelines, every error must be handled or have a written justification. Suppressing these errors can hide resource leaks or incomplete operations in production.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/skill_marketplace.go` around lines 84 - 94, The defer and
error-path currently discard cleanup errors for archive.Data.Close(),
gzipReader.Close(), and file.Close(); change each ignore (“_ = ...Close()”) to
capture the error and handle it (e.g., if closeErr := archive.Data.Close();
closeErr != nil { return or wrap/log the error } or log it with context) so
cleanup failures are not silently dropped; update the defer for tempRoot removal
similarly if needed, and ensure the error-path after creating/writing the file
returns or logs the file.Close() error instead of discarding it, referencing
archive.Data.Close, gzipReader.Close, and file.Close in skill_marketplace.go.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `archive.Data.Close()`, `gzipReader.Close()`, and the error-path `file.Close()` are currently discarded. Cleanup failures on installation I/O should not disappear silently because they can hide truncated downloads or incomplete writes.
- Fix approach: Convert the install/extract functions to report close failures with contextual errors, using joined errors where the primary operation already failed.
