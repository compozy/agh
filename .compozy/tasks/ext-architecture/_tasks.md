# Extension Architecture — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Minimal Tool struct and ToolProvider interface | completed | low | — |
| 02 | Shared subprocess lifecycle package | completed | high | — |
| 03 | Extension manifest parser (TOML and JSON) | completed | medium | — |
| 04 | Capability checker and source-trust tiers | completed | medium | task_03 |
| 05 | Extension registry (SQLite) | completed | medium | task_03, task_04 |
| 06 | Extension Manager (lifecycle orchestrator) | completed | high | task_02, task_04, task_05 |
| 07 | Host API handler (bidirectional JSON-RPC) | completed | high | task_04, task_06 |
| 08 | Daemon boot integration | completed | medium | task_06 |
| 09 | CLI commands (list, install, enable, disable) | completed | medium | task_05, task_06 |
| 10 | TypeScript SDK (@agh/extension-sdk) | completed | high | task_06, task_07 |
| 11 | Reference extensions (Go and TypeScript) | pending | medium | task_06, task_07, task_10 |
