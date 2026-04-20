# Unified Capabilities - Task List

**GREENFIELD (alpha):** nao sacrificar qualidade por retrocompatibilidade. As tasks abaixo assumem que `recipe` pode ser removido de forma limpa do runtime, do wire e da documentacao, com `capability` tornando-se o unico artefato de autoria, discovery e transfer no AGH.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Unified Capability Schema, Canonicalization, and Session Projection | pending | high | — |
| 02 | Replace Recipe Wire Kind with Capability Envelopes | pending | high | task_01 |
| 03 | Preserve Recipe Operational Semantics Under Capability | pending | high | task_02 |
| 04 | Align Discovery, Peer Details, and API Contracts with Unified Capabilities | pending | high | task_01, task_03 |
| 05 | Rewrite RFC 003 and Runtime Capability Guide | pending | medium | task_04 |
| 06 | Update `web/` Network UX and Typed Client for Unified Capabilities | pending | high | task_04 |
| 07 | Update `packages/site` Protocol Reference and Examples | pending | high | task_05 |
| 08 | Update `packages/site` Runtime Capability Docs | pending | medium | task_05 |
| 09 | Unified Capabilities QA Plan and Regression Artifacts | pending | high | task_01, task_02, task_03, task_04, task_05, task_06, task_07, task_08 |
| 10 | Unified Capabilities QA Execution and End-to-End Validation | pending | critical | task_09 |
