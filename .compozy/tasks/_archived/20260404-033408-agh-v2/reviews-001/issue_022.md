---
status: resolved
file: internal/cli/root.go
line: 15
severity: low
author: claude-code
provider_ref:
---

# Issue 022: cli/ imports daemon/ directly, violating architecture rule

## Review Comment

`internal/cli/root.go` (line 15) and `internal/cli/daemon.go` (line 15) import `aghdaemon "github.com/pedronauck/agh/internal/daemon"` in production code. CLAUDE.md states: "No package imports `daemon/`, `httpapi/`, `udsapi/`, or `cli/`" and "cli/ talks to daemon via UDS."

The import is used for: `aghdaemon.New()` in foreground mode, and `aghdaemon.ReadInfo` / `aghdaemon.Info` for reading daemon info from disk. The `Boundaries()` check in `magefile.go` does not enforce this restriction for `cli/` -> `daemon/`.

**Suggested fix:** If the foreground daemon spawn from CLI is intentional (composition root behavior), document the exception in CLAUDE.md and add a boundary check exception. Otherwise, extract `daemon.Info` / `daemon.ReadInfo` to a shared package, and have foreground mode handled differently.

## Triage

- Decision: `invalid`
- Notes: The CLI importing `internal/daemon` is a composition choice for foreground daemon startup and daemon-info access, not a broken runtime behavior in the scoped code. The review points to a documentation or boundary-enforcement mismatch, but fixing that would require project-level rule or package-boundary changes outside this batch.
