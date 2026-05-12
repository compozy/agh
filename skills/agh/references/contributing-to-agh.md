# Contributing To AGH

## Contents

- Repository posture
- Go runtime rules
- Public surfaces
- Error and context discipline
- Tests
- Dirty worktrees
- Documentation impact

## Repository Posture

AGH is greenfield alpha. There are no production users, and backward compatibility must not reduce code quality. Prefer hard cuts over compatibility bridges:

- no aliases for renamed public concepts
- no dual fields
- no schema fallbacks for old state
- no defensive compatibility paths unless a written plan explicitly requires them

Never run destructive git commands without explicit user permission.

## Go Runtime Rules

Before editing Go runtime code, read the local repository instructions and the relevant internal/CLAUDE.md section. Core invariants:

- internal/daemon is the composition root.
- Packages must not import daemon, api, or cli.
- Interfaces are defined where consumed.
- Direct calls through interfaces beat event-bus style routing.
- Long-running work must detach from request contexts intentionally.
- Public runtime features must be agent-manageable.

Use structured errors with wrapping. Do not discard errors with \_ in production or tests.

## Public Surfaces

Any change touching a public surface should close the loop:

- contract and generated references when applicable
- HTTP/UDS handlers when state crosses daemon boundaries
- CLI command/client support
- tool or skill surfaces for agent operation
- docs
- tests

Backend-only work may declare no web/docs impact only after analysis.

## Error And Context Discipline

Use context.Context consistently for runtime operations. Do not tie detached prompts, network sends, automation jobs, or session work to a request lifetime unless cancellation is the intended behavior.

Never log or return raw claim tokens, provider secrets, OAuth codes, PKCE verifiers, MCP credentials, sandbox internals, or secret-shaped environment values.

## Tests

Every task requires a test decision. Before adding, moving, or broadening tests, name:

- the invariant
- the owning layer
- the canonical suite

Default to updating an existing canonical suite. Do not add tests that only freeze implementation details, snapshots, generated output, config shape, CSS literals, or file existence unless that artifact itself is the product contract.

When a test reveals broken production behavior, fix production code. Do not weaken the test to match the bug.

## Dirty Worktrees

Assume unrelated changes belong to the user or another agent. Do not revert them. If unrelated, ignore them. If they affect the task, read and work with them.

## Documentation Impact

Public wording follows COPY.md; visual and UI guidance follows generated DESIGN.md and token source files. Runtime docs must describe behavior the daemon actually supports, not aspirational behavior.
