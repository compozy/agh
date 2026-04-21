---
status: completed
title: Agent Capabilities QA Execution and End-to-End Validation
type: test
complexity: critical
dependencies:
  - task_06
---

# Task 07: Agent Capabilities QA Execution and End-to-End Validation

## Overview

Execute the full QA pass for agent capabilities using the artifacts from task_06, then commit durable regression coverage in the repository's existing verification lanes. This task is the quality gate for the entire feature: it must validate real loader, session/runtime, router, and API flows, fix root-cause regressions, and leave fresh evidence under the shared QA artifact layout.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from task_06 before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/agent-capabilities` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT QA CONTRACT - use repository-defined gates and real runtime/message flows as final proof, not one-off scripts
- DO NOT WEAKEN TESTS TO GET GREEN - fix production code or configuration at the source, then rerun the narrow reproduction and full gates
- GREENFIELD: a validacao final precisa provar o comportamento real de loader + network discovery, nao apenas snapshots de structs em testes isolados
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/agent-capabilities`
- MUST consume `.compozy/tasks/agent-capabilities/qa/test-plans/` and `.compozy/tasks/agent-capabilities/qa/test-cases/` from task_06 as the execution matrix seed
- MUST execute the repository verification contract plus real capability flows that exercise local catalog loading, session join plumbing, brief discovery in greets/peer listings, and rich discovery via `whois`
- MUST capture fresh QA evidence in `.compozy/tasks/agent-capabilities/qa/verification-report.md` and store issue files or screenshots under the same artifact root when applicable
- MUST fix root-cause regressions and add or update the narrowest durable regression coverage for every discovered bug
- MUST rerun the repository verification gates after the last fix, including the integration lanes that now prove capability behavior
- SHOULD validate API-visible peer payloads in addition to runtime/router flows when the changed branch exposes them
</requirements>

## Subtasks
- [x] 7.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/agent-capabilities` and derive the execution matrix from task_06 artifacts
- [x] 7.2 Run the baseline repository verification gate and establish the pre-execution health state
- [x] 7.3 Execute real loader, session join, brief discovery, rich discovery, and empty/oversized edge-case scenarios through supported repo surfaces
- [x] 7.4 Fix root-cause regressions, add matching regression coverage, and rerun impacted scenarios
- [x] 7.5 Rerun final verification gates and publish `.compozy/tasks/agent-capabilities/qa/verification-report.md`

## Implementation Details

See the TechSpec "Testing Approach" plus the QA artifacts from task_06. The key constraint is that capability QA must prove behavior through the repo's real seams: `internal/config` loader behavior, session-to-network join plumbing, router `greet`/`whois` flows, and API payload conversion where applicable.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` - required workflow for execution matrix discovery, evidence capture, and verification reporting
- `.agents/skills/qa-execution/scripts/discover-project-contract.py` - canonical project-contract discovery entrypoint required by `/qa-execution`
- `Makefile` - repository-defined verification gate that must be rerun after the last fix
- `internal/config/agent_test.go` - likely regression destination for loader and agent-directory bugs found during QA
- `internal/network/router_test.go` - likely regression destination for brief/rich discovery and filtering bugs
- `internal/network/manager_test.go` - likely regression destination for join/regreet behavior bugs
- `internal/api/core/network_test.go` - likely regression destination for API-visible peer-card projection bugs

### Dependent Files
- `.compozy/tasks/agent-capabilities/qa/verification-report.md` - final QA evidence produced by `/qa-execution`
- `.compozy/tasks/agent-capabilities/qa/issues/BUG-*.md` - structured bug reports for failures discovered during execution
- `.compozy/tasks/agent-capabilities/qa/screenshots/` - only when visual or API-console evidence is useful during execution
- `internal/network/router_integration_test.go` - natural home for richer end-to-end `whois` and presence regressions
- `internal/session/manager_integration_test.go` - natural home for session activation and join-plumbing regressions

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - QA execution must prove explicit local catalogs drive advertised capabilities
- [ADR-002: Dual Storage Modes Without Merge](adrs/adr-002.md) - QA execution must prove the supported layouts and mixed-layout rejections
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - QA execution must prove both brief and rich discovery reflect the structured capability model

## Deliverables
- Fresh `.compozy/tasks/agent-capabilities/qa/verification-report.md` produced by `/qa-execution`
- QA evidence covering loader correctness, runtime join plumbing, brief discovery, rich discovery, empty-catalog behavior, and response-size guard behavior **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh issue files and supplementary evidence under `.compozy/tasks/agent-capabilities/qa/` **(REQUIRED)**
- Passing repository verification gates after the final QA fix set **(REQUIRED)**
- Fresh evidence that integration lanes, not only isolated unit tests, prove the capability flow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Any bug found in capability loading gains a narrow regression that covers the exact failing layout or validation rule instead of only a broad parser happy path
  - [x] Any bug found in join plumbing gains a regression that proves the specific session-to-network payload invariant that failed
  - [x] Any bug found in brief or rich discovery gains the narrowest regression in `internal/network` or `internal/api/core` that proves the exact ext or filtering behavior
  - [x] Any bug found in the oversized-response guard gains a regression that proves invalid responses are not emitted
- Integration tests:
  - [x] Real agent-directory fixtures prove capability catalogs load through the runtime path rather than only through isolated helper calls
  - [x] Real session activation or manager flows prove local peers join with capability-aware peer cards
  - [x] Real `greet` or peer-listing flows prove `peer_card.capabilities` and `agh.capabilities_brief` remain aligned
  - [x] Real `whois` flows prove explicit rich discovery, `capability_ids` filtering, no-catalog behavior, and unknown-ID behavior end to end
  - [x] API-visible peer payloads remain correct after the runtime/router flows complete
  - [x] `make verify` passes from a clean rerun after the final QA fix set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-execution` has been run explicitly with artifacts stored under `.compozy/tasks/agent-capabilities/qa/`
- The feature has fresh runtime evidence proving local authoring, brief discovery, and rich discovery end to end
