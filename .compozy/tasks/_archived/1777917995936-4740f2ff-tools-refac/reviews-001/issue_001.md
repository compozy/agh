---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: .compozy/tasks/tools-refac/qa/bootstrap.env
line: 10
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulIr,comment:PRRC_kwDOR5y4QM680KGx
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid committing machine-specific absolute paths in bootstrap env.**

Line 2 through Line 10 hardcode one developer’s local directories and temp paths. This makes the scenario non-portable and leaks local machine path details.

<details>
<summary>Suggested refactor (template-style defaults)</summary>

```diff
-export WORKSPACE_PATH=/Users/pedronauck/dev/qa-labs/agh-tools-refac-real-scenario-20260430-074748-514234-lab
-export QA_OUTPUT_PATH=/Users/pedronauck/dev/qa-labs/agh-tools-refac-real-scenario-20260430-074748-514234-lab/qa-artifacts
-export AGH_HOME=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/runtime
-export AGH_UDS_PATH=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/runtime/aghd.sock
-export TMUX_BRIDGE_SOCKET=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/runtime/tmux-bridge.sock
-export PROVIDER_HOME=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/provider
-export PROVIDER_CODEX_HOME=/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-afda92fbd4d8/provider/.codex
+export WORKSPACE_PATH="${WORKSPACE_PATH:-$HOME/dev/qa-labs/${SCENARIO_SLUG}-lab}"
+export QA_OUTPUT_PATH="${QA_OUTPUT_PATH:-${WORKSPACE_PATH}/qa-artifacts}"
+export AGH_HOME="${AGH_HOME:-${TMPDIR:-/tmp}/aghqa/runtime}"
+export AGH_UDS_PATH="${AGH_UDS_PATH:-${AGH_HOME}/aghd.sock}"
+export TMUX_BRIDGE_SOCKET="${TMUX_BRIDGE_SOCKET:-${AGH_HOME}/tmux-bridge.sock}"
+export PROVIDER_HOME="${PROVIDER_HOME:-${TMPDIR:-/tmp}/aghqa/provider}"
+export PROVIDER_CODEX_HOME="${PROVIDER_CODEX_HOME:-${PROVIDER_HOME}/.codex}"
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In @.compozy/tasks/tools-refac/qa/bootstrap.env around lines 2 - 10, The file
currently commits machine-specific absolute paths for environment variables
(e.g., WORKSPACE_PATH, QA_OUTPUT_PATH, AGH_HOME, AGH_UDS_PATH,
TMUX_BRIDGE_SOCKET, PROVIDER_HOME, PROVIDER_CODEX_HOME), which should be
replaced with portable, template-style or runtime-resolved values; update
bootstrap.env to use either environment-variable fallbacks or placeholders
(e.g., reference an external config or use ${VAR:-default} style semantics)
instead of hardcoded developer paths, ensure socket/temporary paths default to
OS temp (or relative workspace) and keep AGH_HTTP_PORT as a configurable value,
and document the expected variables so CI/dev machines can supply their own
values.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `bootstrap.env` currently commits absolute developer and temporary paths for the QA lab, runtime home, sockets, and provider home. This is non-portable and leaks machine-local path details. Fix by changing the file to environment-variable fallbacks based on `SCENARIO_SLUG`, `$HOME`, and `${TMPDIR:-/tmp}` while keeping the HTTP port and proxy target overrideable.
