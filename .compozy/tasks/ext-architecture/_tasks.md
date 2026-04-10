# Extension Architecture — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Minimal Tool struct and ToolProvider interface | pending | low | — |
| 02 | Shared subprocess lifecycle package | pending | high | — |
| 03 | Extension manifest parser (TOML and JSON) | pending | medium | — |
| 04 | Capability checker and source-trust tiers | pending | medium | task_03 |
| 05 | Extension registry (SQLite) | pending | medium | task_03, task_04 |
| 06 | Extension Manager (lifecycle orchestrator) | pending | high | task_02, task_04, task_05 |
| 07 | Host API handler (bidirectional JSON-RPC) | pending | high | task_04, task_06 |
| 08 | Daemon boot integration | pending | medium | task_06 |
| 09 | CLI commands (list, install, enable, disable) | pending | medium | task_05, task_06 |
| 10 | TypeScript SDK (@agh/extension-sdk) | pending | high | task_06, task_07 |
| 11 | Reference extensions (Go and TypeScript) | pending | medium | task_06, task_07, task_10 |
