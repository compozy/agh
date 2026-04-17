## TC-REG-002: Local file read/write through ToolHost matches direct OS

**Priority:** P0 (Critical)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 02

---

### Objective

Verify that `localToolHost.ReadTextFile` and `WriteTextFile` produce identical results to the previous direct `os.ReadFile`/`os.WriteFile` calls in ACP handlers.

---

### Context

Recent changes: File IO handlers now route through `ToolHost` interface instead of direct `os.*` calls.

---

### Critical Path Tests

1. [x] ReadTextFile reads existing file with correct content
2. [x] ReadTextFile with non-existent file returns appropriate error
3. [x] WriteTextFile creates file with correct content and 0644 permissions
4. [x] WriteTextFile creates parent directories as needed
5. [x] ResolvePath resolves relative paths against workspace root
6. [x] ResolvePath rejects paths that escape workspace root (path traversal)
