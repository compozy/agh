---
status: resolved
file: internal/registry/extract.go
line: 108
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564r-6,comment:PRRC_kwDOR5y4QM63phdR
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap entry-path validation failures with archive context.**

Returning `err` directly drops the extractor context here, and `ErrArchiveEntryPathRequired` becomes especially hard to trace because it carries no entry name.

<details>
<summary>Suggested fix</summary>

```diff
 		entryName, err := CleanArchiveEntryPath(header.Name)
 		if err != nil {
-			return err
+			return fmt.Errorf("clean archive entry %q: %w", header.Name, err)
 		}
```
</details>


As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		entryName, err := CleanArchiveEntryPath(header.Name)
		if err != nil {
			return fmt.Errorf("clean archive entry %q: %w", header.Name, err)
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract.go` around lines 106 - 108, Wrap validation errors
from CleanArchiveEntryPath with archive and entry context instead of returning
err directly: replace the bare return err after calling
CleanArchiveEntryPath(header.Name) with a wrapped error using fmt.Errorf that
includes the archive identifier (e.g. archivePath or archiveName), the original
header.Name (entry) and %w to wrap the original error (so
ErrArchiveEntryPathRequired remains inspectable); for example return
fmt.Errorf("archive %s: entry %q: %w", archivePath, header.Name, err). Ensure
fmt is imported if not already.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: extraction returns `CleanArchiveEntryPath` errors without identifying which archive entry triggered the failure.
- Evidence: [`internal/registry/extract.go`](internal/registry/extract.go) lines 106-108 return the raw error directly.
- Fix plan: wrap entry-path validation failures with the entry name so sentinel matching still works and callers get actionable context.
- Resolution: Wrapped archive entry path validation failures with entry context and kept sentinel matching intact. Verified with package tests and `make verify`.
