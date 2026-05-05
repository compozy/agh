# Network Threads QA Execution Verification Report

VERIFICATION REPORT
-------------------
Claim: Task 19 QA execution for `.compozy/tasks/network-threads` completed with root-cause fixes and fresh full-repo verification.
Command: `make verify 2>&1 | tee .compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/final-make-verify.log`
Executed: 2026-05-05T17:50:01Z, after runtime and Web fixes.
Exit code: 0
Output summary: Bun lint found `0 warnings and 0 errors`; Bun/Vitest reported `355 passed` files and `2223 passed` tests; Web production build completed; Go lint reported `0 issues`; Go tests reported `DONE 8401 tests`; package boundaries reported `OK: all package boundaries respected`.
Warnings: Vite emitted the existing chunk-size advisory for chunks above 500 kB. macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS`. E2E Web emitted Node's `NO_COLOR`/`FORCE_COLOR` warning.
Errors: none.
Verdict: PASS.

BEHAVIORAL EVIDENCE
-------------------
Operator journey: An operator created two isolated local ACP mock sessions, used them as AGH Network peers in channel `builders`, created a public thread, resolved a direct room, sent direct `say`, `receipt`, and `trace` messages, attached direct work state, rejected invalid legacy/cross-container commands, inspected the same state through CLI, HTTP API, and Web UI, then reran post-gate checks after `make verify`.

Business outcome: PASS. Public thread messages remained visible only on the thread surface, direct-room messages stayed scoped to their `direct_id`, direct work remained bound to the direct surface, legacy `interaction_id` input stayed rejected, and Web routes for missing conversations now show unavailable states without reply/direct composers.

Live provider/LLM: BLOCKED as live LLM proof. The active provider catalog contained native CLI providers with operator-home policy, but this deterministic local QA used `acpmock` fixture-backed sessions to exercise reachable CLI/API/Web/runtime boundaries. No live LLM transcript is claimed.

Agent behavior:

- `ops-coordinator.sess-f225672703443d05`: created public thread `thread_builders_qa2`, sent public thread messages `msg_say_qa2` and `msg_summary_qa2`, and initiated direct coordination.
- `patch-worker.sess-8658903011bda794`: participated through direct room `direct_8eeac8fd8fc697295a67c6206051e959`.
- Direct trace/work behavior: direct message `msg_trace_qa2` recorded `Trace update recorded.` and `work_patch_qa2` remained `state=working`, `surface=direct`.

Artifacts produced and used:

- `.compozy/tasks/network-threads/qa/bootstrap-manifest.json`: fresh isolated QA lab contract.
- `.compozy/tasks/network-threads/qa/bootstrap.env`: isolated daemon/Web/provider environment used for scenario commands only.
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/cli-api-scenario.zsh`: executable CLI/API operator scenario.
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/scenario.env`: durable IDs for post-gate rechecks.
- `.compozy/tasks/network-threads/qa/bug-reports/BUG-001-session-event-query-finalization-race.md`: runtime E2E defect, fixed.
- `.compozy/tasks/network-threads/qa/bug-reports/BUG-002-web-network-missing-conversation-state.md`: Web missing-conversation state defect, fixed.
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/legacy-scan-classification.md`: legacy surface scan classification.

Cross-surface truth checks:

- CLI thread messages: `post-final-thread-messages.stdout` contains 2 public thread messages for `thread_builders_qa2`.
- API thread messages: `post-final-api-thread-messages.stdout` contains the same 2 thread messages.
- CLI direct messages: `post-final-direct-messages.stdout` contains 3 direct messages for `direct_8eeac8fd8fc697295a67c6206051e959`.
- API direct messages: `post-final-api-direct-messages.stdout` contains the same 3 direct messages.
- CLI work lookup: `post-final-work-lookup.stdout` reports `work_patch_qa2`, `state=working`, `surface=direct`.
- Web valid thread: `browser/post-final-network-thread-detail.snapshot.txt` shows the thread detail with reply composer for the existing thread.
- Web valid direct: `browser/post-final-network-direct-detail.snapshot.txt` shows `Trace update recorded.` and `working`.
- Web missing thread: `browser/post-final-network-missing-thread.snapshot.txt` shows `Thread unavailable` and no missing-thread reply composer.
- Web missing direct: `browser/post-final-network-missing-direct.snapshot.txt` shows `Direct room unavailable` and no missing-direct direct composer.
- Invalid-route negative check: `browser/post-final-invalid-route-negative-check.txt` confirms missing-conversation snapshots expose no reply/direct composer and no normal empty state.

Disruption probes:

- Invalid cross-container direct work lookup: `invalid-cross-container-work.exit` is `1`.
- Legacy `interaction_id` send request: `legacy-interaction-id.status` is `400`.
- Missing thread API: `api-missing-thread.stdout` returns the expected not-found error.
- Invalid direct API: `api-missing-direct.stdout` returns the expected validation error.
- Runtime E2E fault path: initial failure exposed BUG-001; `test-e2e-runtime-after-fix.log` passed after the root-cause fix.
- Web missing conversation route: initial browser probes exposed BUG-002; final snapshots show unavailable states without misleading composers.

Smoke/readiness checks only:

- Clean baseline `make verify` before scenario execution: `baseline-make-verify-clean.log` passed.
- Initial `make verify` with `bootstrap.env` sourced was invalid harness evidence because `AGH_HOME` pollution intentionally broke env-isolation tests; correction is documented in `execution-notes.md`.
- `make web-lint` and `make web-typecheck` were Web-specific guardrails after the Web fix; broad completion is supported by final `make verify`.

