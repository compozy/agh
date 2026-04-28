# Deterministic Local QA Bootstrap for AGH

## Summary

- The main bottlenecks are in the non-bundled QA flow (`real-scenario-qa`, `qa-execution`, `agh-worktree-isolation`) plus the global `browser-use` and `codex-timed-loop` runtime under `~/.codex`.
- Confirmed issues:
  - `real-scenario-qa` only automates a thin `mkdir` bootstrap; most runtime/provider/browser setup is recreated manually in every QA wave.
  - `real-scenario-qa` prefers `browser-use:browser`, but `qa-execution` still instructs `agent-browser` as the default web path.
  - Skill helper paths are ambiguous; agents have already treated bundled helper scripts as “missing” because instructions said `scripts/...` without an explicit skill-root path.
  - Isolation covers `AGH_HOME`, ports, and tmux, but not provider `HOME`/`CODEX_HOME`, `AGH_WEB_API_PROXY_TARGET`, or sequential config writes; that already caused `missing field path` and config-write races.
  - `browser-use` setup is expensive and should be deferred until the web surface is actually healthy.
  - The global timed loop encourages a fresh pass, but it does not explicitly prefer reusing a healthy QA lab before rebuilding one.

## Key Changes

- Add a new repo-local skill: `agh-qa-bootstrap [scenario-slug]`.
- Make `agh-qa-bootstrap` the mandatory first step for production-like local QA. It must write:
  - `<qa-output-path>/qa/bootstrap-manifest.json`
  - `<qa-output-path>/qa/bootstrap.env`
- The bootstrap contract must include:
  - `SCENARIO_SLUG`
  - `WORKSPACE_PATH`
  - `QA_OUTPUT_PATH`
  - `AGH_HOME`
  - `AGH_HTTP_PORT`
  - `AGH_UDS_PATH`
  - `TMUX_BRIDGE_SOCKET`
  - `AGH_WEB_API_PROXY_TARGET`
  - `PROVIDER_HOME`
  - `PROVIDER_CODEX_HOME`
  - `BROWSER_MODE`
  - `BROWSER_BLOCKER`
- `real-scenario-qa` must delegate bootstrap to `agh-qa-bootstrap`.
- `qa-execution` must consume the bootstrap manifest when present instead of redetecting runtime paths and browser mode ad hoc.
- `agh-worktree-isolation` remains the low-level runtime primitive, but it is no longer the full QA bootstrap story.

## Implementation Changes

- New `agh-qa-bootstrap` skill:
  - Add a concise `SKILL.md`, helper scripts, and references/templates needed for a deterministic QA bootstrap.
  - Bootstrap order is fixed:
    1. Resolve slug and fresh-vs-reuse mode.
    2. Create or reuse the scenario workspace.
    3. Allocate isolated AGH runtime values.
    4. Create an isolated provider home that does not inherit `~/.codex/config.toml`, hooks, plugins, or loops.
    5. Run contract discovery using an explicit repo-root helper path.
    6. Write `bootstrap-manifest.json` and `bootstrap.env`.
    7. Seed reusable lab scaffolding for repeated QA waves.
    8. Perform cheap browser/tool availability preflight.
  - Prefer reusing a healthy lab by default. Only create a fresh lab when the user explicitly asks for a fresh wave/folder or the previous lab fails health checks.

- Update existing QA skills:
  - `real-scenario-qa`
    - Bootstrap via `agh-qa-bootstrap`.
    - Prefer reusing a healthy lab.
    - Only branch to a fresh wave when explicitly requested or when health checks fail.
    - Delay `browser-use:browser` loading until the dev server and proxy target are ready.
  - `qa-execution`
    - Switch browser policy to `browser-use` first, `agent-browser` only after documented preflight failure.
    - Use the explicit helper path `.agents/skills/qa-execution/scripts/discover-project-contract.py --root .`.
    - Treat `agh config set` in parallel against the same runtime home as forbidden.
  - `qa-report`
    - Align shared output/browser references with the new bootstrap contract.

- Enforce the workflow in instructions:
  - Root `AGENTS.md` and `CLAUDE.md`
    - `Release / scenario QA` requires `agh-qa-bootstrap + real-scenario-qa + qa-report + qa-execution`.
    - Real QA must isolate provider `HOME`/`CODEX_HOME`; raw global `~/.codex` cannot be reused for Codex-backed provider sessions.
    - Real QA against an isolated daemon must export `AGH_WEB_API_PROXY_TARGET`.
    - `agh config set` must be sequential for a shared runtime home.
    - Skill helpers must be referenced by explicit repo-root paths.
  - `web/AGENTS.md` and `web/CLAUDE.md`
    - Isolated daemon QA must follow the bootstrap manifest/env instead of assuming `localhost:2123`.
  - `~/.codex/AGENTS.md` and `~/.codex/codex-timed-loop`
    - Continuations for long QA loops should reuse a healthy lab when a bootstrap manifest and ledger already exist.

- Add one authoring guardrail:
  - `skill-best-practices` must require explicit repo-root helper paths and classify helper scripts as read-only vs bootstrap vs mutating.

## Test Plan

- Validate metadata/frontmatter for the new and edited skills.
- Confirm that every referenced helper script exists at the documented path.
- Smoke the bootstrap:
  - Run `agh-qa-bootstrap release-qa-smoke`.
  - Verify manifest/env creation.
  - Verify isolated provider home does not inherit global config/hooks/plugins/loops.
  - Verify `AGH_WEB_API_PROXY_TARGET` matches the isolated daemon port.
- Real flow smoke:
  - Start daemon/web from bootstrap outputs.
  - Create at least one Codex-backed provider session without `missing field path`.
  - Run a `browser-use` preflight when available.
  - Run a documented `agent-browser` fallback when browser-use is unavailable.
- Regression flow:
  - Run one short `real-scenario-qa` pass and confirm startup reduces to bootstrap + health checks rather than rewriting the whole lab.
  - Run one `codex-timed-loop` continuation and confirm it prefers reusing a healthy lab before rebuilding.
- Final repo gate:
  - `make verify`

## Assumptions and Defaults

- Default behavior is now “reuse the healthy lab,” not “always start fresh.”
- Explicit “fresh analysis/folder creation” language still forces a new wave.
- `browser-use:browser` remains the preferred web gate; `agent-browser` remains the documented fallback.
- `bootstrap-manifest.json` becomes the authoritative handoff artifact across bootstrap, execution, browser preflight, and loop continuations.
