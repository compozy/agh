## TC-PERF-001: Search and Reindex Stay Operational on a Realistic Memory Corpus

**Priority:** P2
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Memory Search/Reindex Performance
**Requirement:** REQ-MEM-010

---

### Objective

Verify that search and reindex remain operational and do not regress materially on a realistic mixed corpus size.

---

### Preconditions

- [ ] A temp corpus of at least 200 Markdown memories exists across global and workspace scopes.
- [ ] Real SQLite catalog storage is used.
- [ ] Benchmark or repeated-timing commands can run on a stable dev host.

---

### Test Steps

1. Seed a realistic corpus with overlapping terms and varied timestamps.
   - **Expected:** The corpus size is large enough to exercise ranking and indexing behavior.

2. Run `agh memory reindex` three to five times or run the targeted benchmark suite for `internal/memory`.
   - **Expected:** Each run completes successfully without hang, OOM, or pathological slowdown.

3. Run representative search queries for hot terms and narrow phrases.
   - **Expected:** Queries return correct top hits and remain responsive across repeated runs.

4. Compare the observed timings or benchmark numbers with the current-branch baseline on the same host.
   - **Expected:** No core candidate regresses by more than 25% without explanation, and no new hotspot becomes obviously pathological.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Many low-signal docs | noisy corpus | Search still ranks the strongest hit correctly |
| Repeated reindex | 5 consecutive runs | No cumulative failure or growing corruption |
| Mixed scopes | 100 global + 100 workspace | Scope filters still behave correctly under load |

---

### Related Test Cases

- `TC-FUNC-003`
- `TC-FUNC-004`

---

### Notes

This is a trend-detection case, not a hard release blocker by itself. Use the same host and command shape when comparing numbers.
