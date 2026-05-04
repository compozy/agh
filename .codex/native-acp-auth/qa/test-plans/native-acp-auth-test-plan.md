# Native ACP Auth Boundary Test Plan

## Executive Summary

This QA plan validates that AGH distinguishes native CLI provider authentication from AGH-managed
bound-secret provider authentication across runtime config, process launch, settings APIs, web UI,
and public documentation.

The primary risk is a regression where native ACP providers such as Claude Code, Codex, Gemini CLI,
OpenCode, Hermes, or OpenClaw are blocked by missing API-key environment variables. The secondary
risk is secret leakage or accidental secret inheritance when native providers are launched.

## Scope

In scope:

- Provider config resolution, validation, merge, and root `config.toml` defaults.
- Session provider startup environment and isolated provider home behavior.
- `agh provider auth status/login` CLI behavior.
- Settings API/OpenAPI/web Settings provider metadata and editor behavior.
- Runtime docs, internal memory, and policy instructions.

Out of scope:

- Live browser/device login against third-party provider accounts.
- Changing provider-specific external CLI authentication implementations.
- Database schema migration behavior; this change does not add persistent columns.

## Behavioral Scenario Charter

Operator intent: run AGH with native ACP providers that already manage their own credentials, while
still supporting API-key gateways whose secrets are owned by AGH or the daemon service manager.

Startup situation: the daemon has no `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`,
`BLACKBOX_API_KEY`, `KIMI_API_KEY`, or `QODER_PERSONAL_ACCESS_TOKEN` in its environment.

Agent roles: a general agent using a native ACP provider and a gateway agent using a `pi_acp`
provider with a required bound secret.

Expected artifacts:

- Resolved native providers report `auth_mode = native_cli`, `env_policy = filtered`, and
  `home_policy = operator`.
- Bound-secret providers report missing required credentials until their `env:` or `vault:` secret
  is available.
- Settings and web surfaces display redacted auth state without exposing secret values.
- Public docs tell operators to use native provider login for direct ACP CLIs and bound secrets for
  API-key providers.

Disruption probes:

- Add a `credential_slots` entry to a native provider without changing `auth_mode`.
- Launch a provider with `env_policy = isolated` while secret-shaped variables exist in the daemon
  environment.
- Launch a provider with `home_policy = isolated` and confirm it uses a private AGH-owned home.

## Test Strategy

Smoke readiness:

- `make codegen-check`
- Focused Go tests for config/session/settings/API/CLI provider auth.
- Focused web Vitest coverage for settings provider contract/editor/card behavior.

Release-grade behavioral evidence:

- Validate config and runtime launch code paths directly with Go tests that exercise missing native
  env vars and missing bound secrets.
- Validate web/API contract propagation with generated OpenAPI TypeScript types and Settings tests.
- Validate docs consistency with targeted text scans for obsolete native provider key claims.
- Run final `make verify` from the repository root.

## Environment Requirements

- macOS or Linux development host.
- Go toolchain configured for the repository.
- Bun workspace dependencies installed.
- No live provider account credentials required for automated validation.

## Entry Criteria

- The accepted implementation plan exists at `.codex/plans/2026-05-03-native-acp-auth-isolation.md`.
- Codegen has been run after API contract changes.
- QA artifacts live under `.codex/native-acp-auth/qa/`.

## Exit Criteria

- All P0 automated checks pass.
- Native ACP providers are not modeled as required-key providers.
- Bound-secret providers still fail when required secrets are missing.
- No docs or UI surfaces instruct operators that native providers universally require API keys.
- Final `make verify` passes.

## Risk Assessment

| Risk                                                   | Probability | Impact | Mitigation                                                         |
| ------------------------------------------------------ | ----------- | ------ | ------------------------------------------------------------------ |
| Native provider still blocked by missing API key       | Medium      | High   | Config/session tests cover native providers with no env secrets.   |
| Bound-secret regression lets missing API key launch    | Medium      | High   | CLI/settings/runtime tests cover required missing secret state.    |
| Secret-shaped daemon env leaks into native subprocess  | Medium      | High   | Env policy tests cover filtered and isolated process environments. |
| Web Settings shows misleading provider status          | Medium      | Medium | Web contract, hook, route, adapter, and card tests cover status.   |
| Docs continue to tell operators API keys are universal | Medium      | Medium | Targeted docs scan and site docs updates cover stale language.     |

## Deliverables

- Test plan: `.codex/native-acp-auth/qa/test-plans/native-acp-auth-test-plan.md`
- Test cases: `.codex/native-acp-auth/qa/test-cases/`
- Verification report: `.codex/native-acp-auth/qa/verification-report.md`
