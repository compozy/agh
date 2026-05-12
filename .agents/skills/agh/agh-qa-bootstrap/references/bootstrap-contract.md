# QA Bootstrap Contract

The bootstrap helper writes two canonical artifacts under:

`<qa-output-path>/qa/`

## Required files

- `bootstrap-manifest.json`
- `bootstrap.env`
- `scenario-contract.json`
- `behavioral-scenario-charter.yaml`
- `journey-log.jsonl`
- `provider-attempt.json`

## Required manifest fields

```json
{
  "schema_version": 1,
  "scenario_slug": "release-qa",
  "workspace_path": "/abs/path/to/lab",
  "qa_output_path": "/abs/path/to/lab/qa-artifacts",
  "manifest_path": "/abs/path/to/lab/qa-artifacts/qa/bootstrap-manifest.json",
  "bootstrap_env_path": "/abs/path/to/lab/qa-artifacts/qa/bootstrap.env",
  "status": {
    "reused_lab": true,
    "health": "healthy",
    "notes": []
  },
  "env": {
    "SCENARIO_SLUG": "release-qa",
    "WORKSPACE_PATH": "/abs/path/to/lab",
    "QA_OUTPUT_PATH": "/abs/path/to/lab/qa-artifacts",
    "AGH_HOME": "/abs/path/to/lab/.agh/runtime",
    "AGH_HTTP_PORT": "2235",
    "AGH_UDS_PATH": "/abs/path/to/lab/.agh/runtime/aghd.sock",
    "TMUX_BRIDGE_SOCKET": "/abs/path/to/lab/.agh/runtime/tmux-bridge.sock",
    "AGH_WEB_API_PROXY_TARGET": "http://127.0.0.1:2235",
    "PROVIDER_HOME": "/abs/path/to/lab/.provider-home",
    "PROVIDER_CODEX_HOME": "/abs/path/to/lab/.provider-home/.codex",
    "BROWSER_MODE": "browser-use",
    "BROWSER_BLOCKER": "",
    "SCENARIO_CONTRACT": "/abs/path/to/lab/qa-artifacts/qa/scenario-contract.json",
    "BEHAVIORAL_CHARTER": "/abs/path/to/lab/qa-artifacts/qa/behavioral-scenario-charter.yaml",
    "JOURNEY_LOG": "/abs/path/to/lab/qa-artifacts/qa/journey-log.jsonl",
    "PROVIDER_ATTEMPT": "/abs/path/to/lab/qa-artifacts/qa/provider-attempt.json",
    "AUDIT_COMMAND": "/abs/path/to/repo/.agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py",
    "PLAYBOOK_REF": "northstar-pay",
    "KICKOFF_POSTED": "false",
    "KICKOFF_TIMESTAMP": ""
  },
  "browser": {
    "mode": "browser-use",
    "blocker": ""
  },
  "project_contract": {}
}
```

## QA evidence contract files

- `scenario-contract.json` defines the release-grade minimums that downstream QA must satisfy before a `PASS` claim.
- `behavioral-scenario-charter.yaml` is JSON-compatible YAML. It must name the startup situation, operator intent, business outcome, agents, channels, task tree, provider plan, cross-surface targets, disruption probes, and artifacts. When `--playbook` was passed, the charter is materialized from the playbook spec and includes `playbook_ref`, `required_deliverables`, and `required_collaboration`.
- `journey-log.jsonl` is append-only structured evidence. Each meaningful CLI/API/Web/runtime/provider action must add one row.
- `provider-attempt.json` records live provider-backed proof or the exact blocked boundary. A blocked provider boundary supports a `BLOCKED` result, not a live-provider `PASS`.
- The auditor writes `qa-audit-report.json` and `qa-audit-report.md`; exit code `2` is a blocking QA failure.

## Playbook scaffolding (when --playbook is passed)

The bootstrap helper additionally writes the following under `WORKSPACE_PATH`:

- `.agh/playbook.json` — the resolved playbook spec (the canonical structured JSON parsed from `references/playbooks/<ref>.md`).
- `.agh/agents/<agent-id>.json` — one file per agent declared by the playbook (id, role, persona, system_prompt, workspace_id, workspace_path, skills, playbook_ref).
- `.agh/tasks/open-tasks.json` — array of open tasks with owner_agent, owner_workspace_id, owner_workspace_path, deliverable_type, deliverable_path, review_required_by, channel, playbook_ref.
- `.agh/disruption-seeds.json` — playbook disruption_probe_seeds for downstream consumers.
- `workspaces/<workspace-name>/README.md` — per-workspace stub README.
- `knowledge/<...>` — every knowledge file declared by the playbook.

`PLAYBOOK_REF` and `KICKOFF_POSTED=false` are written to the manifest env. `real-scenario-qa` Step 4 flips `KICKOFF_POSTED=true` and sets `KICKOFF_TIMESTAMP` after posting the single in-persona kickoff.

## Reuse policy

- Default to a fresh lab for each new QA pass, even when an older lab exists for the same feature or scenario.
- Reuse a lab only when the caller passes the exact manifest path from the same active QA session or loop continuation.
- Repair that same-session lab in place before rebuilding when only derived files are missing.
- Rebuild when the requested manifest is missing, unreadable, or points at missing directories.

## Mandatory launch rules

- Bound-secret, brokered, or explicitly isolated-home provider commands: `HOME="$PROVIDER_HOME" CODEX_HOME="$PROVIDER_CODEX_HOME" <cmd>`
- `native_cli` providers with `home_policy=operator`: preserve the operator `HOME` / native login state unless the scenario explicitly validates isolated provider-home behavior
- Web dev server for isolated daemon QA: `AGH_WEB_API_PROXY_TARGET="$AGH_WEB_API_PROXY_TARGET" make web-dev`
- Config mutations such as `agh config set` must run sequentially when they target the same isolated home.
- Before claiming behavior-first QA completion, run `python3 "$AUDIT_COMMAND" --qa-output-path "$QA_OUTPUT_PATH" --strict` and include its result in the verification report.

## Machine-readable continuation block

Append this block to the end of a QA summary whenever a continuation may need to reuse the lab:

```text
[QA_BOOTSTRAP]
manifest_path=/abs/path/to/lab/qa-artifacts/qa/bootstrap-manifest.json
lab_root=/abs/path/to/lab
runtime_home=/abs/path/to/lab/.agh/runtime
base_url=http://127.0.0.1:2235
verification_report=/abs/path/to/lab/qa-artifacts/qa/verification-report.md
health_status=healthy
[/QA_BOOTSTRAP]
```

Keep the keys exactly as shown so external loop tooling can parse them deterministically.
