# ADR-001: Agent Tool Surface Is Tool-First With Default Discovery

## Status

Accepted

## Date

2026-04-29

## Context

The `tools-registry` foundation is already present on this branch. It
established AGH-native tool descriptors, CLI commands, hosted MCP exposure, and
the initial built-in/toolset MVP:

- `agh__bootstrap`
- `agh__catalog`
- `agh__coordination`
- `agh__tasks`

What it did not settle is the final agent-facing surface. Some AGH
capabilities are available as built-in tools while others remain reachable only
through `agh ...` CLI calls. The startup prompt exposes skills and network
guidance, but it does not expose any tool-specific guidance. The skills catalog
still teaches `agh skill view` as the normal loading path. Agents therefore do
not yet get a canonical textual instruction to discover or prefer AGH-native
tools.

The repo's product posture is explicit: AGH features are incomplete if they are
not agent-manageable through structured surfaces. Competitor references show the
same direction. Claude Code has a startup tools section and tool-search
guidance; Hermes uses toolsets and tool/runtime docs; OpenClaw has a dedicated
tooling block plus a tools guide. In every case, internal runtime capabilities
are framed as tools first, with CLI or slash commands as management surfaces.

The user explicitly chose a `tools-first by convention` posture and rejected any
sandbox- or driver-level attempt to block shell access to the `agh` binary. Bash
behavior belongs to each ACP driver, not to this TechSpec.

## Decision

AGH's final agent surface is tool-first by convention:

1. AGH capabilities that an agent should use during runtime are exposed as
   dedicated tools, not primarily via `agh ...` shell commands.
2. Shell access may still exist because ACP drivers may provide native Bash or
   shell tools, but AGH guidance and structured surfaces must steer agents
   toward dedicated tools for AGH internals.
3. Every agent receives `agh__bootstrap` and `agh__catalog` by default unless
   effective policy narrows or denies them.
4. The startup prompt gains a dedicated `tools` section, and bundled skills gain
   a dedicated `agh-tools-guide` that teaches the discovery and invocation loop.
5. CLI, HTTP, and UDS remain first-class management surfaces for operators and
   for agent workflows that need structured non-tool management paths.

## Alternatives Considered

### Alternative 1: Permanent hybrid surface

- **Description**: Keep half of AGH internals as tools and the rest as CLI/Bash
  paths, with documentation explaining when to use each.
- **Pros**: Lower implementation cost in the short term; fewer new tool
  handlers.
- **Cons**: Keeps two competing agent paths for the same runtime; weakens
  auditing and policy clarity; invites inconsistent agent behavior.
- **Why rejected**: It preserves the ambiguity this redesign is meant to remove.

### Alternative 2: Tool-first with structural shell blocking

- **Description**: Make AGH tools canonical and block the `agh` binary from
  shell execution inside agent sandboxes.
- **Pros**: Strong enforcement; reduces accidental CLI bypass.
- **Cons**: Pushes sandbox/driver constraints into a TechSpec whose scope is the
  canonical runtime surface, not ACP driver behavior.
- **Why rejected**: The user explicitly rejected making shell blocking part of
  this design.

## Consequences

### Positive

- Agent behavior becomes more predictable and easier to teach.
- Discovery and invocation can be documented once and reused across providers.
- Structured tools become the canonical place for policy, redaction, and
  observability.

### Negative

- AGH must cover more capabilities as dedicated tools to avoid leaving obvious
- CLI-only gaps.
- Startup prompt and bundled-skill maintenance becomes part of the tool surface.

### Risks

- Agents may still call `agh ...` through shell when a native shell tool exists.
  Mitigation: the prompt section and bundled guide make the preferred path
  explicit, and core AGH workflows gain dedicated tools so the structured path
  is easier than the shell path.

## Implementation Notes

- Add a new `HarnessPromptSectionTools` startup section.
- Add bundled `agh-tools-guide` content under `internal/skills/bundled/skills/`.
- Default discovery toolsets should be applied during effective policy
  resolution, not by duplicating them into every agent definition file.
- Replace the current catalog guidance in `internal/skills/catalog.go` and the
  shipped `agh-agent-setup` examples so tool-first discovery is explicit.

## References

- `.compozy/tasks/tools-registry/_techspec.md`
- `.compozy/tasks/tools-registry/adrs/adr-004-mvp-native-tool-scope.md`
- `.compozy/tasks/tools-registry/adrs/adr-006-tool-visibility-by-surface.md`
- `docs/_memory/standing_directives.md` (SD-011)
- `.resources/claude-code/constants/prompts.ts`
- `.resources/hermes/agent/prompt_builder.py`
- `.resources/openclaw/src/agents/system-prompt.ts`
