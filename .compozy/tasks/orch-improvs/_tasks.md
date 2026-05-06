# Orchestration Improvements - Task List

**GREENFIELD (alpha):** do not preserve old local state. This task list is the traceability graph for the approved orchestration-improvements aggregate TechSpec, including completed free-mode slices and the remaining implementation, web, docs, QA, and review gates.

Normative inputs: `_techspec.md`, `_techspec_orchestration.md`, `_techspec_review_gate.md`, ADR-001 through ADR-010, the peer-review artifacts under `qa/`, the workflow memory under `memory/`, and `state.yaml`. Historical completed tasks map to the existing `state.yaml.progress.checklist` entries; pending tasks define the remaining executable work.

## MVP Boundary

Tasks 01-20 are completed backend and planning slices already recorded by `cy-codex-loop` memory/state. Tasks 21-30 are the remaining MVP implementation, frontend, docs, and memory-lessons work. Tasks 31-32 are the mandatory QA pair. Post-MVP excludes generic notification fan-out, bridge-owned review verdicts, multi-reviewer quorum, private frontend plugin SDKs, and compatibility shims for any old task/profile/review state.

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | Task Orchestration Config Defaults and Validation | completed | medium | - |
| 02 | Task Orchestration GlobalDB Schema Foundation | completed | high | task_01 |
| 03 | Fresh/Migrated Schema Drift Guard for Execution Profiles | completed | medium | task_02 |
| 04 | Review-Gate GlobalDB Schema Foundation | completed | high | task_02 |
| 05 | Current Run Projection and Transition Invariants | completed | high | task_02 |
| 06 | Durable Notification Cursor Primitive | completed | high | task_02 |
| 07 | Bridge Task Subscription Store and Terminal Notifier | completed | high | task_06 |
| 08 | Task Execution Profile Domain, Service Authority, and Store CRUD | completed | high | task_02, task_03 |
| 09 | Run Review Request, Binding, and Review Store Authority | completed | high | task_04, task_08 |
| 10 | Run Review Verdicts, Continuation Runs, and Review Events | completed | critical | task_09 |
| 11 | Profile-Based Claim Eligibility Filtering | completed | high | task_08 |
| 12 | Worker Agent, Provider, and Model Session Selection | completed | high | task_08 |
| 13 | Sandbox Profile Runtime Application | completed | high | task_08, task_12 |
| 14 | Bundled Orchestration and Reviewer Skills | completed | medium | task_08, task_09 |
| 15 | Reviewer-Bound Native `submit_run_review` Tool | completed | high | task_09, task_10, task_14 |
| 16 | Native Task Execution Profile Tools | completed | medium | task_08 |
| 17 | Native Run Review Request/List/Show Tools | completed | medium | task_09 |
| 18 | Execution Profile HTTP, UDS, CLI, and OpenAPI Surfaces | completed | high | task_16 |
| 19 | Run Review HTTP, UDS, CLI, and OpenAPI Surfaces | completed | high | task_15, task_17 |
| 20 | Formal Remaining-Work Cross-Walk and QA Tail | completed | medium | task_19 |
| 21 | Bridge Notification Transport Consolidation and Spec Alignment | completed | high | task_07, task_18, task_20 |
| 22 | ReviewRouter Runtime Routing and Reviewer Binding Orchestration | completed | critical | task_10, task_13, task_14, task_15, task_17, task_19 |
| 23 | Task Context Bundle and Review Continuation Redaction | completed | high | task_10, task_12, task_22 |
| 24 | Latest Event Sequence and Cursor-Seeded Task SSE | completed | high | task_05, task_23 |
| 25 | Notification Cursor Diagnostics and Bridge Subscription Lifecycle | completed | high | task_06, task_07, task_21, task_24 |
| 26 | Web Data Layer for Orchestration, Review, and Notifications | completed | high | task_18, task_19, task_23, task_24, task_25 |
| 27 | Web UI for Execution Profiles, Review State, and Notification Diagnostics | completed | critical | task_26 |
| 28 | Site Docs for Orchestration Profiles and Configuration | completed | high | task_18, task_27 |
| 29 | Site Docs for Review Gate, Bundled Skills, and Notification Cursors | completed | high | task_19, task_25, task_27 |
| 30 | Durable Lessons and Glossary Alignment | completed | medium | task_28, task_29 |
| 31 | QA Plan and Test Coverage | completed | high | task_30 |
| 32 | Real-Scenario QA Execution | completed | critical | task_31 |

## Post-QA Loop Gate

After task_32 completes, continue `cy-codex-loop` Phase D with `cy-review-round` and `cy-fix-reviews` until `state.yaml.coderabbit.rounds_clean_streak >= 3`. Do not set `progress.deliverables_complete=true` until task_01 through task_32 are complete, `qa.report_done=true`, `qa.execution_done=true`, the CodeRabbit clean streak is satisfied, the tracked `.pyc` artifact is resolved with explicit user permission, and a final `make verify` passes.
