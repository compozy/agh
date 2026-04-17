## TC-REG-003: Terminal create/kill through ToolHost works

**Priority:** P0 (Critical)
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 02

---

### Objective

Verify that terminal operations (create, kill, output, wait, release) through `localToolHost` work identically to the previous direct handler implementation.

---

### Context

Recent changes: Terminal handlers extracted from `acp/handlers.go:323-405` into `localToolHost`.

---

### Critical Path Tests

1. [x] CreateTerminal spawns process with correct cwd
2. [x] KillTerminal sends signal and terminates process
3. [x] TerminalOutput returns captured output
4. [x] WaitForTerminalExit returns correct exit code
5. [x] ReleaseTerminal cleans up resources
