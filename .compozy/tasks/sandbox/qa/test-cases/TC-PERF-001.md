## TC-PERF-001: Tar sync duration under threshold

**Priority:** P1 (High)
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 3 minutes
**Created:** 2026-04-16
**Task:** 06

---

### Objective

Verify that workspace tar sync (both directions) completes within acceptable time thresholds for typical workspace sizes.

---

### Preconditions

- [x] Daytona sandbox available (or benchmarked locally)
- [x] Sync metrics logged (duration_ms, file_count, bytes_transferred)

---

### Performance Criteria

| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Sync 10 files / 100KB | < 2s | < 5s | | [ ] |
| Sync 100 files / 1MB | < 5s | < 15s | | [ ] |
| Sync 1000 files / 10MB | < 30s | < 60s | | [ ] |
| Tar creation overhead | < 100ms for 100 files | < 500ms | | [ ] |

---

### Test Steps

1. **Create workspace with 100 files, 1MB total**
   - **Expected:** Tar creation < 500ms

2. **Sync to runtime**
   - **Expected:** Total duration (tar + transfer + extract) logged, within acceptable threshold

3. **Sync from runtime**
   - **Expected:** Total duration within acceptable threshold

4. **Verify exclude patterns reduce transfer size**
   - Input: Workspace with `node_modules/` (large), excludes configured
   - **Expected:** Excluded directories not transferred, significant size reduction
