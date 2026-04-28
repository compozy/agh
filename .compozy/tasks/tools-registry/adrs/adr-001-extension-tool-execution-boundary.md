# ADR-001: Extension Tool Execution Boundary

## Status

Accepted

## Context

AGH's Tool Registry must support native/bundled tools and tools contributed by extensions. Competitor research shows three broad patterns:

- Hermes allows in-process plugin/tool registration, which is flexible but broadens daemon compromise risk.
- Claude Code routes plugin-contributed tools primarily through MCP, keeping third-party tools behind a protocol boundary.
- OpenClaw uses manifest-first static discovery and runtime materialization through plugin/MCP adapters.
- AGH already lets extension manifests publish static `resources.tools`, but those records are currently metadata-only.

The registry foundation must preserve AGH's extensibility goal without letting arbitrary extension code run inside the daemon process as a first implementation step.

## Decision

Extension-contributed tools will use a manifest-first, out-of-process execution boundary for the MVP.

Extensions may declare tool descriptors in their manifest so AGH can discover, index, authorize, and show them without executing extension code. A declared extension tool becomes executable only when it is backed by an approved out-of-process adapter:

- MCP server adapter,
- extension sidecar / Host API adapter,
- subprocess adapter managed by AGH runtime supervision,
- future bridge SDK adapter with equivalent process/protocol isolation.

AGH built-in tools may register in-process because they are compiled into the daemon binary and reviewed as daemon code.

Third-party extension handlers will not run in-process in the daemon for the MVP.

## Consequences

The registry must separate descriptor records from executable handles. A tool can be installed and discoverable while still being unavailable or non-executable until its backend is healthy and authorized.

Extension manifests need backend metadata or a follow-up mechanism that binds static tool declarations to sidecar/MCP/subprocess handles.

Dispatch must recheck availability, authorization, and backend health at call time. Discovery filtering is not a security boundary.

The MVP can support extension tool discovery before full extension tool invocation. This keeps the foundation extensible while avoiding a daemon plugin ABI or in-process third-party execution model.

## Rejected Alternatives

### Metadata-only extension tools

Keeping extension tools discoverable but never executable in the MVP would reduce scope, but it would not prove the registry's adapter model. It remains acceptable as a staged implementation path for individual extension backends, but not as the architectural boundary.

### Trusted in-process extension handlers

Allowing trusted or bundled extensions to register in-process handlers would be faster for first-party bundles, but it creates a second execution model and risks pressure to admit third-party handlers later. Built-ins should remain daemon code; extensions should cross a process/protocol boundary.

## Evidence

- `.compozy/tasks/tools-registry/analysis/analysis_openclaw.md`: manifest-first discovery and runtime materialization.
- `.compozy/tasks/tools-registry/analysis/analysis_claude-code.md`: plugin tools primarily flow through MCP adapters.
- `.compozy/tasks/tools-registry/analysis/analysis_hermes.md`: in-process plugin/tool registration is flexible but mismatched with AGH's desired safety model.
- `.compozy/tasks/tools-registry/analysis/analysis_agh_current_state.md`: AGH extension manifests already publish static tool metadata but have no executable registry handle.
