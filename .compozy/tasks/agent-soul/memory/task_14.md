# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Regenerate web and SDK contract consumers after Soul/Heartbeat/session-health/extension contracts; no UI editor.

## Important Decisions
- Closed real spec/handler drift by declaring `include_health` query parameter on `listSessions` and `getSession` in `internal/api/spec/spec.go`. Handler already parsed it; spec was missing.
- Skipped reformat of full `openapi/agh.json` and avoided changing the spec writer indent: surgically appended the `include_health` parameter blocks to `openapi/agh.json` to keep diff focused; canonical-JSON `codegen-check` tolerates indent/array-style differences.
- Did not introduce domain type aliases for Soul/Heartbeat/session health in web systems (no current web consumer uses them); coverage remains in `web/src/lib/agent-authored-context-contract.test.ts` to keep types truthful for future UI work.

## Learnings
- `make codegen` rewrites `openapi/agh.json` with `json.MarshalIndent(doc, "", "  ")` (2-space indent + array-per-line), but the committed file historically uses 4-space indent and compact short arrays. Running codegen blindly produces a 130k-line whitespace-only diff. Future renames or contract changes should reformat only the impacted spans rather than the entire file unless a focused reformat task is approved.
- `__dirname` works reliably in Vitest tests; `fileURLToPath(new URL("..", import.meta.url))` failed under jsdom env because `import.meta.url` isn't always a `file://` URL.
- Vitest project filter for `sdk/typescript/vitest.config.ts` is `extension-sdk`, not `sdk-typescript`.

## Files / Surfaces
- `internal/api/spec/spec.go`: added `boolQueryParam("include_health", ...)` to `listSessions` + `getSession` operation specs.
- `openapi/agh.json`: surgical addition of two `include_health` query parameter blocks for `listSessions` and `getSession`.
- `web/src/generated/agh-openapi.d.ts`: regenerated via `make codegen` (only the two `include_health` query slots changed).
- `web/src/lib/agent-authored-context-contract.test.ts`: added `OperationQuery` import and a Soul/Heartbeat session health include_health assertion test case.
- `web/src/lib/agent-authored-context-no-ui.test.ts`: new guard scanning `web/src/**` for forbidden Soul/Heartbeat/SessionHealth editor/form/composer/panel components.
- `sdk/typescript/src/index.ts`: added named type exports for Soul, Heartbeat, SessionHealth, AuthoredContext, wake/result/reason/source enums, session health/status DTOs, session soul refresh DTOs.
- `sdk/typescript/src/authored-context-contracts.test.ts`: added an SDK barrel guard asserting no `*Editor|*Form|*Composer|*Settings|*Panel|*Inspector|*Workbench|*Builder` exports for Soul/Heartbeat/SessionHealth.

## Errors / Corrections
- First attempt at `make codegen` produced a 130k-line whitespace-only diff in `openapi/agh.json`. Reverted via `git checkout HEAD -- openapi/agh.json` and applied the include_health additions surgically to keep the diff minimal.
- First `agent-authored-context-no-ui.test.ts` used `fileURLToPath(new URL("..", import.meta.url))`, which threw `TypeError: The URL must be of scheme file` under Vitest jsdom; switched to `join(__dirname, "..")`.
- `bunx vitest run --project sdk-typescript` failed with no project; correct project filter is `extension-sdk`.

## Ready for Next Run
- Task 15 (docs) should reference SDK named exports added here (no SDK-internal type imports needed) and document the `?include_health=true` parameter as visible on `listSessions`/`getSession` HTTP endpoints. Docs should also note that no Soul/Heartbeat editor UI exists in MVP; web surfaces consume generated types only.
- Task 17 (QA execution) typecheck step is already green; running `make bun-typecheck` + `make bun-test` will exercise the new contract assertions.
