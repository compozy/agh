# TC-FUNC-042 — TypeScript create-extension `tool-provider` template scaffolds buildable tool

- **Priority:** P2
- **Type:** Functional / scaffolding
- **Trace:** Task 07

## Test Steps

1. Run `npx @agh/create-extension --template tool-provider-typescript test-tool-ext`.
2. Generated project contains `extension.toml` with `resources.tools` plus `extension.json` for tool schemas (TOML inline schema not supported per shared decision).
3. `bun install`, `bun build`, `bun test` succeed.
4. Install resulting extension into daemon → tool is callable.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `bun test sdk/create-extension --grep tool-provider-typescript`
