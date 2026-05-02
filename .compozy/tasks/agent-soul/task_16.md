---
status: completed
title: Agent Soul QA plan and regression artifacts
type: docs
complexity: high
dependencies:
    - task_01
    - task_02
    - task_03
    - task_04
    - task_05
    - task_06
    - task_07
    - task_08
    - task_09
    - task_10
    - task_11
    - task_12
    - task_13
    - task_14
    - task_15
---

# Task 16: Agent Soul QA plan and regression artifacts

<!-- compozy-qa-workflow:qa-report -->

## Overview

Generate reusable QA planning artifacts for this workflow before live execution begins. Leave the repo with a concrete test plan, traceable execution cases, and regression-suite definitions stored under the same feature-local QA root that the execution task will consume.

<critical>
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/agent-soul` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution`; all planning and execution artifacts must live under `.compozy/tasks/agent-soul/qa/`
- FOCUS ON WHAT: define coverage, risks, automation targets, and evidence layout; do not execute validation flows or fix bugs in this task
- CLASSIFY critical flows explicitly as `E2E`, `Integration`, `Manual-only`, or `Blocked`, with reasons
</critical>

## Requirements

1. MUST use `/qa-report` with `qa-output-path=.compozy/tasks/agent-soul`.
2. MUST generate a feature-level test plan under `.compozy/tasks/agent-soul/qa/test-plans/`.
3. MUST generate execution-ready test cases under the workflow QA root.
4. MUST create at least one regression-suite document that defines smoke, targeted, and full validation priorities.
5. MUST identify P0/P1 flows that `/qa-execution` must run first, including any blocked or manual-only coverage.

## Success Criteria

- QA artifacts are complete, traceable, and ready for the QA execution task.
- The QA execution task can start without redefining scope, paths, or validation priorities.
