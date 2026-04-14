---
status: resolved
file: internal/registry/extract_test.go
line: 177
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM567Hj4,comment:PRRC_kwDOR5y4QM63sxX3
---

# Issue 013: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Independent subtests should call `t.Parallel()` inside each `t.Run`.**

These subtests use isolated temp dirs and don’t share mutable state, so they qualify as independent.

<details>
<summary>Suggested pattern (example)</summary>

```diff
 t.Run("decompressed-size", func(t *testing.T) {
+	t.Parallel()
 	root := t.TempDir()
 	archive := mustTarGz(t, []tarEntry{
 		{name: "review/SKILL.md", content: "0123456789"},
 	})
 	// ...
 })
```
</details>



As per coding guidelines, `Use t.Parallel() for independent subtests in Go tests`.


Also applies to: 217-231, 279-366, 444-462

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract_test.go` around lines 62 - 177, The subtests in
internal/registry/extract_test.go (e.g., the t.Run closures named
"decompressed-size", "file-count", "symlink", "path-traversal",
"symlinked-parent", "empty-destination", "invalid-gzip" and the other t.Run
groups noted in the comment) are independent and should call t.Parallel() inside
each subtest body; update each t.Run(...) func(t *testing.T) { ... } to begin
with t.Parallel() so each subtest runs in parallel and keep the rest of the
assertions unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: independent subtests in `extract_test.go` omit `t.Parallel()` even though they use isolated temp dirs and do not share mutable state.
- Evidence: the referenced subtests in [`internal/registry/extract_test.go`](internal/registry/extract_test.go) perform independent filesystem setup per subtest and can run concurrently.
- Fix plan: add `t.Parallel()` at the start of each independent subtest while keeping the parent grouping intact.
- Resolution: Added `t.Parallel()` to the independent extractor subtests referenced in the review. Verified with package tests and `make verify`.
