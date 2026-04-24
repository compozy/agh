# ADR-001: Organize Hermes Fixes as a Single Hardening TechSpec

## Status

Accepted

## Date

2026-04-24

## Context

The selected Hermes issues span persistence, observability, ACP/session lifecycle, automation scheduling, MCP/tool security, memory CLI surfaces, and release/setup ergonomics. Several fixes share foundations: schema migrations, durable state, health payloads, CLI/API surfacing, process ownership, and retry/backoff behavior.

Splitting the work issue-by-issue would preserve traceability but duplicate design choices across packages. Splitting into separate TechSpecs would make review smaller but would require separate coordination for shared state and config changes.

## Decision

Create one Hermes Hardening TechSpec with domain tracks and shared foundations first:

- State, migrations, retention, and retry foundations.
- ACP/session lifecycle, failure classification, agent probes, and crash bundles.
- Automation scheduler durability and at-most-once dispatch.
- MCP auth, tool security, process registry, and per-turn interrupts.
- Memory CLI health/history with prepared hooks for future runtime context refs.
- CLI/setup/release hardening.

The TechSpec must preserve direct mapping back to the selected analysis issue IDs while sequencing shared foundations before dependent feature tracks.

## Alternatives Considered

- Separate TechSpecs per domain: easier to parallelize, but higher risk of schema/API/config drift.
- One issue-by-issue TechSpec: high traceability, but weak architectural cohesion and more duplicated work.

## Consequences

- The build order can place migrations, config, DTOs, and shared runtime packages before domain-specific implementations.
- Task decomposition must keep track boundaries clear to avoid oversized implementation tasks.
- The TechSpec must call out dependencies between tracks explicitly.

## Implementation Notes

- Use the selected issue list as a requirements map.
- Do not include excluded issues 6, 8, and 9 in the implementation scope.
- Keep each track independently testable while sharing common foundations.

## References

- `.compozy/tasks/hermes/analysis/analysis.md`
- Issues: 10, 11, 14, 15, 16, 17, 20, 21, 22, 25, 27, 28, 29, 30, 33, 34, 35, 36, 37, 39, 40, 41, 42, 43, 57, 59, 60
