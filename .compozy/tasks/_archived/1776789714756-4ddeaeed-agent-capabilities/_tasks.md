# Agent Capabilities - Task List

**GREENFIELD (alpha):** nao sacrificar qualidade por retrocompatibilidade. As tasks abaixo assumem que a capability catalog surface pode introduzir mudancas estruturais limpas no runtime/network quando alinhadas ao TechSpec, sem compat layers, merge semantics ambiguas, ou discovery parcial.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Capability Catalog Loader and Validation | completed | medium | - |
| 02 | Capability-Aware Runtime Join Plumbing | completed | high | task_01 |
| 03 | Brief Capability Projection in Peer Cards | completed | high | task_02 |
| 04 | Explicit Rich Capability Discovery via Whois | completed | high | task_03 |
| 05 | Runtime Authoring Documentation for Capabilities | completed | low | task_03, task_04 |
| 06 | Agent Capabilities QA Plan and Regression Artifacts | completed | high | task_04 |
| 07 | Agent Capabilities QA Execution and End-to-End Validation | completed | critical | task_06 |
