## TC-SEC-001: Tar extraction rejects path traversal

**Priority:** P0 (Critical)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 06
**Risk Level:** High

---

### Objective

Verify that tar extraction rejects archives containing absolute paths, `..` traversal components, and paths that escape the destination directory after symlink evaluation.

---

### Preconditions

- [x] `extractTar()` function available in `internal/sandbox/daytona/tar.go`

---

### Test Steps

1. **Archive with absolute path**
   - Input: Tar entry with name `/etc/passwd`
   - **Expected:** Entry rejected, error logged, extraction continues for safe entries

2. **Archive with `..` traversal**
   - Input: Tar entry with name `../../etc/shadow`
   - **Expected:** Entry rejected, error logged

3. **Archive with symlink escape**
   - Input: Tar with symlink `link -> /tmp` followed by entry `link/evil`
   - **Expected:** Entry rejected after symlink resolution shows escape

4. **Archive with unsupported file modes**
   - Input: Tar entry with socket or device file type
   - **Expected:** Entry skipped with warning log, not silently misapplied

5. **Normal archive extracts correctly**
   - Input: Tar with safe relative paths
   - **Expected:** All files extracted to correct destination

---

### Attack Vectors

- [x] Path traversal via `..` components
- [x] Absolute path injection
- [x] Symlink-based escape
- [x] Device file injection
