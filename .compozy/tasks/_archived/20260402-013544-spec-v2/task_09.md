---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_01
    - task_05
---

# Task 9: Prompt Assembler

## Overview
Implement the three-layer prompt assembly system that composes the final system prompt for each agent from a type template (kernel behavior), role specialization (domain config from roles/*.toml), and session context (goal, domain, agent ID, workgroup).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement templates for all 5 agent types (master, worker, advisor, reviewer, researcher) per docs/spec-v2/03-agents.md
- MUST include all template sections: ROLE, COMMANDS AVAILABLE, RULES, EXAMPLES, ERROR HANDLING
- MUST inject role specialization from RoleConfig.SystemPrompt
- MUST inject session context: goal, domain, agent ID, workgroup ID/name, type, role name
- MUST include prescriptive boot instructions in the master template per docs/spec-v2/03-agents.md
- MUST restrict COMMANDS AVAILABLE per type (full for master, subset for worker, read-only for researcher)
- MUST handle empty role specialization gracefully (bootstrap agents have no role)
- MUST output a single string suitable for passing to StartOpts.SystemPrompt
</requirements>

## Subtasks
- [x] 9.1 Define templates for all 5 agent types with all required sections
- [x] 9.2 Implement role specialization injection from RoleConfig.SystemPrompt
- [x] 9.3 Implement context injection (goal, domain, agent ID, workgroup, type, role)
- [x] 9.4 Implement the assembly pipeline: template + specialization + context → final prompt
- [x] 9.5 Handle edge cases: empty role specialization, missing context fields

## Implementation Details
Refer to docs/spec-v2/03-agents.md for prompt assembly spec, template sections per type, and the assembly example. Refer to docs/spec-v2/06-cli.md for command subsets per type.

### Relevant Files
- `docs/spec-v2/03-agents.md` — prompt assembly, template sections, type behaviors
- `docs/spec-v2/06-cli.md` — CLI commands available per agent type
- `docs/spec-v2/15-examples.md` — real-world prompt assembly examples

### Dependent Files
- `internal/kernel/types.go` — RoleConfig, StartOpts types
- `internal/registry/roles.go` — role catalog for specialization lookup
- `internal/config/` — goal text, domain from session config

## Deliverables
- internal/prompt/assembler.go — assembly pipeline
- internal/prompt/templates.go — built-in templates per agent type
- internal/prompt/context.go — context injection
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Master prompt includes full CLI list, hook handling rules, boot instructions
  - [x] Worker prompt includes subset CLI, report-to-master rules
  - [x] Advisor prompt includes consultation rules, stay-updated behavior
  - [x] Reviewer prompt includes review criteria, finding-report rules
  - [x] Researcher prompt includes read-only tools, auto-destroy rule
  - [x] Context injection: goal, domain, agent ID, workgroup ID correctly placed
  - [x] Role specialization: system_prompt appended after type template
  - [x] Empty role specialization: prompt is valid without role section
  - [x] Assembly produces single string with all 3 layers
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Templates match behavior descriptions in docs/spec-v2/03-agents.md
- Prompts are prescriptive enough for LLMs to follow expected workflows
