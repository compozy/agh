---
status: pending
title: TypeScript SDK (@agh/extension-sdk)
type: frontend
complexity: high
dependencies:
  - task_06
  - task_07
---

# Task 10: TypeScript SDK (@agh/extension-sdk)

## Overview

Create the TypeScript SDK npm package that TypeScript extension authors use to build AGH extensions. The SDK provides an `Extension` class that handles the JSON-RPC 2.0 stdio transport, initialize handshake, inbound method routing, and a typed `HostAPI` client for calling back into AGH. It also ships a test harness (mock transport) and a scaffolding CLI (`npx @agh/create-extension`) with starter templates for common extension types.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `@agh/extension-sdk` npm package with TypeScript source and compiled JavaScript output
- MUST implement `Extension` class per `_examples.md` section 3 with `handle()`, `onReady()`, `start()` methods
- MUST implement `StdioTransport` class providing JSON-RPC 2.0 over stdin/stdout line-delimited framing
- MUST implement typed `HostAPI` client exposing `sessions.*`, `memory.*`, `observe.*`, `skills.*` methods matching `_protocol.md` section 5.2
- MUST implement initialize handshake per `_protocol.md` section 4 — send extension info, accept runtime grants, respond with accepted capabilities
- MUST handle bidirectional, multiplexed JSON-RPC (multiple outstanding requests in both directions)
- MUST expose a `TestHarness` class in `@agh/extension-sdk/testing` that allows unit testing extensions without spawning a real subprocess
- MUST emit extension log messages to `stderr` (stdout is reserved for protocol frames per `_protocol.md` section 1.1)
- MUST provide TypeScript type definitions matching the Go contracts (Tool, Manifest sections, hook payloads)
- MUST create a scaffolding CLI `npx @agh/create-extension` with at least two templates: `hook-subprocess` and `memory-backend`
- MUST publish as ESM with CommonJS fallback for Node.js compatibility
- MUST target Node.js 18+
</requirements>

## Subtasks
- [ ] 10.1 Initialize `@agh/extension-sdk` npm package with TypeScript configuration
- [ ] 10.2 Implement `StdioTransport` with line-delimited JSON-RPC framing and multiplexing
- [ ] 10.3 Implement `Extension` class with initialize handshake, handle(), onReady(), start()
- [ ] 10.4 Implement typed `HostAPI` client for all Host API methods
- [ ] 10.5 Implement `TestHarness` for unit testing extensions
- [ ] 10.6 Create scaffolding CLI `@agh/create-extension` with starter templates
- [ ] 10.7 Write unit tests using Vitest and the in-memory transport

## Implementation Details

New directory at `sdk/typescript/` (or similar) with `package.json`, `tsconfig.json`, source in `src/`, tests in `src/*.test.ts`.

See TechSpec "TypeScript SDK" section for the package structure. See `_examples.md` section 3 for the developer-facing API. See `_protocol.md` sections 1-5 for the wire protocol the transport must implement.

Follow AGH's existing web frontend patterns for TypeScript conventions (biome formatting, Vitest testing).

### Relevant Files
- `web/` — Existing TypeScript patterns, biome config, testing setup to mirror
- `internal/extension/host_api.go` — Source of truth for Host API method signatures (task 07)
- `internal/extension/manifest.go` — Manifest types to mirror in TypeScript (task 03)
- `internal/tools/tool.go` — Tool types to mirror (task 01)

### Dependent Files
- `.compozy/tasks/ext-architecture/task_11.md` — Reference extensions will use this SDK (task 11)

### Related ADRs
- [ADR-001: Two-Tier Extension Model](adrs/adr-001.md) — TypeScript as first-class subprocess extension language
- [ADR-004: Generalize ACP as Subprocess Extension Protocol](adrs/adr-004.md) — TypeScript transport mirrors Go subprocess transport

## Deliverables
- New `sdk/typescript/` package directory with `package.json`, `tsconfig.json`, source
- `@agh/extension-sdk` with `Extension`, `StdioTransport`, `HostAPI`, type definitions
- `@agh/extension-sdk/testing` subpath export with `TestHarness`
- `create-extension` scaffolding CLI with two templates
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration test exercising a real subprocess running an SDK-built extension **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `StdioTransport` encodes one JSON object per line
  - [ ] `StdioTransport` decodes multiple concurrent requests correctly
  - [ ] `StdioTransport` rejects messages over 10 MiB
  - [ ] `StdioTransport` ignores notifications (no id field)
  - [ ] `Extension.start()` performs initialize handshake first
  - [ ] `Extension.handle()` routes inbound requests to correct handler
  - [ ] `Extension.handle()` returns error if method not registered
  - [ ] `Extension.onReady()` fires after successful handshake
  - [ ] `HostAPI.sessions.create()` sends correct JSON-RPC request
  - [ ] `HostAPI.sessions.list()` parses response array
  - [ ] `HostAPI.memory.store()` sends correct params
  - [ ] `HostAPI.observe.events()` supports `since` parameter
  - [ ] Capability denied error throws typed error with code -32001
  - [ ] Rate limited error throws typed error with retry_after_ms
  - [ ] `TestHarness.mockHostAPI()` returns mocked responses
  - [ ] `TestHarness.loadExtension()` loads extension without spawning subprocess
  - [ ] `TestHarness.call()` invokes extension handlers directly
- Integration tests:
  - [ ] Build an SDK-based extension, spawn it as a subprocess, send real JSON-RPC, verify responses
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Package builds via `npm run build`
- `npx @agh/create-extension` scaffolds a working starter project
- `make verify` still passes in Go workspace
