# Tool Registry - Task List

**GREENFIELD (alpha):** implement the final executable tool-registry model directly. Do not add compatibility aliases, descriptor-only backends, dotted tool IDs, or fallback execution paths for old state.

Source artifacts: `_techspec.md`, ADR-001 through ADR-010, `analysis/synthesis.md`, and the approved task decomposition from 2026-04-28.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Core Tool Contracts and Canonical ToolID | completed | high | - |
| 02 | Tools Config Lifecycle and Agent Grammar | completed | high | task_01 |
| 03 | Registry Indexing, Toolsets, Policy, and Projections | completed | critical | task_01, task_02 |
| 04 | Dispatch Pipeline, Hooks, Budgets, and Observability | completed | critical | task_03 |
| 05 | Native Go Built-In Providers | completed | high | task_04 |
| 06 | Extension Manifest Tool Metadata and Reconciliation | completed | high | task_03 |
| 07 | Extension Runtime Protocol and TypeScript SDK Tools | completed | critical | task_04, task_06 |
| 08 | Public Go Extension SDK | completed | critical | task_07 |
| 09 | Daemon-Owned MCP Call-Through and Auth Diagnostics | completed | critical | task_03, task_04 |
| 10 | Hosted AGH MCP Session Exposure and Approval Bridge | completed | critical | task_05, task_09 |
| 11 | API Contracts, HTTP/UDS Routes, and Codegen | completed | critical | task_05, task_07, task_09, task_10 |
| 12 | CLI Operator Commands | completed | high | task_11 |
| 13 | Web Operator Tool Diagnostics Surface | pending | high | task_11, task_12 |
| 14 | Site Documentation and Generated References | pending | high | task_13 |
| 15 | QA Plan and Test Coverage | completed | high | task_14 |
| 16 | Real-Scenario QA Execution | pending | critical | task_15 |

## MVP Boundary

Tasks 01-14 implement the Tool Registry MVP. Tasks 15-16 prepare and execute release-grade QA.
