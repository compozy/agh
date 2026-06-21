# Product

## Register

product

> Default register is `product` — the runtime operator UI in `web/` + `packages/ui`, where design SERVES the product. AGH also ships a brand surface (the agh.network marketing + Fumadocs site in `packages/site`, where design IS the product). When a task targets that site, override the register to `brand` for that task. PRODUCT.md keeps `product` as the standing default.

## Users

**Primary — Operators running agent work.** Developers and operators who run ACP-compatible agent CLIs (Claude Code, OpenClaw, Hermes) and need them as durable, inspectable sessions rather than throwaway terminal tabs. Their context: a local machine running a background daemon, often several concurrent agent sessions, where the job is to start, supervise, resume, inspect, and repair real work — and to see truthful, real-time state for sessions, tasks, memory, tools, and AGH Network activity.

**Secondary — Agent/runtime developers.** Engineers extending AGH against daemon contracts: extensions, hooks, skills, capabilities, bridges, and SDKs. They need the UI to expose the same structured surfaces the daemon exposes, not a UI-only shortcut.

**Also first-class — Agents themselves.** Agents operate AGH through structured surfaces (CLI `-o json`, HTTP/SSE, UDS, tool registry). The UI is one view over state that agents can equally drive; it is never the only path to a capability.

## Product Purpose

AGH is a local-first agent operating system. One Go daemon hosts durable, inspectable agent sessions; one operator surface (CLI, HTTP/SSE, UDS, and this web UI) serves humans and agents over the same daemon-owned state; and `agh-network/v0` lets sessions discover peers, delegate work, exchange capabilities, and close the loop with receipts.

The runtime UI's job is to give operators truthful, real-time visibility and control over sessions, tasks, memory, the tool registry, and network coordination — beyond the boundary of a single terminal tab. Success looks like: an operator can supervise concurrent agent work, understand exactly what the daemon is doing, and act on it (resume, inspect, repair, route) without ever being shown a control or metric the runtime does not actually support.

## Brand Personality

Operator-first, engineer-to-engineer, dry-confident. (COPY.md §5 is the authority; this is the design-facing distillation.)

- **Calm, not cute.** Technical, not academic. Confident, not inflated.
- Prefer nouns and mechanisms over adjectives; lead with the outcome, then the mechanism, then the proof.
- The product is usually the subject — "AGH keeps...", "The runtime exposes..." — not "we" or "you" as a sales hook.
- No emoji, no exclamation marks, no fake urgency, no fabricated stats or maturity claims.
- Emotional goal: the steadiness of a well-built control surface. The operator should feel informed and in control, never marketed to.

## Anti-references

This must NOT look like:

- **Generic SaaS dashboards.** Hero-metric templates (big number + gradient accent), identical icon-heading-text card grids, decorative glassmorphism.
- **Consumer "assistant" chat skins** that hide runtime truth behind a friendly veneer. Sessions are durable operator objects, not a cute chat bubble stream.
- **Hype copy.** `AI-powered`, `revolutionary`, `next-generation`, `supercharge`, `unleash`, `seamless`, `10x`, `cutting-edge` — banned per COPY.md §6.
- **Plausible-but-untrue UI.** Controls, metrics, or states the daemon does not actually support. When a mockup conflicts with daemon truth, daemon wins.
- **Decorative depth.** Freehand drop shadows, sketchy/hand-drawn SVG, stripe backgrounds, gradient text, side-stripe borders, over-rounded cards (32px+), eyebrow-on-every-section scaffolding. Depth comes only from the exported `--shadow-*` / hairline tokens.

## Design Principles

1. **Truthful UI over plausible UI.** Render only what the runtime supports. Daemon state is the source of truth; never invent controls or metrics to fill a layout.
2. **Operator-first, engineer-to-engineer.** Assume a technical reader; surface the operator's real job and the mechanism behind it without forcing them to decode internals before the value is clear.
3. **Extensible and agent-manageable by default.** Anything a human can do in the UI, an agent can do through CLI/HTTP/UDS over the same state. A UI-only capability is an incomplete feature.
4. **Show shipped behavior, not aspiration.** Every visible claim, control, and label maps to merged runtime mechanisms — commands, protocol objects, events, artifacts.
5. **Calm confidence through restraint.** Clarity over decoration. Hierarchy from type scale, weight, and tokenized structure (flat warm-dark surface ramp, hairlines), not ornament. No hacks, no theater.

## Accessibility & Inclusion

Target: **WCAG 2.2 AA**, measured against the warm-dark surface ramp.

- Body text ≥ 4.5:1, large text (≥18px or bold ≥14px) ≥ 3:1; placeholder text held to 4.5:1, not the muted-gray default. DESIGN.md's contrast tokens are authoritative.
- Visible, tokenized focus states and complete keyboard paths for every interactive surface.
- `prefers-reduced-motion: reduce` alternative for every animation (crossfade or instant).
- The signal palette (accent = action, success, danger, warning, info) is never the sole carrier of state — pair color with text, icon, or shape so color-blind and high-contrast users read the same meaning.