BROWSER EVIDENCE
----------------
Browser tool used: `browser-use` was requested by the bootstrap manifest, but the available tool list did not expose the Node REPL JS command required by the browser-use skill. `agent-browser` was used as the approved fallback and closed after the run.

Dev server: `source .compozy/tasks/network-threads/qa/bootstrap.env; AGH_WEB_API_PROXY_TARGET="$AGH_WEB_API_PROXY_TARGET" make web-dev`, served at `http://localhost:3001/` because `:3000` was already occupied.

Flows tested: 6.

Flow details:

- Thread list: `http://localhost:3001/network/builders/threads` -> thread list; Verdict: PASS. Evidence: `.compozy/tasks/network-threads/qa/screenshots/network-threads-list.png`.
- Existing thread detail: `http://localhost:3001/network/builders/threads/thread_builders_qa2` -> thread detail; Verdict: PASS. Evidence: `.compozy/tasks/network-threads/qa/screenshots/post-final-network-thread-detail.png`.
- Direct rooms list: `http://localhost:3001/network/builders/directs` -> direct rooms list; Verdict: PASS. Evidence: `.compozy/tasks/network-threads/qa/screenshots/network-directs-list.png`.
- Existing direct detail: `http://localhost:3001/network/builders/directs/direct_8eeac8fd8fc697295a67c6206051e959` -> direct detail; Verdict: PASS. Evidence: `.compozy/tasks/network-threads/qa/screenshots/post-final-network-direct-detail.png`.
- Missing thread detail: `http://localhost:3001/network/builders/threads/thread_missing_qa` -> unavailable state; Verdict: PASS. Evidence: `.compozy/tasks/network-threads/qa/screenshots/post-final-network-missing-thread.png`.
- Missing direct detail: `http://localhost:3001/network/builders/directs/direct_missing_qa` -> unavailable state; Verdict: PASS. Evidence: `.compozy/tasks/network-threads/qa/screenshots/post-final-network-missing-direct.png`.

Viewports tested: default desktop browser viewport.
Authentication: not required for local AGH Web QA.
Blocked flows: live provider-backed browser transcript was blocked by unavailable live LLM provider execution in this deterministic local QA lane.

TEST CASE COVERAGE
------------------
Test cases found: 7.
Executed: 7.

Results:

- `SMOKE-001`: PASS. Bootstrap, clean baseline verification, isolated daemon health, and post-gate daemon status passed. Bug: BUG-001 found during runtime E2E and fixed.
- `TC-SCEN-001`: PASS. Public thread creation and thread message persistence validated through CLI, API, and Web. Bug: none.
- `TC-SCEN-002`: PASS. Direct-room resolution and direct message isolation validated through CLI, API, and Web. Bug: none.
- `TC-SCEN-003`: PASS. Direct work trace and status stayed direct-scoped and visible in CLI/API/Web. Bug: none.
- `TC-INT-001`: PASS. CLI/API/Web cross-surface state agreement validated post-gate. Bug: none.
- `TC-UI-001`: PASS after fix. Missing thread/direct routes now show unavailable states without composer controls. Bug: BUG-002.
- `TC-REG-001`: PASS after fix. `make test-e2e-runtime`, `make test-e2e-web`, legacy rejection probes, and final `make verify` passed. Bug: BUG-001.

Not executed: none. Live LLM provider behavior was explicitly blocked and is not claimed as live-provider evidence.

ISSUES FILED
------------
Total: 2.
By severity:

- Critical: 0
- High: 1
- Medium: 1
- Low: 0

Details:

- BUG-001: Session event queries can race with session recorder finalization | Severity: High | Priority: P0 | Status: Fixed.
- BUG-002: Web network detail routes rendered composer controls before missing conversations resolved | Severity: Medium | Priority: P1 | Status: Fixed.

SUPPORTING COMMAND EVIDENCE
---------------------------
- Targeted runtime fix validation:
  - `go test -race -count=1 ./internal/session`: PASS.
  - `go test -race -count=1 ./internal/store/sessiondb`: PASS.
  - `make test-e2e-runtime`: PASS after fix; `test-e2e-runtime-after-fix.log`.
- Targeted Web fix validation:
  - `bunx vitest run web/src/systems/network/components/directs/direct-room.test.tsx web/src/systems/network/components/thread-overlay/thread-overlay.test.tsx web/src/systems/network/lib/query-options.test.ts`: `3 passed`, `20 passed`.
  - `make web-lint`: `0 warnings`, `0 errors`.
  - `make web-typecheck`: PASS.
  - `make test-e2e-web`: `19 passed`.
- Legacy scan:
  - `legacy-scan.txt`: 30 matches, all classified as rejection code, negative tests, invalid-shape docs, or non-wire UI discriminants.
- Final gate:
  - `final-make-verify.log`: PASS.

[QA_BOOTSTRAP]
manifest_path=/Users/pedronauck/Dev/compozy/agh2/.compozy/tasks/network-threads/qa/bootstrap-manifest.json
lab_root=/Users/pedronauck/dev/qa-labs/agh-network-threads-20260505-170603-687358-lab
runtime_home=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-0517b5b397c4/runtime
base_url=http://127.0.0.1:60149
verification_report=/Users/pedronauck/Dev/compozy/agh2/.compozy/tasks/network-threads/qa/verification-report.md
health_status=healthy
[/QA_BOOTSTRAP]
