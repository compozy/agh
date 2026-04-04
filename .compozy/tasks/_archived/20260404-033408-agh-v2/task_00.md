---
status: completed
domain: Infrastructure
type: Configuration
scope: Full
complexity: low
dependencies: []
---

# Task 00: Project Scaffold — go.mod Cleanup and Magefile Update

## Overview

Clean up the project foundation before implementation begins. The current `go.mod` contains ~40 dependencies from the old project (NATS, suture, gobreaker, PTY, WebSocket, QUIC, MongoDB, TPM, etc.) that must be removed. The `magefile.go` references a `web/` directory that no longer exists. Without this cleanup, `make verify` will fail on the first task.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST remove all deprecated dependencies from go.mod (NATS, suture, gobreaker, creack/pty, nhooyr/websocket, quic-go, mongo-driver, go-tpm, antithesis-sdk, oklog/run, charmbracelet/*)
- MUST add `github.com/coder/acp-go-sdk` to go.mod
- MUST ensure direct dependencies are correct: BurntSushi/toml, spf13/cobra, gin-gonic/gin, rs/xid, gofrs/flock, joho/godotenv, modernc.org/sqlite, toon-format/toon-go
- MUST remove WebBuild, WebCheck, WebTest targets from magefile.go
- MUST remove `node_modules/` from project root
- MUST update Verify() to only run Fmt, Lint, Test, Build (no web checks)
- MUST add a `Boundaries()` mage target that greps for forbidden package imports
- MUST create stub `cmd/agh/main.go` and `internal/version/version.go` so `make build` succeeds
- MUST ensure `make verify` passes after cleanup
</requirements>

## Subtasks
- [ ] 0.1 Remove all deprecated dependencies from go.mod via `go get` (remove nats, suture, gobreaker, pty, websocket, quic, mongo, tpm, antithesis, oklog, charmbracelet)
- [ ] 0.2 Add new dependencies: `go get github.com/coder/acp-go-sdk`
- [ ] 0.3 Promote indirect deps to direct: cobra, gin, flock, xid
- [ ] 0.4 Run `go mod tidy` to clean up go.sum
- [ ] 0.5 Remove WebBuild/WebCheck/WebTest from magefile.go, update Verify()
- [ ] 0.6 Remove `node_modules/` directory from project root
- [ ] 0.7 Add `Boundaries()` mage target: grep for forbidden imports (no package imports daemon/, httpapi/, udsapi/, cli/)
- [ ] 0.8 Create minimal `cmd/agh/main.go` stub (just `package main` + `func main()`)
- [ ] 0.9 Create minimal `internal/version/version.go` stub
- [ ] 0.10 Verify `make verify` passes

## Implementation Details

### Files to modify
- `go.mod` — Dependency cleanup
- `go.sum` — Regenerated via `go mod tidy`
- `magefile.go` — Remove web targets, add Boundaries()
- `Makefile` — Verify still calls correct mage targets

### Files to create
- `cmd/agh/main.go` — Minimal stub
- `internal/version/version.go` — Build metadata stub

### Files to delete
- `node_modules/` — No longer needed (old web/ frontend)

### Related ADRs
- [ADR-001: Rewrite From Scratch](../adrs/adr-001.md) — Clean slate
- [ADR-002: Pragmatic Flat Architecture](../adrs/adr-002.md) — CI boundary checks

## Deliverables
- Clean `go.mod` with only v2 dependencies
- Updated `magefile.go` without web targets + with Boundaries()
- Minimal stubs so `make verify` passes
- No `node_modules/` at project root

## Tests
- Unit tests:
  - [ ] `make verify` passes (fmt + lint + test + build)
  - [ ] `mage boundaries` reports no violations on clean project
  - [ ] `go mod verify` passes
- Test coverage target: N/A (scaffold task)

## Success Criteria
- `make verify` passes with zero errors
- `go.mod` contains only v2 dependencies
- No web-related targets in magefile
- `mage boundaries` target exists and passes
