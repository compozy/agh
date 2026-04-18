# Settings UI — Task List

**GREENFIELD (alpha):** não sacrificar qualidade por retrocompatibilidade — mudanças breaking são aceitáveis quando alinhadas ao TechSpec; evitar migrações, shims ou código só para compat com estado/API antiga.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Comment-preserving config editors and write targets | completed | high | — |
| 02 | Settings service orchestration in internal/settings | completed | high | task_01 |
| 03 | Daemon relaunch helper and restart operation store | completed | high | — |
| 04 | Settings API contract and OpenAPI surface | completed | high | task_02, task_03 |
| 05 | Shared settings handlers in api/core | completed | high | task_02, task_03, task_04 |
| 06 | HTTP settings transport and loopback mutation policy | completed | high | task_05 |
| 07 | UDS settings transport and parity coverage | completed | high | task_05 |
| 08 | Settings entrypoint and route shell | completed | medium | task_06, task_07 |
| 09 | web/src/systems/settings domain scaffold | completed | high | task_08 |
| 10 | General, Memory, and Observability pages | completed | high | task_09 |
| 11 | Skills, Automation, and Network summary pages | completed | high | task_09 |
| 12 | Providers and Environments collection pages | completed | high | task_09 |
| 13 | MCP Servers scoped collection page | pending | high | task_09 |
| 14 | Hooks and Extensions page | pending | high | task_09 |
| 15 | Settings QA plan and regression artifacts | pending | high | task_10, task_11, task_12, task_13, task_14 |
| 16 | Settings QA execution and daemon-served browser E2E | pending | critical | task_15 |
