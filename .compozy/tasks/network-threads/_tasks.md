# Network Threads - Task List

## Tasks

| # | Title | Status | Complexity | Dependencies |
|---|-------|--------|------------|--------------|
| 01 | RFC, Glossary, and Protocol Hard Cut | completed | high | none |
| 02 | Network Wire Model, Validation, and Hard-Cut Symbol Deletion | completed | critical | task_01 |
| 03 | Work Lifecycle and Direct-Room Identity Primitives | completed | critical | task_02 |
| 04 | SQLite Conversation Schema and Store DTO Foundation | completed | critical | task_02, task_03 |
| 05 | Conversation Persistence, Queries, Summaries, and Audit Writes | completed | critical | task_04 |
| 06 | Runtime Routing, Delivery Wrappers, and Task Ingress | completed | critical | task_03, task_05 |
| 07 | Network Hooks, Status Counters, and Observability | completed | critical | task_05, task_06 |
| 08 | Public Contracts, HTTP/UDS Parity, and Codegen | completed | critical | task_05, task_06 |
| 09 | CLI Network Thread, Direct, Work, and Send Commands | completed | high | task_08 |
| 10 | Native Agent Tools and Hosted Tool Schemas | completed | high | task_08 |
| 11 | Extension Host API, SDK, and Bridge Mapping | completed | high | task_08 |
| 12 | Agent Prompt Wrappers and Bundled Network Skill | completed | high | task_06, task_09, task_10 |
| 13 | Web Network Shell, Routes, Channel-Pivot IA & Query Isolation | completed | critical | task_08 |
| 14 | Web Message Row, Timeline, Thread Overlay & Author Group Collapse | completed | critical | task_13 |
| 15 | Web Composer, Work Surfacing, Empty/Error States & Realtime Polling | completed | critical | task_14 |
| 16 | Site, Runtime Docs, Examples, and CLI Reference Co-Ship | completed | high | task_01, task_09, task_10, task_11, task_12, task_13, task_14, task_15 |
| 17 | E2E Harness and Fixture Alignment | completed | high | task_06, task_08, task_09, task_10, task_11, task_12, task_13, task_14, task_15 |
| 18 | Network Threads QA plan and regression artifacts | completed | high | task_01, task_02, task_03, task_04, task_05, task_06, task_07, task_08, task_09, task_10, task_11, task_12, task_13, task_14, task_15, task_16, task_17 |
| 19 | Network Threads QA execution and operator-flow validation | completed | critical | task_18 |
