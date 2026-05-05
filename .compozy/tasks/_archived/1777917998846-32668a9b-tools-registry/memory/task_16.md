# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute Task 16 real-scenario QA for tools-registry against a fresh isolated AGH lab.
- Required evidence includes smoke P0, targeted/full/security lanes, real TypeScript + Go extension-host tool execution, external MCP/OAuth and hosted MCP flows, CLI/HTTP/UDS parity, web diagnostics via browser-use or documented fallback, docs/build checks, `make test-e2e-runtime`, `make test-e2e-web`, sentinel scan, final `make verify`, and `.compozy/tasks/tools-registry/qa/verification-report.md`.

## Important Decisions
- Use Task 15 QA artifacts under `.compozy/tasks/tools-registry/qa/` as the execution contract; do not redefine priorities, paths, sentinels, or scope.
- Fresh QA lab is required for this independent pass; reuse is not allowed unless this exact active run later resumes from its manifest.
- Bootstrap manifest is mirrored into `.compozy/tasks/tools-registry/qa/bootstrap-manifest.json`; the canonical generated manifest remains at the lab QA path.

## Learnings
- Shared workflow memory says Task 16 must seed `qa/fixtures/redaction-sentinels.json`, run smoke first with stop-on-P0-failure, then targeted/full/security-redaction lanes, file `BUG-NNN.md` for reproduced defects, root-cause fix them, and only then write the final report and complete tracking.
- Root AGH guidance requires isolated `AGH_HOME`, unique daemon ports, provider homes, tmux socket, and `AGH_WEB_API_PROXY_TARGET` for web QA when the daemon is not on the default port.
- AGH repo does not contain `scripts/discover-project-contract.py`; used the installed `qa-execution` helper path explicitly. It identified `make verify` as canonical, E2E support through `make test-e2e-runtime`/`make test-e2e-web`, and a web UI surface.
- Baseline `make verify` failed before smoke execution because UDS `WithHomePaths` did not realign the default config socket away from the process provider home. `BUG-001.md` filed and fixed by tracking explicit `WithConfig` usage and rebuilding defaults from `WithHomePaths` only when config was not explicitly supplied.
- UDS tests that exercise constructor dependency validation must not rely on `t.TempDir()` as the actual Unix socket parent on macOS; the existing `shortSocketPath` helper avoids unrelated portable socket path failures.
- Initial QA bootstrap manifest was invalid for daemon/UDS coverage because workspace-derived socket paths exceeded the 103-byte portable Unix socket limit. `BUG-002.md` filed and fixed by validating socket-limited paths and allocating short unique runtime/provider homes under the system temp directory when needed.
- Smoke TC-SEC-001 initially failed the real-execution guard: `go test ./internal/tools -run TestPolicyDenyAll` exited 0 with `[no tests to run]`. `BUG-003.md` filed and fixed by adding `TestPolicyDenyAll` across native Go, extension-host, and MCP descriptors.

## Files / Surfaces
- `.compozy/tasks/tools-registry/qa/test-plans/*`
- `.compozy/tasks/tools-registry/qa/test-cases/TC-*.md`
- `.agents/skills/agh-qa-bootstrap/SKILL.md`
- `.agents/skills/real-scenario-qa/SKILL.md`
- `.agents/skills/agh-worktree-isolation/SKILL.md`
- `.agents/skills/qa-execution/SKILL.md`
- `.compozy/tasks/tools-registry/qa/bootstrap-manifest.json`
- `.compozy/tasks/tools-registry/qa/fixtures/redaction-sentinels.json`
- `.compozy/tasks/tools-registry/qa/behavioral-scenario-charter.md`
- `.compozy/tasks/tools-registry/qa/issues/BUG-001.md`
- `.compozy/tasks/tools-registry/qa/issues/BUG-002.md`
- `.compozy/tasks/tools-registry/qa/issues/BUG-003.md`
- `internal/api/udsapi/server.go`
- `internal/api/udsapi/server_test.go`
- `internal/api/httpapi/server.go`
- `internal/api/httpapi/server_test.go`
- `internal/tools/dispatch_test.go`
- `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py`

## Errors / Corrections
- `python3 scripts/check-test-conventions.py ...` is unavailable in this repo; `rg --files -g check-test-conventions.py` returned no script.
- Initial targeted rerun still failed because the new UDS regression test and existing missing-skills case used default `homePaths.DaemonSocket` from `t.TempDir()`; corrected those cases to inject `shortSocketPath`.
- Targeted rerun passed: `go test -race -count=1 ./internal/api/udsapi ./internal/api/httpapi ./internal/config -run "TestNewWithHomePathsRealignsDefaultConfig|TestNewRequiresSessionManagerTaskServiceObserverAndWorkspaceResolver|TestLoadUsesDotEnvForAGHHome" -v`.
- Baseline rerun passed: `make verify` under isolated provider `HOME`/`CODEX_HOME` with no global `AGH_HOME` completed with Go lint `0 issues`, `DONE 6970 tests`, and `OK: all package boundaries respected`.
- Smoke TC-SEC-001 targeted rerun passed after adding coverage: `go test -race -count=1 ./internal/tools -run TestPolicyDenyAll -v`.

## Ready for Next Run
- Fresh corrected lab bootstrapped:
  - manifest: `/Users/pedronauck/dev/qa-labs/agh-tools-registry-task16-20260429-075857-781754-lab/qa-artifacts/qa/bootstrap-manifest.json`
  - lab root: `/Users/pedronauck/dev/qa-labs/agh-tools-registry-task16-20260429-075857-781754-lab`
  - runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-ed6aa49b48f8/runtime`
  - base URL: `http://127.0.0.1:64177`
  - provider home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-ed6aa49b48f8/provider`
  - provider Codex home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-ed6aa49b48f8/provider/.codex`
  - browser mode: `browser-use`
- Next action: rerun the smoke P0 lane from TC-SEC-001 and stop on any subsequent P0 failure.
