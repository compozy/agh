# Regression Suite: Execution Sandbox Abstraction

**Version:** 1.0
**Date:** 2026-04-16
**Feature:** Execution Sandbox Abstraction Layer
**Suite Type:** Full Regression

---

## Suite Classification

| Suite | Duration | Frequency | Coverage |
|-------|----------|-----------|----------|
| **Smoke** | 15 min | Every build | Critical paths only (SMOKE-001 to SMOKE-008) |
| **Targeted** | 30 min | Per task completion | Affected area + dependencies |
| **Full** | 60 min | Pre-merge / weekly | All test cases below |

---

## Execution Order

### 1. Smoke First (SMOKE-001 to SMOKE-008)

If any smoke test fails, **stop** and investigate before proceeding.

```
SMOKE-001: make verify passes (build + lint + test)
SMOKE-002: Local session create -> prompt -> stop lifecycle works
SMOKE-003: Config with sandbox profile loads without error
SMOKE-004: Workspace register with --sandbox flag persists
SMOKE-005: Session list shows environment backend column
SMOKE-006: Provider registry resolves local and daytona backends
SMOKE-007: Sandbox hooks fire during session lifecycle
SMOKE-008: Daemon boot completes with sandbox registry wired
```

### 2. P0 Critical Path (24 tests)

Must all pass before proceeding. Covers:
- ACP Launcher/ToolHost extraction regression (TC-REG-001 to TC-REG-004)
- Session sandbox lifecycle (TC-FUNC-008 to TC-FUNC-016)
- Security boundaries (TC-SEC-001 to TC-SEC-004)
- Core integration paths (TC-INT-001 to TC-INT-004)

### 3. P1 High Priority (16 tests)

Covers:
- Config validation and merge (TC-FUNC-001 to TC-FUNC-007)
- SSH transport and token management (TC-FUNC-017 to TC-FUNC-019)
- Daemon reconciliation (TC-FUNC-020 to TC-FUNC-024)
- Hook dispatch correctness (TC-FUNC-025 to TC-FUNC-030)
- Performance baselines (TC-PERF-001)

### 4. P2 Standard (11 tests)

Covers:
- Edge cases in config validation
- CLI flag behavior
- API contract serialization
- Performance measurements
- Host API methods

### 5. Exploratory

- Unscripted testing around concurrent session creation with different backends
- Edge cases in workspace resolution cascade
- Malformed TOML config handling

---

## Pass/Fail Criteria

### PASS
- All SMOKE tests pass
- All P0 tests pass
- 90%+ P1 tests pass
- No Critical or High severity bugs open
- `make verify` passes clean

### FAIL (Block Merge)
- Any SMOKE test fails
- Any P0 test fails
- Critical bug discovered (data loss, security vulnerability, ACP regression)
- `make verify` fails

### CONDITIONAL PASS
- P1 failures with documented workarounds and fix plan
- P2 failures documented as known issues
- Daytona E2E skipped due to missing API key (acceptable for CI)

---

## Test Case Priority Map

### P0 — Must Run Always (24 tests)

