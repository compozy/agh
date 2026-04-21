# Assistant-UI Migration - Task List

**GREENFIELD (alpha):** do not sacrifice quality for backward compatibility. Tasks below assume the hand-rolled chat renderer can be deleted cleanly in the same series of PRs that introduce the assistant-ui surface. No feature flags, no dual paths, no transcript compat shims.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Install assistant-ui and Scaffold Themed Thread Components | pending | medium | — |
| 02 | Unify Transcript Endpoint on AI SDK UIMessage Shape | pending | high | — |
| 03 | Introduce SessionChatRuntimeProvider with History Adapter | pending | high | task_01, task_02 |
| 04 | Register Per-Tool Renderers via makeAssistantToolUI | pending | medium | task_03 |
| 05 | Render data-agh-permission via makeAssistantDataUI | pending | medium | task_03 |
| 06 | Integrate Session Lifecycle Controls (Stop/Resume/Clear) | pending | medium | task_03 |
| 07 | Delete Legacy Chat Renderer, Hooks, Mappers, and Store Fields | pending | high | task_03, task_04, task_05, task_06 |
| 08 | Rewrite Storybook Stories and Integration Tests | pending | medium | task_07 |
| 09 | End-to-End QA and Parity Validation | pending | high | task_01, task_02, task_03, task_04, task_05, task_06, task_07, task_08 |
