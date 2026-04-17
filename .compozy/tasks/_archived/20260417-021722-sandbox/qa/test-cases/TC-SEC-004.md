## TC-SEC-004: Tar extraction rejects symlink escapes

**Priority:** P0 (Critical)
**Type:** Security
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16
**Task:** 06
**Risk Level:** High

---

### Objective

Verify that tar extraction detects and rejects symlink-based directory traversal attacks where a symlink points outside the extraction destination.

---

### Test Steps

1. **Archive with symlink pointing outside destination**
   - Input: Tar containing `link -> /tmp/outside` and then file `link/secret`
   - **Expected:** File `link/secret` rejected because resolved path escapes destination

2. **Archive with nested symlink chain**
   - Input: Chain of symlinks that ultimately resolve outside destination
   - **Expected:** Rejected at resolution step

3. **Archive with safe symlink (within destination)**
   - Input: Symlink `subdir/link -> ../other_subdir` (stays within destination)
   - **Expected:** Accepted, file extracted correctly
