## VERIFICATION REPORT

Claim: Bridge adapter integration suite passes after the QA fixes.
Command: `make test-integration`
Executed: 2026-04-15 12:21:00 -03
Exit code: 0
Output summary: 4,082 integration-tagged tests passed in 42.619s. Notable package completions included `internal/daemon` in 28.146s and `internal/extension` in 40.457s.
Warnings: none
Errors: none
Verdict: PASS

Claim: Repository verification gate passes from a clean rerun after the last code change.
Command: `make verify`
Executed: 2026-04-15 12:21:45 -03
Exit code: 0
Output summary: OpenAPI generation, web format/lint/typecheck/test/build, Go lint, Go tests, Go build, and package-boundary verification all passed. Web tests reported `78 passed` / `649 passed`; Go tests reported `DONE 3785 tests in 15.230s`; boundary check ended with `OK: all package boundaries respected`.
Warnings: `ld: warning: -bind_at_load is deprecated on macOS` from the `golangci-lint` link step
Errors: none
Verdict: PASS

Claim: Targeted daemon deadlock regression is fixed and the full daemon integration package is green.
Command: `go test -race -tags integration -count=1 -v ./internal/daemon`
Executed: 2026-04-15 12:20:28 -03
Exit code: 0
Output summary: The previously timing-out `TestCreateEnabledBridgeAfterBootReloadsErroredExtension` passed, the new lifecycle regression tests passed, and the full `internal/daemon` integration package passed in 18.383s.
Warnings: verbose Gin route registration output during daemon startup tests
Errors: none
Verdict: PASS

Claim: Public daemon, CLI, and HTTP bridge surfaces work against an isolated live daemon home.
Command: `AGH_HOME=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/agh-qa-home-14oq8c8v AGH_BRIDGE_TELEGRAM_LISTEN_ADDR=127.0.0.1:52371 AGH_BRIDGE_TELEGRAM_API_BASE_URL=http://127.0.0.1:52370 ./bin/agh daemon start --foreground`
Executed: 2026-04-15 12:24:59 -03
Exit code: 0
Output summary: The daemon booted on `http://127.0.0.1:52369` with a live Telegram extension install. CLI `daemon status`, `extension status telegram`, `bridge create`, `bridge list`, `bridge get`, `bridge test-delivery`, and `bridge routes` all worked. HTTP `GET /api/bridges/providers`, `PUT/GET/DELETE /api/bridges/:id/secret-bindings/:binding_name`, `POST /api/bridges/:id/test-delivery`, and `GET /api/observe/health` all worked. The bridge instance correctly converged from `starting` to `auth_required` without secret material, and health reported `auth_failures_total: 1`.
Warnings: daemon startup logged pre-existing skill frontmatter warnings for `allowed-tools`, `argument-hint`, and `user-invocable`; Gin emitted debug-mode startup warnings
Errors: none for reachable unauthenticated flows
Verdict: PASS

Claim: Authenticated bridge restart through the public secret-binding surface is blocked in the stock daemon binary.
Command: `curl -sf -X PUT http://127.0.0.1:52369/api/bridges/brg-7784fca7c6e8d8cf/secret-bindings/bot_token -H 'content-type: application/json' -d '{"vault_ref":"vault://qa/telegram/bot","kind":"token"}' && AGH_HOME=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/agh-qa-home-14oq8c8v ./bin/agh bridge restart brg-7784fca7c6e8d8cf -o json`
Executed: 2026-04-15 12:26:14 -03
Exit code: 1
Output summary: Restart failed and rolled back. The bridge remained `auth_required`, and `extension status telegram` reported `state: "error"` / `health: "unhealthy"`.
Warnings: none
Errors: `daemon: reload extensions for bridge instance "brg-7784fca7c6e8d8cf": extension "telegram" initialize: extension: resolve bridge runtime for "telegram": daemon: resolve bound secrets for bridge instance "brg-7784fca7c6e8d8cf": daemon: bridge secret resolver is required`
Verdict: FAIL

## BROWSER EVIDENCE (when Web UI flows were tested)

Dev server: daemon-served SPA via `AGH_HOME=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/agh-qa-home-14oq8c8v AGH_BRIDGE_TELEGRAM_LISTEN_ADDR=127.0.0.1:52371 AGH_BRIDGE_TELEGRAM_API_BASE_URL=http://127.0.0.1:52370 ./bin/agh daemon start --foreground` at `http://127.0.0.1:52369`
Flows tested: 3
Flow details:

- Workspace bootstrap: `http://127.0.0.1:52369/bridges` -> `http://127.0.0.1:52369/bridges` | Verdict: PASS
  Evidence: initial page required workspace setup; selecting "Use global workspace" navigated into the main app shell
- Bridges page render: `http://127.0.0.1:52369/bridges` -> `http://127.0.0.1:52369/bridges` | Verdict: PASS
  Evidence: bridge detail panel rendered `QA Telegram Bridge` with `AUTH_REQUIRED`; screenshot at `.compozy/tasks/bridge-adapters/qa/screenshots/bridges-page.png`
- Bridges search filter: `http://127.0.0.1:52369/bridges` -> `http://127.0.0.1:52369/bridges` | Verdict: PASS
  Evidence: searching for `nope` hid the bridge list item; searching for `QA` restored the live bridge row and detail panel
  Viewports tested: default only
  Authentication: not required
  Blocked flows: authenticated bridge activation through persisted secret bindings is blocked by `BUG-005` because the stock daemon has no bridge secret resolver

## TEST CASE COVERAGE (when qa-report artifacts exist)

Test cases found: 56
Executed: 9
Results:

- `SMOKE-001`: PASS | Bug: none
- `SMOKE-002`: PASS | Bug: none
- `SMOKE-008`: PASS | Bug: `BUG-001`, `BUG-002`, `BUG-004`
- `TC-FUNC-009`: PASS | Bug: `BUG-001`, `BUG-002`
- `TC-INT-001`: PASS | Bug: `BUG-004`
- `TC-INT-007`: PASS | Bug: none
- `TC-INT-009`: PASS | Bug: none
- `TC-INT-012`: PASS | Bug: `BUG-003`
- `TC-INT-006`: BLOCKED | Reason: `BUG-005` / stock daemon restart path cannot resolve persisted bridge secret bindings
  Not executed: remaining case files were not run as standalone manual scripts during this QA pass; they were covered by the repository umbrella gates where automated coverage exists

## ISSUES FILED

Total: 5
By severity:

- Critical: 1
- High: 4
- Medium: 0
- Low: 0
  Details:
- `BUG-001`: Linear agent-session final ack drops replace_remote_message_id | Severity: High | Priority: P1 | Status: Fixed
- `BUG-002`: Telegram final no-op delivery performs a duplicate edit | Severity: High | Priority: P1 | Status: Fixed
- `BUG-003`: Managed extension install rejects runtime node_modules because it copies dev-only symlinks | Severity: High | Priority: P1 | Status: Fixed
- `BUG-004`: Bridge lifecycle deadlocks when same-extension reload resolves managed instances | Severity: Critical | Priority: P0 | Status: Fixed
- `BUG-005`: Public bridge secret bindings are unusable in the stock daemon because no secret resolver is wired | Severity: High | Priority: P1 | Status: Open
