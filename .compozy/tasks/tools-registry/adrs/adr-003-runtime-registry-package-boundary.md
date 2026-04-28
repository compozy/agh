# ADR-003: Runtime Registry Package Boundary

## Status

Accepted

## Context

AGH already has `internal/tools`, but it currently defines only metadata records and a list-only provider interface. The Tool Registry foundation needs runtime concerns: executable handles, provider adapters, availability, policy projection, dispatch, hooks, telemetry, result budgeting, and hosted MCP exposure.

The registry also needs to compose with skills for agent-facing discovery operations such as `agh__tool_search`, `agh__skill_list`, and `agh__skill_view`. That creates a package-boundary question: should the runtime registry live in `internal/tools`, in a new broad `internal/catalog`, or in daemon orchestration code?

## Decision

`internal/tools` will own the runtime Tool Registry contracts and execution path.

The package will evolve from metadata-only records into the home for:

- tool descriptors and source/provenance types,
- executable handles and providers,
- availability and reason codes,
- policy projection for tool views,
- central `Registry.Call` dispatch,
- result normalization and result-budget metadata,
- provider adapters for `native_go`, `extension_host`, `mcp`, and future bridges.

A thin `internal/catalog` facade may compose `internal/tools` and `internal/skills` for cross-domain list/search/view surfaces and for AGH-native tools such as `agh__tool_search` and `agh__skill_view`.

The daemon composition root will wire providers and dependencies, but it will not own registry logic.

## Consequences

Tool execution remains in the tool domain instead of a broad catalog domain. This keeps dispatch, policy, availability, and telemetry testable without booting the daemon.

`internal/catalog` stays small and read-oriented. It coordinates cross-domain discovery and progressive disclosure, but does not become a second execution engine.

Existing `internal/tools.Tool` resource compatibility must be handled through clear type splits rather than by adding runtime function fields to resource records. The TechSpec must distinguish cold resource specs from runtime descriptors and handles:

- cold `tool` resources describe desired state and provenance;
- runtime `Descriptor` values normalize policy, schema, source, backend, and risk metadata;
- runtime `Handle` values execute through Go function closures, extension subprocess calls, or MCP client calls.

Daemon boot must register providers explicitly, following AGH's composition-root discipline.

`internal/tools` must not import `internal/extension`, `internal/mcp`, `internal/api/*`, `internal/cli`, or `internal/daemon`. Those adapters are injected by daemon wiring behind interfaces defined in `internal/tools`.

## Rejected Alternatives

### New `internal/catalog` owns everything

This would centralize skills and tools, but it would also mix skill content loading, tool dispatch, extension adapter logic, search, and policy into one large package too early.

### Daemon-owned registry logic

Keeping registry logic inside daemon orchestration would reduce package churn, but it would make dispatch harder to test and would couple tool semantics to boot wiring.

## Evidence

- `.compozy/tasks/tools-registry/analysis/analysis_agh_current_state.md`: `internal/tools` is currently metadata-only while daemon boot already owns resource projection.
- `.compozy/tasks/tools-registry/analysis/analysis_goclaw.md`: GoClaw keeps executable tool contracts and policy close to the tools package.
- `.compozy/tasks/tools-registry/analysis/synthesis.md`: recommends `internal/tools` for runtime contracts and a thin catalog facade for cross-skill/tool search.
- `internal/CLAUDE.md`: AGH favors interfaces where consumed and composition-root wiring over daemon package logic accumulation.
