# Sandbox Hard Cut, Docs, Web IA, Memory, and Internal Instructions

## Summary

Rename AGH's execution-environment feature to **Sandbox** across runtime, storage, APIs, CLI, hooks, web UI, generated contracts, landing page, docs, institutional memory, and internal agent instructions.

This is a greenfield hard cut: remove feature-sense `environment(s)` names and do not add aliases, redirects, dual JSON fields, or config fallback paths. Generic OS language remains valid: `Env`, `.env`, `os.Getenv`, "environment variable", and unrelated prose such as "production environment".

## Runtime And Contracts

- Move the Go domain from `internal/environment` to `internal/sandbox`; rename feature types/methods such as `EnvironmentProfile` -> `SandboxProfile`, `Config.Environments` -> `Config.Sandboxes`, `Defaults.Environment` -> `Defaults.Sandbox`, and `ResolveEnvironment` -> `ResolveSandbox`.
- Rename TOML from `[environments.<name>]` to `[sandboxes.<name>]`; default selection becomes `[defaults] sandbox = "<name>"`.
- Rename persisted and public fields:
  - `environment_ref` -> `sandbox_ref`.
  - `environment_id`, `environment_backend`, `environment_profile`, `environment_instance_id`, `environment_state`, provider state, and sync columns -> `sandbox_*`.
  - `environment_profiles` -> `sandbox_profiles`.
  - Daytona labels `agh_environment_id` -> `agh_sandbox_id`.
- Add a numbered SQLite migration that renames/rebuilds existing alpha DB columns to `sandbox_*`, then update the fresh schema so runtime code only reads/writes the new names.
- Rename hooks, Host API, logs, spans, and extension capabilities:
  - Hooks: `sandbox.prepare`, `sandbox.ready`, `sandbox.sync.before`, `sandbox.sync.after`, `sandbox.stop`.
  - Host API: `sandbox/list`, `sandbox/info`, `sandbox/exec`.
  - Capability: `sandbox.exec`.
  - ACP mock fixture step: `sandbox_exec`.
- Rename HTTP/OpenAPI/UDS settings collection endpoints from `/api/settings/environments` to `/api/settings/sandboxes`; regenerate OpenAPI and generated web types.
- Rename CLI surfaces:
  - `agh workspace add/edit --sandbox`.
  - `agh spawn --sandbox-profile`.
  - `agh config set sandboxes.<name>...`.
  - Regenerate CLI docs with `make cli-docs`.

## Web UI

- Remove the old Settings subroute for environments and remove that item from Settings navigation.
- Add top-level route `/sandbox` and a main sidebar item labeled **Sandbox** immediately after **Skills**.
- Treat the page as a first-class runtime surface, not a Settings section. Use AGH design tokens, flat depth, Inter typography, lucide iconography, and existing shell patterns.
- Surface copy:
  - Page title: `Sandbox`.
  - Primary action: `New sandbox profile`.
  - Empty state: `No sandbox profiles defined`.
  - Backend labels: `local`, `Daytona`; do not present E2B as implemented unless there is a real runtime provider.
- Move/rename web environment-specific adapters, hooks, fixtures, tests, stories, and data-test IDs into sandbox naming. Reuse generic settings primitives only when they are presentational and not tied to Settings navigation.

## Site, Docs, Memory, And Instructions

- Add a landing-page Sandbox section after the runtime section without changing the locked hero. The section should explain host execution vs Daytona sandbox execution, sync, lifecycle, and session metadata.
- Use a brand-consistent AGH visual/diagram for the landing section: host workspace -> AGH daemon -> sandbox provider -> session lifecycle/sync.
- Add `packages/site/content/runtime/core/sandbox/` with:
  - `index.mdx`: what Sandbox is, lifecycle, local vs remote execution, and implemented providers.
  - `profiles.mdx`: `[sandboxes.<name>]`, `[defaults] sandbox`, workspace `--sandbox`, and safe-spawn `--sandbox-profile`.
  - `daytona.mdx`: Daytona setup using `DAYTONA_API_KEY`, `api_url`, `target`, `image`/`snapshot`, `class`, `auto_stop`, `auto_archive`, sync modes, persistence, and a working example.
- Update runtime docs navigation/meta so Sandbox is discoverable near Skills/Workspaces.
- Fix `config-toml.mdx`, which currently says no environment config is implemented; replace it with sandbox config reference and examples.
- Update `internal/CLAUDE.md` and `internal/AGENTS.md`; both currently list `internal/environment` as "Env-profile resolution", and must instead document `internal/sandbox` as the sandbox-profile/provider runtime package.
- Update any root instruction references found by the final search guard if they describe the active feature using old `environment` vocabulary.
- Update `.compozy/tasks/*` artifacts that describe the active sandbox feature so stale `environment_*` contracts are not preserved as implementation truth.
- After implementation and verification, update `docs/_memory`:
  - Add a new lesson file such as `docs/_memory/lessons/L-NNN-sandbox-vocabulary-drift.md`.
  - Update `docs/_memory/lessons/README.md`.
  - Add or update a `Sandbox` glossary entry in `docs/_memory/glossary.md` if the term is absent or ambiguous.
  - Cite concrete evidence from the final diff/commit and verification results.
  - Map the lesson to Principle of Least Surprise, High Cohesion, and truthful public surfaces.
  - Do not add a new standing directive unless the final change reveals a genuinely new durable rule not already covered.

## Test Plan

- Focused Go tests: config, sandbox provider, workspace resolver/CRUD, session lifecycle, hooks, extension Host API, API core/spec/contracts, CLI, store migrations, daemon reconciliation, and ACP mock fixtures.
- Add/adjust tests proving the hard cut:
  - `[sandboxes.daytona-dev]` loads and `[environments.daytona-dev]` is rejected.
  - Workspace CRUD persists/returns `sandbox_ref`.
  - Safe spawn uses `sandbox_profiles`.
  - Hook dispatch uses `sandbox.prepare`.
  - Host API authorizes `sandbox/exec` with `sandbox.exec`.
  - Migration produces `sandbox_*` columns and removes old feature columns.
- Run `make codegen` and `make codegen-check`.
- Run web checks: `make web-typecheck`, `make web-lint`, and relevant route/component tests.
- Run site checks: `cd packages/site && bun run source:generate && bun run typecheck && bun run test && bun run build`.
- Run final `make verify`.
- Before completion, run an `rg` guard for feature-sense `environment` names across `internal`, `cmd`, `web`, `packages/site`, `openapi`, `.compozy/tasks`, `docs/_memory`, `internal/CLAUDE.md`, and `internal/AGENTS.md`; allow only generic OS/env-variable usages and unrelated prose.

## Assumptions And Defaults

- This is a hard cut, not a compatibility migration for public API/config/CLI names.
- The implemented providers documented as supported are `local` and `daytona`.
- Daytona docs reflect current provider behavior: `DAYTONA_API_KEY` is required for SSH access, `snapshot` wins over `image` when both are configured, and the default API URL remains `https://app.daytona.io/api` unless code changes it.
- The API may remain config-backed as `/api/settings/sandboxes`, but the user-facing web IA is top-level `/sandbox`.