| ID | Title | Area |
|----|-------|------|
| TC-REG-001 | ACP session lifecycle unchanged after extraction | ACP |
| TC-REG-002 | Local file read/write through ToolHost matches direct OS | ACP |
| TC-REG-003 | Terminal create/kill through ToolHost works | ACP |
| TC-REG-004 | Permission enforcement unchanged after extraction | ACP |
| TC-FUNC-008 | Session start allocates SandboxID | Session |
| TC-FUNC-009 | Session start persists meta in creating state | Session |
| TC-FUNC-010 | Session start calls Provider.Prepare with correct fields | Session |
| TC-FUNC-011 | Session start calls SyncToRuntime after Prepare | Session |
| TC-FUNC-012 | Session start uses RuntimeRootDir in StartOpts | Session |
| TC-FUNC-013 | Session stop calls SyncFromRuntime | Session |
| TC-FUNC-014 | Session stop calls Destroy when DestroyOnStop | Session |
| TC-FUNC-015 | Session resume restores sandbox metadata | Session |
| TC-FUNC-016 | Session crash calls SyncFromRuntime best-effort | Session |
| TC-SEC-001 | Tar extraction rejects path traversal | Security |
| TC-SEC-002 | Env var allowlist blocks DAYTONA_API_KEY | Security |
| TC-SEC-003 | Env var allowlist allows AGH_SESSION_ID | Security |
| TC-SEC-004 | Tar extraction rejects symlink escapes | Security |
| TC-INT-001 | Config -> workspace -> environment resolution round-trip | Integration |
| TC-INT-002 | Full local session lifecycle through daemon | Integration |
| TC-INT-003 | Session resume with local provider | Integration |
| TC-INT-004 | Session list API includes sandbox field | Integration |
| TC-REG-005 | Existing ACP client_integration_test passes | ACP |
| TC-REG-006 | Existing session manager tests pass | Session |
| TC-FUNC-026 | sandbox.prepare deny aborts session creation | Hooks |

### P1 — Should Run Weekly+ (16 tests)

| ID | Title | Area |
|----|-------|------|
| TC-FUNC-001 | Valid sandbox profile parses from TOML | Config |
| TC-FUNC-002 | Snapshot wins over Image in DaytonaProfile | Config |
| TC-FUNC-003 | Invalid backend returns validation error | Config |
| TC-FUNC-004 | Invalid sync_mode returns validation error | Config |
| TC-FUNC-005 | Sandbox overlay merge preserves fields | Config |
| TC-FUNC-006 | Defaults.Sandbox cascade resolves profile | Config |
| TC-FUNC-007 | Missing SandboxRef resolves to local | Config |
| TC-FUNC-017 | SSH token refresh at 50% expiry | Daytona |
| TC-FUNC-018 | SSH auth failure triggers refresh and retry | Daytona |
| TC-FUNC-019 | SSH keepalive 30s interval | Daytona |
| TC-FUNC-020 | Reconciliation no-op with no remote sessions | Daemon |
| TC-FUNC-021 | Reconciliation reattaches recoverable session | Daemon |
| TC-FUNC-022 | Reconciliation destroys unrecoverable session | Daemon |
| TC-FUNC-023 | Reconciliation finds sandbox by agh_sandbox_id | Daemon |
| TC-FUNC-024 | Reconciliation skips local backend sessions | Daemon |
| TC-PERF-001 | Tar sync duration under threshold | Performance |

### P2 — Run at Release (11 tests)

| ID | Title | Area |
|----|-------|------|
| TC-FUNC-025 | SandboxProfile.Env map parses key-value pairs | Config |
| TC-FUNC-027 | sandbox.sync.before deny skips sync | Hooks |
| TC-FUNC-028 | sandbox.stop deny prevents destroy | Hooks |
| TC-FUNC-029 | Host API sandbox/list returns instances | Hooks |
| TC-FUNC-030 | Host API sandbox/exec requires capability | Hooks |
| TC-INT-005 | Concurrent sessions same workspace no corruption | Integration |
| TC-INT-006 | Daytona provider E2E lifecycle | Integration |
| TC-INT-007 | Workspace CRUD exposes sandbox_ref via API | Integration |
| TC-INT-008 | CLI workspace add --sandbox flag | Integration |
| TC-PERF-002 | Session start latency with local provider | Performance |
| TC-PERF-003 | Sandbox reconciliation boot time | Performance |
| TC-SEC-005 | Network policy unsupported setting logs warning | Security |

---

## Maintenance Notes

- Review and update this suite after each task completion
- Add regression cases for any bugs found during execution
- Remove or update tests when interfaces change
- Daytona E2E tests (TC-INT-006) require `DAYTONA_API_KEY` — skip in CI without key
