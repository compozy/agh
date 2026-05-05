# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 05 MCP Auth and Skill Security: OAuth 2.1 + PKCE remote MCP auth lifecycle, durable token storage, redacted status/config/API/CLI surfaces, `agh mcp auth login|status|logout`, skill and managed-extension symlink escape rejection, downstream web/site impact assessment, clean verification, tracking updates, and one local commit.

## Important Decisions

- ADR-003 is the governing auth design: implement a first-class `internal/mcp/auth` subsystem rather than static token configuration.
- Task 01 migration/retry foundations are available; durable token schema work should use `internal/store.RunMigrations` patterns, and retryable auth refresh should reuse `internal/retry` where applicable.
- Public/operator surfaces must expose redacted auth status only; token material, OAuth codes, PKCE verifiers, and client secrets stay behind narrow internal APIs.
- Remote MCP settings are represented as typed config (`transport`, `url`, `auth`) and redacted public views; bearer tokens are persisted only in global DB `mcp_auth_tokens`.
- The web settings MCP editor remains stdio-only for mutation in this task; it displays remote endpoints and disables editing remote entries rather than inventing incomplete remote-auth UI.

## Learnings

- Shared Hermes memory records Task 01-04 as completed. Task 05 can rely on `internal/store.RunMigrations`, `internal/retry`, typed health foundations, and generated contract workflows.
- Hermes tools/security analysis calls out existing skill escape gaps in `internal/skills/loader.go`, `internal/skills/provenance.go`, and managed extension loading; Task 05 scope is symlink escape hardening, not the broader content scanner/trust-tier matrix.
- `packages/site` already has tracked CLI/configuration documentation covering `agh mcp auth`, remote MCP OAuth config, and symlink escape behavior; no additional tracked site diff was required after impact review.
- A daemon-level restart/status test was added after review to cover persisted MCP auth status through global DB reopen, matching the task integration expectation.
- Final self-review tightened the loopback OAuth listener so explicit redirect URLs must use localhost or a loopback IP; `:0` redirect URLs are rewritten to the bound listener port.

## Files / Surfaces

- Planned surfaces: `internal/config` MCP models, `internal/api/contract` and `internal/api/core` settings/status DTOs, new `internal/mcp/auth`, `internal/cli` MCP auth commands, `internal/skills`, `internal/extension`, generated API/web/SDK surfaces if contracts change, and site docs if user-facing behavior changes.
- Implemented surfaces: `internal/config`, `internal/mcp/auth`, `internal/store/globaldb`, `internal/settings`, `internal/api/{contract,core,spec}`, `internal/daemon`, `internal/cli`, `internal/skills`, `internal/extension`, `openapi/agh.json`, and `web/src/**` generated/contracts/settings fixtures/tests.

## Errors / Corrections

- Initial full verification found web generated-contract fallout: settings MCP fixtures/tests and the stdio editor needed explicit `transport` handling. Fixed with generated type/test updates and remote rows displayed as endpoints.
- Go lint found CLI cleanup/noctx/line-length/huge-param issues and a skill loader cyclomatic issue. Fixed by propagating cleanup errors, using context-aware listeners, passing config pointers, splitting scan helper logic, and using `strconv.FormatBool`.

## Ready for Next Run

- Task 05 implementation is complete in commit `156b8e8b feat: add mcp oauth auth security`. Verification before commit: targeted CLI/daemon tests passed and final `make verify` passed (`DONE 5811 tests`, `0 issues`, package boundaries OK).
