---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: resolved
file: .compozy/tasks/agent-soul/qa/bootstrap.env
line: 10
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_Irc-,comment:PRRC_kwDOR5y4QM69Xby9
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Remove machine-specific absolute paths and user-identifying values from committed bootstrap config.**

Lines 2-10 expose local username/path details and lock this artifact to one workstation. That is a privacy/compliance risk and makes QA replay brittle across environments.

 

<details>
<summary>🔧 Proposed sanitization (template-style)</summary>

```diff
-export SCENARIO_SLUG=agent-soul-<local-run-id>
-export WORKSPACE_PATH=<local-qa-lab>
-export QA_OUTPUT_PATH=<local-qa-lab>/qa-artifacts
-export AGH_HOME=<local-qa-lab>/.agh/runtime
-export AGH_HTTP_PORT=49165
-export AGH_UDS_PATH=<local-qa-lab>/.agh/runtime/aghd.sock
-export TMUX_BRIDGE_SOCKET=<local-qa-lab>/.agh/runtime/tmux-bridge.sock
-export AGH_WEB_API_PROXY_TARGET=http://127.0.0.1:49165
-export PROVIDER_HOME=<local-qa-lab>/.provider-home
-export PROVIDER_CODEX_HOME=<local-qa-lab>/.provider-home/.codex
+export SCENARIO_SLUG="${SCENARIO_SLUG:-agent-soul-local}"
+export WORKSPACE_PATH="${WORKSPACE_PATH:-$PWD}"
+export QA_OUTPUT_PATH="${QA_OUTPUT_PATH:-$WORKSPACE_PATH/qa-artifacts}"
+export AGH_HOME="${AGH_HOME:-$WORKSPACE_PATH/.agh/runtime}"
+export AGH_HTTP_PORT="${AGH_HTTP_PORT:-49165}"
+export AGH_UDS_PATH="${AGH_UDS_PATH:-$AGH_HOME/aghd.sock}"
+export TMUX_BRIDGE_SOCKET="${TMUX_BRIDGE_SOCKET:-$AGH_HOME/tmux-bridge.sock}"
+export AGH_WEB_API_PROXY_TARGET="${AGH_WEB_API_PROXY_TARGET:-http://127.0.0.1:$AGH_HTTP_PORT}"
+export PROVIDER_HOME="${PROVIDER_HOME:-$WORKSPACE_PATH/.provider-home}"
+export PROVIDER_CODEX_HOME="${PROVIDER_CODEX_HOME:-$PROVIDER_HOME/.codex}"
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
export SCENARIO_SLUG="${SCENARIO_SLUG:-agent-soul-local}"
export WORKSPACE_PATH="${WORKSPACE_PATH:-$PWD}"
export QA_OUTPUT_PATH="${QA_OUTPUT_PATH:-$WORKSPACE_PATH/qa-artifacts}"
export AGH_HOME="${AGH_HOME:-$WORKSPACE_PATH/.agh/runtime}"
export AGH_HTTP_PORT="${AGH_HTTP_PORT:-49165}"
export AGH_UDS_PATH="${AGH_UDS_PATH:-$AGH_HOME/aghd.sock}"
export TMUX_BRIDGE_SOCKET="${TMUX_BRIDGE_SOCKET:-$AGH_HOME/tmux-bridge.sock}"
export AGH_WEB_API_PROXY_TARGET="${AGH_WEB_API_PROXY_TARGET:-http://127.0.0.1:$AGH_HTTP_PORT}"
export PROVIDER_HOME="${PROVIDER_HOME:-$WORKSPACE_PATH/.provider-home}"
export PROVIDER_CODEX_HOME="${PROVIDER_CODEX_HOME:-$PROVIDER_HOME/.codex}"
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In @.compozy/tasks/agent-soul/qa/bootstrap.env around lines 1 - 10, The
bootstrap env file contains machine-specific absolute paths and a username in
variables like SCENARIO_SLUG, WORKSPACE_PATH, QA_OUTPUT_PATH, AGH_HOME,
AGH_UDS_PATH, TMUX_BRIDGE_SOCKET, AGH_WEB_API_PROXY_TARGET, PROVIDER_HOME and
PROVIDER_CODEX_HOME; replace these hard-coded values with neutral template
placeholders or environment-default expressions (e.g.
${WORKSPACE_PATH:-/path/to/workspace}) and/or relative paths, remove any
user-identifying parts from SCENARIO_SLUG, and commit a sanitized template
(e.g., bootstrap.env.example) so CI/other devs populate real values locally
rather than storing local absolute paths in the repo.
```

</details>

<!-- fingerprinting:phantom:triton:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `.compozy/tasks/agent-soul/qa/bootstrap.env` was committed with a one-off QA lab slug, user-specific absolute paths, and a fixed proxy target derived from that single local run.
- Impact: the file leaks local workstation/user path details and cannot be replayed cleanly by CI or another developer without editing every path-bearing variable.
- Fix approach: keep the bootstrap contract but convert the committed values to neutral environment-default expressions. `WORKSPACE_PATH` becomes the base, path variables derive from it or `AGH_HOME`, and `AGH_WEB_API_PROXY_TARGET` derives from `AGH_HTTP_PORT`. Existing browser-mode variables stay unchanged because they are not machine-identifying.

## Resolution

- Replaced hard-coded local QA lab values in `.compozy/tasks/agent-soul/qa/bootstrap.env` with environment-default expressions.
- Sanitized this scoped review artifact so it does not reintroduce the same local absolute paths in committed review text.
- Verification: `make verify` completed successfully on 2026-05-02 15:26:00 -03 with exit code 0.
