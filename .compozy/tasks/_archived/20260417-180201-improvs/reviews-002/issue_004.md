---
status: resolved
file: internal/sandbox/daytona/perf_bench_test.go
line: 30
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57zno0,comment:PRRC_kwDOR5y4QM645ile
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n internal/sandbox/daytona/perf_bench_test.go
```

Repository: compozy/agh

Length of output: 5911

---

🏁 Script executed:

```shell
rg -n "panic|log\.Fatal|log\.Printf|fmt\.Println" internal/sandbox/daytona/perf_bench_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n "context\.Background()" internal/sandbox/daytona/perf_bench_test.go
```

Repository: compozy/agh

Length of output: 156

---

🏁 Script executed:

```shell
rg -n "_\s*=" internal/sandbox/daytona/perf_bench_test.go
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Fix bytes-per-iteration accounting in this benchmark.**

Line 20 sets `SetBytes` to 2 MiB (source data size), but lines 28–30 assert only 1 MiB (the `limit`) is copied per iteration. Reported throughput metrics will be inflated by 2×.

Change `b.SetBytes(int64(len(sourceData)))` to `b.SetBytes(int64(limit))` to report actual bytes processed.

<details>
<summary>Proposed patch</summary>

```diff
-	b.SetBytes(int64(len(sourceData)))
+	b.SetBytes(int64(limit))
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	b.SetBytes(int64(limit))

	for b.Loop() {
		var dst bytes.Buffer
		session := newBenchmarkSession(sourceData)
		if err := ioCopyLimit(&dst, session, limit, &mu); err != nil {
			b.Fatalf("ioCopyLimit() error = %v", err)
		}
		if got := dst.Len(); got != limit {
			b.Fatalf("ioCopyLimit() len = %d, want %d", got, limit)
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/sandbox/daytona/perf_bench_test.go` around lines 20 - 30, The
benchmark currently calls b.SetBytes(int64(len(sourceData))) which overstates
per-iteration bytes because ioCopyLimit(&dst, session, limit, &mu) only copies
up to limit bytes; change the call to b.SetBytes(int64(limit)) so the reported
throughput matches the actual bytes processed per iteration (locate the
b.SetBytes call near the benchmark loop and update it to use the limit variable
used by ioCopyLimit and the dst length checks).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the benchmark reports `len(sourceData)` as bytes processed even though `ioCopyLimit` caps each iteration at `limit`, which inflates throughput metrics by 2x.
- Fix plan: set benchmark bytes to `limit` so the reported throughput matches the actual copied payload per iteration.
- Resolution: `BenchmarkIOCopyLimitSlidingWindow` now reports `limit` as the processed byte count, so throughput reflects the copied payload instead of the larger backing buffer.
- Verification: `go test ./internal/bundles ./internal/sandbox/daytona ./internal/extension ./internal/tools` and `make verify` passed on 2026-04-17.
