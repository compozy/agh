# ADR-005: Prioritize Memory Health and History Before Runtime Context References

## Status

Accepted

## Date

2026-04-24

## Context

Selected Hermes memory issues include CLI memory health, memory history, provider hooks, and context references such as `@file`, `@folder`, `@git`, and `@url`. The current memory package already has health statistics and a memory operation log, but the CLI does not expose those surfaces directly.

Runtime context references and provider hooks touch prompt assembly and pre-turn augmentation. That work is broader and can change prompt behavior, token budgets, and sensitive-path handling.

## Decision

Scope this Hermes TechSpec to implement memory health/history CLI surfaces and prepare stable interfaces for future context references and provider hooks.

The implementation scope includes:

- `agh memory health`.
- `agh memory history`.
- API/CLI wiring over existing memory health and operation-log data.
- Interface definitions and package seams for future `ContextRefResolver` and memory provider lifecycle hooks.

Runtime prompt integration for `@file`, `@folder`, `@git`, `@url`, token budgeting, and pre-turn hook execution is explicitly deferred to a follow-up implementation phase.

## Alternatives Considered

- Full memory context runtime now. This resolves more of the analysis immediately but increases behavior risk in prompt assembly.
- Implement context refs in session prompt assembly outside the memory package. This is smaller locally but fragments memory semantics.

## Consequences

- Issues 34 and 60 can be completed directly.
- Issues 33 and 35 will be designed with interfaces and seams, but full runtime behavior remains follow-up work.
- The TechSpec must clearly mark deferred runtime integration to avoid overstating completion.

## Implementation Notes

- Reuse existing memory health stats and `memory_operation_log` where possible.
- Keep CLI output consistent with existing `observe events` and `memory` command patterns.
- Define interfaces in the consuming package or memory boundary without wiring them into prompt execution yet.

## References

- `.compozy/tasks/hermes/analysis/analysis_memory_context.md`
- Issues: 33, 34, 35, 60
