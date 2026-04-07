# Refac V2 — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Extract shared frontmatter package | pending | high | — |
| 02 | Create internal/api/contract and migrate shared DTOs | pending | high | task_01 |
| 03 | Migrate CLI to internal/api/contract | pending | medium | task_02 |
| 04 | Re-root shared API core into internal/api/core and merge apisupport | pending | high | task_02 |
| 05 | Re-root HTTP and UDS transports plus shared API test utilities | pending | critical | task_03, task_04 |
| 06 | Split persistence into store, store/sessiondb, and store/globaldb | pending | critical | task_05 |
| 07 | Extract canonical transcript assembly into internal/transcript | pending | medium | task_06 |
| 08 | Move dream orchestration into internal/memory/consolidation | pending | high | task_06 |
| 09 | Narrow consumer interfaces and remove transitional bridges | pending | critical | task_07, task_08 |
