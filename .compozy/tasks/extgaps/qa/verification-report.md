VERIFICATION REPORT
-------------------
Claim: Current branch clears the required repository verification gates after fixing QA-blocking regressions in integration test fakes and managed local extension install handling.
Command: `make test-integration`
Executed: 2026-04-15T00:59:07Z
Exit code: 0
Output summary: 3674 integration-tagged tests passed in 23.386s; previously failing `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli`, and `internal/extension` integration surfaces are now green.
Warnings: none
Errors: none
Verdict: PASS

Claim: Current branch clears the required repository verification gate for format, lint, unit/integration-adjacent tests, web checks, and build.
Command: `make verify`
Executed: 2026-04-15T00:59:10Z
Exit code: 0
Output summary: frontend format/oxlint/tsgo/vitest/build passed; Go lint reported `0 issues`; package boundary check passed; repository test run finished with `DONE 3420 tests in 0.800s`.
Warnings: macOS linker deprecation warning emitted while building `golangci-lint` (`-bind_at_load`); non-blocking toolchain noise only.
Errors: none
Verdict: PASS

Claim: The extension bundles feature is release-ready based on live daemon, CLI, HTTP, UDS, and browser QA against a real installed reference bundle.
Command: `AGH_HOME=/tmp/codex-qa-extgaps/home go run ./cmd/agh daemon start --foreground` plus live CLI/HTTP/browser flows captured under `.compozy/tasks/extgaps/qa/evidence/` and `.compozy/tasks/extgaps/qa/screenshots/`
Executed: 2026-04-15T00:45:14Z through 2026-04-15T00:58:13Z
Exit code: 0
Output summary: real bundle install, preview, activation, binding update, deactivation, reactivation, UDS catalog parity, extension disable guard, automation job/trigger materialization, bridge materialization, and browser rendering after workspace onboarding were all exercised successfully.
Warnings: first-run UI requires workspace registration before route content renders in a clean `AGH_HOME`; browser evidence therefore includes onboarding plus a post-onboarding pass.
Errors: live API responses still reproduce bundle/resource mismatches tracked in `BUG-001` and `BUG-007`.
Verdict: FAIL

BROWSER EVIDENCE (when Web UI flows were tested)
-------------------------------------------------
Dev server: `make web-dev` confirmed at `http://localhost:3000`; Vite proxy path restored via temporary QA forwarders on `127.0.0.1:2123` and `::1:2123` to the isolated daemon at `127.0.0.1:22123`
Flows tested: 4
Flow details:
  - First-run onboarding: `http://localhost:3000/automation?qa_refresh=2` -> `http://localhost:3000/automation?qa_refresh=2` | Verdict: PASS
    Evidence: `.compozy/tasks/extgaps/qa/screenshots/ui-automation-fresh2.png`
  - Automation jobs after workspace registration: `http://localhost:3000/automation?qa_refresh=2` -> `http://localhost:3000/automation?qa_refresh=2` | Verdict: PASS
    Evidence: `.compozy/tasks/extgaps/qa/screenshots/ui-after-workspace.png`
  - Automation triggers after workspace registration: `http://localhost:3000/automation?qa_refresh=2` -> `http://localhost:3000/automation?qa_refresh=2` | Verdict: PASS
    Evidence: `.compozy/tasks/extgaps/qa/screenshots/ui-automation-triggers-final.png`
  - Bridges after workspace registration: `http://localhost:3000/bridges` -> `http://localhost:3000/bridges` | Verdict: PASS
    Evidence: `.compozy/tasks/extgaps/qa/screenshots/ui-bridges-after-workspace.png`
Viewports tested: default only
Authentication: not required
Blocked flows: none

TEST CASE COVERAGE (when qa-report artifacts exist)
----------------------------------------------------------
Test cases found: 30
Executed: 19
Results:
  - SMOKE-005: PASS | Bug: none
  - SMOKE-001: PASS | Bug: none
  - SMOKE-002: PASS | Bug: none
  - SMOKE-003: PASS | Bug: none
  - SMOKE-004: FAIL | Bug: BUG-001, BUG-007
  - TC-FUNC-001: PASS | Bug: none
  - TC-FUNC-003: PASS | Bug: none
  - TC-FUNC-004: PASS | Bug: none
  - TC-FUNC-005: PASS | Bug: none
  - TC-FUNC-006: PASS | Bug: none
  - TC-FUNC-009: PASS | Bug: none
  - TC-FUNC-011: PASS | Bug: none
  - TC-FUNC-015: PASS | Bug: none
  - TC-INT-001: PASS | Bug: none
  - TC-INT-002: PASS | Bug: none
  - TC-INT-003: PASS | Bug: none
  - TC-INT-004: PASS | Bug: none
  - TC-INT-005: PASS | Bug: none
  - TC-INT-006: PASS | Bug: none
Not executed: `TC-FUNC-002`, `TC-FUNC-007`, `TC-FUNC-008`, `TC-FUNC-010`, `TC-FUNC-012`, `TC-FUNC-013`, `TC-FUNC-014`, `TC-INT-010`, `TC-SEC-001`, `TC-SEC-002`, `TC-SEC-003`

ISSUES FILED
-------------
Total: 7
By severity:
  - Critical: 0
  - High: 3
  - Medium: 4
  - Low: 0
Details:
  - BUG-001: API bundlepkgStableID uses colon separator instead of SHA256 hash | Severity: Medium | Priority: P2 | Status: Open
  - BUG-002: Missing test coverage for bundle store persistence layer | Severity: High | Priority: P1 | Status: Open
  - BUG-003: Missing test coverage for bundle validation in extension/bundle.go | Severity: High | Priority: P1 | Status: Open
  - BUG-004: Zero handler tests for all 8 bundle API endpoints | Severity: High | Priority: P1 | Status: Open
  - BUG-005: Reconciliation race condition with concurrent Activate/Deactivate | Severity: Medium | Priority: P1 | Status: Open
  - BUG-006: defaultSessionChannel fallback logic may bypass bundle effective default | Severity: Medium | Priority: P1 | Status: Open
  - BUG-007: Bundle activation bridge payload omits owning extension_name | Severity: Medium | Priority: P2 | Status: Open
