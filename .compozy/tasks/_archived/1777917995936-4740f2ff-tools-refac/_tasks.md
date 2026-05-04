# Tool Registry Canonical Surface — Task List

**GREENFIELD (alpha):** extend the already-shipped `tools-registry` foundation on this branch into the final canonical AGH tool surface. Do not preserve raw-`claim_token` contracts, CLI-first guidance, opt-in discovery defaults, or stale management splits through compatibility shims.

Source artifacts: `_techspec.md`, ADR-001 through ADR-006, and `analysis/competitor-tool-surface-notes.md`.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Dynamic Policy Input Resolver and Default Discovery Overlay | completed | critical | - |
| 02 | Tools Guidance Assets and Startup Prompt Section | completed | high | task_01 |
| 03 | Coordination, Session, and Workspace Read Surfaces | completed | high | task_01 |
| 04 | Memory, Observe, and Bridge Read Surfaces | completed | high | task_01 |
| 05 | Config Mutable Tool Family | completed | high | task_01 |
| 06 | Hook Management Tool Family | completed | high | task_01, task_05 |
| 07 | Automation Tool Family | completed | high | task_01 |
| 08 | Extension Lifecycle Tool Family | completed | high | task_01 |
| 09 | Session-Bound Autonomy Tools and Claim-Token Hard Cut | completed | critical | task_01 |
| 10 | MCP Auth Status and Hosted MCP Projection Parity | completed | critical | task_01, task_03, task_04, task_05, task_06, task_07, task_08, task_09 |
| 11 | Site Docs, Generated References, and Example Alignment | completed | high | task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09, task_10 |
| 12 | QA Plan and Test Coverage | completed | high | task_11 |
| 13 | Real-Scenario QA Execution | completed | critical | task_12 |

## MVP Boundary

Tasks 01-11 implement the canonical `tools-refac` follow-up on top of the `tools-registry` foundation already shipped on this branch. Tasks 12-13 prepare and execute release-grade QA across CLI, HTTP, UDS, hosted MCP, docs, generated contracts, and agent-manageability flows.
