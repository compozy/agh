# Provider Model Catalog - Task List

## MVP Boundary

Tasks 01-11 implement the MVP described in `_techspec.md`: provider model config hard cut, model catalog persistence/service/sources, ACP session config options, HTTP/UDS/CLI/extension/web surfaces, generated contracts, docs, and cross-surface regression hardening. Task 01 is a hard-cut contract/codegen co-ship boundary: it must not leave old provider model fields for later residue cleanup. Tasks 12-13 are the required QA tail. Out of scope: Droid discovery, fake ACP sessions for discovery, using `models.dev` as account availability proof, and treating `models.curated` as an allowlist.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Provider Config and Builtin Model Hard Cut | completed | critical | - |
| 02 | Model Catalog Persistence | completed | critical | - |
| 03 | Model Catalog Service and Catalog Sources | pending | critical | task_01, task_02 |
| 04 | Live Provider Discovery Sources | completed | high | task_03 |
| 05 | Daemon Catalog Wiring | completed | high | task_03, task_04 |
| 06 | ACP SDK Upgrade and Config Options | completed | high | task_01 |
| 07 | HTTP, UDS, CLI, and OpenAI Model Projection Surfaces | completed | critical | task_05 |
| 08 | Extension Model Source Contract | completed | high | task_05, task_07 |
| 09 | Web Model Catalog Experience | completed | high | task_01, task_06, task_07 |
| 10 | Generated Contracts and Runtime Docs | completed | high | task_01, task_07, task_08, task_09 |
| 11 | Cross-Surface Regression Hardening | completed | critical | task_06, task_07, task_08, task_09, task_10 |
| 12 | QA Plan and Test Coverage | completed | high | task_11 |
| 13 | Real-Scenario QA Execution | pending | critical | task_12 |
