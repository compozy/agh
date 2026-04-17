VERIFICATION REPORT
-------------------
Claim: Local regression gate for the sandbox task passes after QA fixes.
Command: `cd /tmp/agh-sandbox-qa.MWRGBE/ext-refac && make verify`
Executed: just now, after all code changes
Exit code: 0
Output summary: Web checks passed (`82` test files, `676` web tests), lint reported `0` issues, backend unit suite finished with `DONE 4327 tests in 25.726s`, and boundary checks reported `OK: all package boundaries respected`.
Warnings: one external macOS toolchain warning while building the golangci-lint helper binary: `ld: warning: -bind_at_load is deprecated on macOS`
Errors: none
Verdict: PASS

VERIFICATION REPORT
-------------------
Claim: Integration coverage for the sandbox task passes for all locally reachable paths.
Command: `cd /tmp/agh-sandbox-qa.MWRGBE/ext-refac && make test-integration`
Executed: just now, after all code changes
Exit code: 0
Output summary: Integration suite finished with `DONE 4664 tests, 2 skipped in 62.309s`.
Warnings: two Daytona integration tests were skipped because `DAYTONA_API_KEY` is not configured locally.
Errors: none
Verdict: PASS

BROWSER EVIDENCE (when Web UI flows were tested)
-------------------------------------------------
Dev server: isolated AGH daemon in `/tmp/agh-sandbox-home`, confirmed at `http://localhost:2124`
Flows tested: 5
Flow details:
  - shell load: `http://localhost:2124/` -> `http://localhost:2124/` | Verdict: PASS
    Evidence: `.compozy/tasks/sandbox/qa/screenshots/web-shell.png`
  - session route: `http://localhost:2124/` -> `http://localhost:2124/session/sess-c8d404ef179e3c5c` | Verdict: PASS
    Evidence: `.compozy/tasks/sandbox/qa/screenshots/web-session.png`
  - network disabled state: `http://localhost:2124/network` -> `http://localhost:2124/network` | Verdict: PASS
    Evidence: `.compozy/tasks/sandbox/qa/screenshots/web-network-disabled.png`
  - automation route: `http://localhost:2124/automation` -> `http://localhost:2124/automation` | Verdict: PASS
    Evidence: `.compozy/tasks/sandbox/qa/screenshots/web-automation.png`
  - back/forward routing: `http://localhost:2124/automation` -> `http://localhost:2124/network` -> `http://localhost:2124/automation` | Verdict: PASS
    Evidence: `.compozy/tasks/sandbox/qa/logs/web-back-forward.log`
Viewports tested: desktop only (`1280x800`)
Authentication: not required
Blocked flows: remote Daytona-backed environment flows remain blocked because `DAYTONA_API_KEY` is not available locally

TEST CASE COVERAGE (when qa-report artifacts exist)
----------------------------------------------------------
Test cases found: 60
Executed: 9
Results:
  - `SMOKE-001`: PASS | Bug: `BUG-001`, `BUG-002`, `BUG-003`
  - `SMOKE-002`: PASS | Bug: none
  - `SMOKE-004`: PASS | Bug: none
  - `SMOKE-005`: PASS | Bug: none
  - `SMOKE-008`: PASS | Bug: none
  - `TC-INT-002`: PASS | Bug: none
  - `TC-INT-004`: PASS | Bug: none
  - `TC-INT-008`: PASS | Bug: none
  - `TC-INT-006`: BLOCKED | Reason: `DAYTONA_API_KEY` is required for live Daytona lifecycle validation
Not executed: `SMOKE-003`, `SMOKE-006`, `SMOKE-007`; `TC-FUNC-001..TC-FUNC-030`; `TC-INT-001`, `TC-INT-003`, `TC-INT-005`, `TC-INT-007`; `TC-REG-001..TC-REG-006`; `TC-SEC-001..TC-SEC-005`; `TC-PERF-001..TC-PERF-003`. Many of these were covered indirectly by `make verify` or `make test-integration`, but they were not traced as separate manual case executions during this run.

ISSUES FILED
-------------
Total: 3
By severity:
  - Critical: 0
  - High: 1
  - Medium: 2
  - Low: 0
Details:
  - `BUG-001`: Async bridge readiness made delivery tests race the auth probe | Severity: Medium | Priority: P1 | Status: Fixed
  - `BUG-002`: Bridge stop paths accessed route state without synchronization | Severity: High | Priority: P1 | Status: Fixed
  - `BUG-003`: Teams delivery test swapped API factory before async init settled | Severity: Medium | Priority: P1 | Status: Fixed
