---
name: agh-qa-bootstrap
description: Builds a reusable local QA bootstrap for AGH by creating a realistic scenario workspace, isolating AGH runtime paths and provider home, writing bootstrap-manifest.json and bootstrap.env, discovering the repository verification contract, and recording browser policy for downstream real-scenario-qa or qa-execution runs. Use when local QA would otherwise rebuild the lab from scratch or inherit broken global ~/.codex state. Do not use for single-command unit-test runs, static planning, or browser-only checks without daemon or workspace setup.
trigger: explicit
argument-hint: "[scenario-slug]"
---

# AGH QA Bootstrap

Create a deterministic local QA lab quickly, default to a fresh lab for each new QA pass, and hand off a canonical bootstrap manifest to downstream QA skills. Reuse an existing lab only when continuing the same active QA session or loop.

Bootstrap is infrastructure only. It does not validate AGH behavior, prove live agent/LLM behavior, satisfy `real-scenario-qa`, or replace CLI/API/Web/operator journey evidence.

## Required Inputs

- **scenario-slug** (optional): short context for the QA lab, such as `release-qa`, `autonomy`, or `hooks-network`. Defaults to `release-candidate`.

## Procedures

**Step 1: Run the Bootstrap CLI**

1. Resolve the scenario slug from the user request. Default to `release-candidate`.
2. For a new QA pass, execute the repo-root bootstrap helper:
   `python3 .agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "<scenario-slug>" --repo-root .`
3. Only when continuing the same active QA session or loop and a manifest path is already known, reuse that exact lab:
   `python3 .agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "<scenario-slug>" --repo-root . --reuse-manifest "<manifest-path>"`
4. Read the helper output and record:
   - `SCENARIO_SLUG`
   - `WORKSPACE_PATH`
   - `QA_OUTPUT_PATH`
   - `BOOTSTRAP_MANIFEST`
   - `BOOTSTRAP_ENV`
   - `AGH_HOME`
   - `AGH_HTTP_PORT`
   - `AGH_UDS_PATH`
   - `TMUX_BRIDGE_SOCKET`
   - `AGH_WEB_API_PROXY_TARGET`
   - `PROVIDER_HOME`
   - `PROVIDER_CODEX_HOME`
   - `BROWSER_MODE`
   - `BROWSER_BLOCKER`
   - `SCENARIO_CONTRACT`
   - `BEHAVIORAL_CHARTER`
   - `JOURNEY_LOG`
   - `PROVIDER_ATTEMPT`
   - `AUDIT_COMMAND`
   - `REUSED_LAB`

**Step 2: Verify the Bootstrap Contract**

1. Read `references/bootstrap-contract.md`.
2. Open `<QA_OUTPUT_PATH>/qa/bootstrap-manifest.json` and treat it as the canonical handoff between bootstrap, `real-scenario-qa`, `qa-execution`, browser setup, audit execution, and timed-loop continuations.
3. Open `<QA_OUTPUT_PATH>/qa/bootstrap.env` only when shell export lines are needed.
4. Treat `REUSED_LAB=true` as valid only when the manifest came from the same active QA session or loop continuation. Do not reuse older labs across separate QA passes just because they target the same feature or scenario slug.
5. Confirm the helper created `<QA_OUTPUT_PATH>/qa/scenario-contract.json`, `<QA_OUTPUT_PATH>/qa/behavioral-scenario-charter.yaml`, `<QA_OUTPUT_PATH>/qa/journey-log.jsonl`, and `<QA_OUTPUT_PATH>/qa/provider-attempt.json`. These files are evidence scaffolding only; they do not validate behavior until downstream QA fills them and the auditor passes.

**Step 3: Launch Downstream QA with the Manifest**

1. Pass the resolved `QA_OUTPUT_PATH` to `qa-report` and `qa-execution`.
2. Downstream QA must execute real operator journeys and live provider-backed agent behavior when reachable. Do not count successful bootstrap, health checks, or generated directories as real-scenario evidence.
3. When starting provider-backed commands, follow the provider's home policy from the manifest/config:
   - Bound-secret, brokered, or explicitly isolated-home lanes:
     `HOME="$PROVIDER_HOME" CODEX_HOME="$PROVIDER_CODEX_HOME" <provider-command>`
   - `native_cli` providers with `home_policy=operator`: preserve the operator `HOME` / native login state and do **not** rewrite it to `PROVIDER_HOME` unless the scenario explicitly tests isolated provider-home behavior.
4. When starting `make web-dev` or any Web surface that proxies to the daemon, export:
   `AGH_WEB_API_PROXY_TARGET="$AGH_WEB_API_PROXY_TARGET"`
5. Keep `agh config set` and any other config mutation against the same isolated home strictly sequential. Do not parallelize writes against the same config file.
6. Downstream QA must run the validation auditor from `AUDIT_COMMAND` before claiming behavior-first completion. The auditor writes `qa-audit-report.json` and `qa-audit-report.md`:
   `python3 .agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py --qa-output-path "$QA_OUTPUT_PATH" --strict`

**Step 4: Report Reuse State**

1. When QA completes, keep the manifest path, lab root, `AGH_HOME`, base URL, and verification report path visible in the final QA summary.
2. For timed-loop or continuation-driven QA, include the machine-readable QA bootstrap block from `references/bootstrap-contract.md` so the next round can reuse the lab instead of rebuilding it.

## Error Handling

- If the bootstrap helper reports `REUSED_LAB=false` because health checks failed, use the fresh manifest it just wrote instead of trying to revive the stale state manually.
- If a bound-secret, brokered, or Codex-specific provider fails with global config errors such as malformed `config.toml`, confirm that commands are using the manifest-derived `PROVIDER_HOME` / `PROVIDER_CODEX_HOME`. If a `native_cli` provider with `home_policy=operator` fails, confirm the lane preserved the operator `HOME` instead of incorrectly rewriting it to `PROVIDER_HOME`.
- If Web flows hit the wrong daemon, confirm `AGH_WEB_API_PROXY_TARGET` matches the manifest and restart the Web dev server with that env.
