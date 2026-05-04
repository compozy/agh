# QA Bootstrap Contract

The bootstrap helper writes two canonical artifacts under:

`<qa-output-path>/qa/`

## Required files

- `bootstrap-manifest.json`
- `bootstrap.env`

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
    "BROWSER_BLOCKER": ""
  },
  "browser": {
    "mode": "browser-use",
    "blocker": ""
  },
  "project_contract": {}
}
```

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
