# Workspace Entity — Task List

**GREENFIELD (alpha):** não sacrificar qualidade por retrocompatibilidade — mudanças breaking são aceitáveis quando alinhadas ao TechSpec; evitar migrações ou código só para compat com estado/API antiga.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Workspace domain types, errors, and store/resolver interfaces | completed | low | — |
| 02 | GlobalDB workspaces table and WorkspaceStore implementation | completed | high | task_01 |
| 03 | Resolver implementation with cache and tests | completed | high | task_01, task_02 |
| 04 | Session Manager workspace ID and Resolver injection | completed | high | task_03 |
| 05 | Config workspace-scoped loading and agent paths | completed | medium | task_04 |
| 06 | Daemon wiring and dream consolidation | completed | medium | task_03, task_04, task_05 |
| 07 | Skills registry delegation to Resolver | completed | medium | task_03 |
| 08 | ACP AdditionalDirs for workspace roots | completed | medium | task_04 |
| 09 | Observe and memory workspace ID references | completed | medium | task_04 |
| 10 | HTTP API workspace routes and session contract | completed | high | task_04, task_06 |
| 11 | UDS API workspace mirror | completed | medium | task_10 |
| 12 | CLI workspace and session commands | completed | medium | task_06, task_11 |
| 13 | Web workspace UI and session grouping | completed | high | task_10 |
