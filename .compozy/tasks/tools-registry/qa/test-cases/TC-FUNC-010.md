# TC-FUNC-010 — Workspace overlays preserve deterministic precedence for tools config

- **Priority:** P2
- **Type:** Functional / config overlay
- **Trace:** Task 02, TechSpec Config Lifecycle

## Objective

Prove that workspace-overlay tools config merges deterministically: workspace overlays workspace-level overrides global, and agent-local fields override agent-global. Conflicts resolve by precedence, not by order of file load.

## Test Steps

1. Set global `[tools.policy].external_default = "disabled"`; workspace overrides to `"ask"`.
   - **Expected:** Effective value `"ask"` for that workspace.
2. Set global `[tools.policy].trusted_sources = ["mcp:fake_http"]`; workspace adds `["extension:test_ext"]`.
   - **Expected:** Effective list = union (or workspace-replaces-global per documented semantics — verify behavior matches docs).
3. Set agent-local `tools = ["agh__skill_view"]`; workspace agent override `tools = ["agh__skill_*"]`.
   - **Expected:** Override applies; precedence documented.
4. Hot-reload not in MVP scope; daemon restart picks up overlay.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/config -run TestToolsConfigOverlay`
