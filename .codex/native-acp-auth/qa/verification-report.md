# Native ACP Auth Boundary Verification Report

## Claim

Native ACP providers no longer require AGH-bound API keys, while `bound_secret` providers still
enforce required credentials. Backend runtime, CLI, API/OpenAPI, web Settings, docs, and memory
surfaces are updated consistently.

## Command Evidence

| Command                                                                                                                                                                                                                                                                                                                                                       | Executed timestamp                                                                                                                                                                         |            Exit code | Output summary                                                                                                                          | Verdict                                                                      |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------: | --------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---- |
| `make codegen-check`                                                                                                                                                                                                                                                                                                                                          | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | OpenAPI and generated TypeScript contract had no drift.                                                                                 | PASS                                                                         |
| `go test ./internal/config ./internal/session ./internal/settings ./internal/api/core ./internal/cli -run 'Test(BuiltinProvidersContainExpectedCommands\|ProviderAuthModeValidation\|ProviderConfigOverrideMergesWithBuiltins\|SaveBootstrapConfig\|Load\|SessionStartEnv\|PrepareProviderForStart\|ProviderAuth\|SessionProviderOption\|Settings)' -count=1` | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Focused provider-auth backend tests passed.                                                                                             | PASS                                                                         |
| `bunx vitest run web/src/lib/settings-api-contract.test.ts web/src/hooks/routes/use-settings-providers-page.test.tsx web/src/routes/_app/settings/-providers.test.tsx web/src/systems/settings/adapters/settings-api.test.ts`                                                                                                                                 | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | 4 files, 48 Settings provider tests passed.                                                                                             | PASS                                                                         |
| `make bun-lint`                                                                                                                                                                                                                                                                                                                                               | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Oxfmt and oxlint completed with 0 warnings and 0 errors.                                                                                | PASS                                                                         |
| `make bun-typecheck`                                                                                                                                                                                                                                                                                                                                          | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Turbo typecheck completed across 5 package tasks.                                                                                       | PASS                                                                         |
| `make bun-test`                                                                                                                                                                                                                                                                                                                                               | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | 329 test files and 2074 tests passed.                                                                                                   | PASS                                                                         |
| `make lint`                                                                                                                                                                                                                                                                                                                                                   | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | GolangCI-Lint completed with 0 issues.                                                                                                  | PASS                                                                         |
| `make test`                                                                                                                                                                                                                                                                                                                                                   | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | 7908 Go tests passed.                                                                                                                   | PASS                                                                         |
| `make build`                                                                                                                                                                                                                                                                                                                                                  | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Web production build, TypeScript check, and Go build passed. Vite emitted the existing non-blocking chunk-size warning.                 | PASS                                                                         |
| `make verify`                                                                                                                                                                                                                                                                                                                                                 | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Full monorepo gate passed: codegen-check, bun-lint, bun-typecheck, bun-test, web-build, fmt, lint, test, build, boundaries.             | PASS                                                                         |
| `AGH_HOME="$(mktemp -d)" go run ./cmd/agh provider auth status claude --no-probe -o json`                                                                                                                                                                                                                                                                     | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Reported `auth_mode: native_cli`, `state: native_cli`, `env_policy: filtered`, `home_policy: operator`, with no credential requirement. | PASS                                                                         |
| `AGH_HOME="$(mktemp -d)" go run ./cmd/agh provider auth status openrouter --no-probe -o json`                                                                                                                                                                                                                                                                 | 2026-05-03T22:24:58Z                                                                                                                                                                       |                    0 | Reported `auth_mode: bound_secret`, `state: missing_required`, and redacted missing `OPENROUTER_API_KEY` metadata.                      | PASS                                                                         |
| `rg -n "Direct ACP built-ins keep their default API-key slots\|export OPENAI_API_KEY\|provider API key in the current shell\|API key metadata\|API key env\\s\*\\                                                                                                                                                                                             | \|OPENAI_API_KEY\|GEMINI_API_KEY\|BLACKBOX_API_KEY\|QODER_PERSONAL_ACCESS_TOKEN" packages/site/content config.toml internal/config/provider.go web/src/systems/settings/mocks/fixtures.ts` | 2026-05-03T22:24:58Z | 1                                                                                                                                       | No stale native-provider API-key claims found. `rg` exit 1 means no matches. | PASS |

## Warnings

- Vite reported existing chunk-size warnings during web build. The build completed successfully and
  the warning was not introduced as a failing gate.
- Live third-party provider login was not executed because this QA run did not use operator
  credentials or browser/device auth.

## Errors

- None remaining.

## Behavioral Evidence

Operator journey:

- `agh provider auth status claude --no-probe -o json` proves a native provider is visible and
  actionable without an AGH-bound API key.
- `agh provider auth status openrouter --no-probe -o json` proves an API-key gateway remains
  blocked until its required AGH-managed secret is present.

Live agent/LLM evidence:

- Blocked by design for this QA pass: no live provider account credentials or native browser/device
  login were used. Reachable local boundaries were validated through CLI, config/runtime tests,
  API/OpenAPI tests, web Settings tests, and the full monorepo gate.

Artifacts produced and used:

- Accepted plan: `.codex/plans/2026-05-03-native-acp-auth-isolation.md`
- QA plan: `.codex/native-acp-auth/qa/test-plans/native-acp-auth-test-plan.md`
- QA cases: `.codex/native-acp-auth/qa/test-cases/`
- Memory lesson: `docs/_memory/lessons/L-015-native-provider-auth-boundary.md`

Disruption probes:

- Native provider with `credential_slots` and no `auth_mode = "bound_secret"` is covered by config
  validation tests.
- Missing `bound_secret` provider credential is covered by runtime/CLI/settings tests and by the
  `openrouter` CLI status command above.
- Isolated env/home policy is covered by session/provider runtime tests.

Cross-surface state checks:

- Backend config and session runtime expose `auth_mode`, `env_policy`, and `home_policy`.
- Settings API/OpenAPI and generated web types include auth metadata and nullable redacted auth
  status.
- Web Settings editor/card flows distinguish native CLI auth, bound secrets, and no-auth providers.
- Runtime docs, internal backend guidance, and memory lessons describe the new auth boundary.

Smoke checks:

- Focused Go and Vitest checks passed before the full gate.
- Final `make verify` passed after all edits.

## Browser Evidence

Browser UI flows were not executed in this QA pass. Web behavior was validated through focused
Settings route/component/adapter tests, generated type checks, `make bun-test`, `make
bun-typecheck`, and production `web-build` inside `make verify`.

## Verdict

PASS
