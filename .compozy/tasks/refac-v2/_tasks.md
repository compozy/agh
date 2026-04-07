# Refac V2 — Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Extract shared frontmatter package | completed | high | — |
| 02 | Create internal/api/contract and migrate shared DTOs | completed | high | task_01 |
| 03 | Migrate CLI to internal/api/contract | completed | medium | task_02 |
| 04 | Re-root shared API core into internal/api/core and merge apisupport | completed | high | task_02 |
| 05 | Re-root HTTP and UDS transports plus shared API test utilities | completed | critical | task_03, task_04 |
| 06 | Split persistence into store, store/sessiondb, and store/globaldb | completed | critical | task_05 |
| 07 | Extract canonical transcript assembly into internal/transcript | completed | medium | task_06 |
| 08 | Move dream orchestration into internal/memory/consolidation | completed | high | task_06 |
| 09 | Narrow consumer interfaces and remove transitional bridges | completed | critical | task_07, task_08 |
