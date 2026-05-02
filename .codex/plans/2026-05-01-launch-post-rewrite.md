# Rewrite The AGH Launch Post

> **Superseded 2026-05-01** — the launch post was rewritten in place during the
> hero relock. Final shipped frontmatter is title `"Introducing AGH: an open
workplace for AI agents"` and description `"AGH gives the agent CLIs you
already use a place to work as a team — finding each other, sharing
capabilities, and closing work with receipts on agh-network/v0."` (not the
> "local-first operating system for AI agents" framing recorded below). Body
> reframed around the workplace metaphor; technical "what ships today" lists,
> commands, and alpha framing preserved per `COPY.md` §7/§8. Ship plan below
> kept as historical record of the original direction.

## Summary

Rewrite `packages/site/content/blog/posts/introducing-agh-the-first-agent-network-protocol.mdx` in place as the canonical AGH launch article. Keep the existing slug/file path, but shift the article from “mostly protocol manifesto” to “complete project launch”: broad enough for non-developer readers, precise enough for infra/agent developers, and structured like a high-quality Vercel launch post.

Primary CTA: star GitHub. Secondary CTAs: install AGH, read Runtime docs, and read Protocol docs.

## Key Changes

- Replace the post body in place; do not create a second draft and do not rename the slug.
- Update frontmatter:
  - `title`: `Introducing AGH: the local-first operating system for AI agents`
  - `description`: `AGH is a local-first agent operating system: one daemon for durable Claude Code, Codex, and Gemini sessions; one open protocol for agent-to-agent coordination; one control plane humans and agents can operate.`
  - keep `date: 2026-04-30`
  - add `updated: 2026-05-01`
  - set `category: runtime`
  - keep launch/protocol/runtime tags and add `agent-operating-system`
  - include all seven `kinds`, adding `say`
  - keep `featured: true`
- Keep the existing custom MDX components already available through the blog renderer: `Callout`, `WireCard`, `KindChip`, and `MonoBadge`.
- Fix the task-run command example so it matches generated CLI docs: use run IDs for `agh task heartbeat` and `agh task complete`, not raw claim tokens.
- Keep claims truthful to current docs: AGH is alpha; `agh-network/v0` is usable today; verified identity, hardened federation, and v1 trust/conformance are roadmap/RFC work.

## Article Structure

- Open with a direct launch: AGH is a local-first agent operating system that runs real ACP-compatible agent CLIs as durable work and exposes an open agent network protocol.
- Explain why agents need an operating system: durability, replay, permissions, memory, context, automation, and operational visibility.
- Explain what AGH gives readers: durable sessions, inspectable/resumable work, and open agent-to-agent coordination.
- Add an “Everything AGH ships today” section grouped by outcome: runtime/daemon, sessions, agents, workspaces, capabilities, skills, memory, tools/MCP, automation, autonomy, network, bridges, hooks, extensions, sandbox, configuration/operations, and public surfaces.
- Preserve the strongest AGH Network explanation as one section, including all seven message kinds and the MCP vs AGH Network distinction.
- Add an “agent-manageable by design” section that shows CLI/API/tool parity as a core product differentiator.
- Add an honest alpha section with works today, not yet, and do not use for boundaries.
- End with GitHub star as primary CTA and install/docs as secondary actions.

## Test Plan

- Run `cd packages/site && bun run content:generate`.
- Run `cd packages/site && bun run typecheck`.
- Run `cd packages/site && bun run test`.
- Run `cd packages/site && bun run build`.
- Run full repo `make verify` before considering the implementation complete, unless an unrelated pre-existing worktree failure blocks it.
- Manually inspect the rendered/post source for all seven kind chips, truthful alpha language, corrected task-run examples, valid internal links, and final GitHub CTA.

## Assumptions

- The implementation is content-only: no blog layout, Open Graph, RSS, image, or component changes are included.
- The post stays in English.
- The article may be longer and more complete than the current draft, but it should remain a product launch article rather than reference documentation.
- QMD semantic search was unavailable because the local QMD vector path failed with `SQLiteError: no such module: vec0`; planning relied on QMD lexical results, local RFC/docs inspection, and Vercel blog research.
