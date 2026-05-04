# Session Driver Override - Task List

**GREENFIELD (alpha):** nao sacrificar qualidade por retrocompatibilidade. As tasks abaixo assumem mudancas limpas no runtime, persistence, API, e `web/`, com apenas a migracao/reparo explicito de metadata legado definido no TechSpec e nenhum fallback silencioso de provider.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Provider-Aware Session Resolution and Validation | completed | medium | - |
| 02 | Session Provider Runtime Plumbing and On-Disk Persistence | completed | high | task_01 |
| 03 | Global Session Index Migration and Legacy Provider Repair | completed | high | task_02 |
| 04 | Explicit Session Provider Contracts and Generated Surfaces | completed | high | task_02 |
| 05 | Workspace Provider Catalog and Automatic Creator Defaults | completed | medium | task_02 |
| 06 | Web Session Creation Dialog and Resume Failure UX | completed | high | task_04, task_05 |
| 07 | Session Provider Override QA Plan and Regression Artifacts | completed | high | task_06 |
| 08 | Session Provider Override QA Execution and End-to-End Validation | completed | critical | task_07 |
