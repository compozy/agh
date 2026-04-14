---
status: resolved
file: internal/registry/multi.go
line: 124
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WU5,comment:PRRC_kwDOR5y4QM63maeM
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Honor cancellation after the fan-out completes.**

Both paths only validate `ctx` before launching goroutines. If cancellation happens while one source already returned a result, `Search()`/`resolveSource()` can still return partial data and let the caller continue with install/update work after the request was canceled.


<details>
<summary>🛠️ Suggested fix</summary>

```diff
 	wg.Wait()
+
+	if err := ctx.Err(); err != nil {
+		return nil, fmt.Errorf("registry: search canceled: %w", err)
+	}
 
 	if searchableSources == 0 {
 		return []Listing{}, nil
 	}
@@
 	wg.Wait()
+
+	if err := ctx.Err(); err != nil {
+		return nil, nil, fmt.Errorf("registry: info canceled for %q: %w", trimmedSlug, err)
+	}
 
 	for index := len(m.sources) - 1; index >= 0; index-- {
 		if results[index].detail != nil {
 			return m.sources[index], results[index].detail, nil
```
</details>

As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via context.Context cancellation".


Also applies to: 205-258

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/multi.go` around lines 93 - 124, After the fan-out
completes (after wg.Wait()), check the incoming context (ctx) for cancellation
before processing results/returning merged listings: if ctx.Err() != nil return
an empty []Listing and ctx.Err() (or wrap ctx.Err() with any collected errs), so
that Search()/resolveSource() honors cancellations and does not return partial
data; update the logic in the function containing wg.Wait(), merged :=
mergeListings(results), successes/errs handling to early-return on ctx
cancellation (and apply same pattern to the other similar block around lines
205-258).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `Search()` and `resolveSource()` only validate `ctx` before the fan-out. If cancellation happens while the goroutines are running, both paths can still return partial results after `wg.Wait()` completes. I will re-check `ctx.Err()` after the fan-out and add cancellation regression coverage in the in-scope `internal/registry/multi_test.go`.
- Resolution: `internal/registry/multi.go` now re-checks `ctx.Err()` after the search/info fan-out completes, and `internal/registry/multi_test.go` now covers cancellation-after-partial-results for both paths.
- Verification: `go test ./internal/registry/...`; `make verify`
