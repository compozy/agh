# Seção Bento Para `packages/site`

## Summary

- Criar uma nova seção `BentoSection` com o layout visual da referência: cinco tiles assimétricos para **Runtime**, **Network**, **Bridges**, **Memory** e **Trace**.
- Inserir a nova seção na home em `packages/site/app/(home)/page.tsx` logo depois de `FeaturesSection` e antes de `SupportedAgents`, preservando o grid atual de features.
- Usar somente os assets já existentes em `packages/site/public/images/bento-illustrations`.
- Não adicionar dependências. O site já tem Next 16, React 19, Tailwind 4 e `lucide-react`; Framer Motion não está instalado e não será introduzido para esta seção.

## Key Changes

- Adicionar `packages/site/components/landing/bento-section.tsx`.
- Exportar `BentoSection` em `packages/site/components/landing/index.ts`.
- Atualizar `packages/site/app/(home)/page.tsx` para renderizar `Hero -> FeaturesSection -> BentoSection -> SupportedAgents -> ...`.
- Manter a linguagem visual AGH: dark-only, canvas `#141312`, surface `#1E1C1B`, divider `#3C3A39`, accent `#E8572A`, mono uppercase para eyebrows, sem emoji, sem roxo/azul AI, sem glow pesado, sem shadow-heavy UI.

## Bento Content

- Section header:
  - Eyebrow: `Runtime map`
  - Title: `The runtime surface in five parts.`
  - Description: `Sessions, network, memory, bridges, and traces stay in one local operating surface instead of scattering across scripts and dashboards.`
- Tiles:
  - **Runtime:** `Your agents. Under control.` Description: `One local daemon keeps every session, event, and status visible.`
  - **Network:** `Built-in network. Delegate. Deliver. Done.` Description: `Discover peers, delegate structured work, and receive receipts with trace IDs.`
  - **Bridges:** `From anywhere. Into a session.` Description: `Slack, Discord, and Telegram events become durable agent runs with replies back to the source thread.`
  - **Memory:** `Context that remembers.` Description: `Skills and workspace memory keep operational intent available across runs.`
  - **Trace:** `Every step. Always replayable.` Description: `Prompts, tool calls, delegation, receipts, and health events become replayable history.`

## Layout Decisions

- Desktop `lg+`: use a 12-column, two-row CSS grid matching the reference:
  - Runtime: `col-span-5`
  - Network: `col-span-7`
  - Bridges: `col-span-5`
  - Memory: `col-span-3`
  - Trace: `col-span-4`
- Tablet `md`: collapse to a 2-column layout while keeping Network wider when possible.
- Mobile `<768px`: strict single-column stack, `w-full`, no horizontal scroll, stable tile heights, and image object positioning tuned per asset.
- Use plain `<img>` instead of `next/image` because the site uses `output: "export"` and currently has no image optimization configuration. Set explicit `width`, `height`, `loading`, `decoding`, and meaningful `alt` text for each illustration.

## Tests And Verification

- Update `packages/site/components/landing/__tests__/landing.test.tsx`:
  - add a `BentoSection` describe block;
  - assert the five tile labels render;
  - assert the five image assets are present;
  - assert the home-facing copy is stable enough to catch accidental removal.
- Run focused site checks:
  - `cd packages/site && bun run test`
  - `cd packages/site && bun run typecheck`
  - `cd packages/site && bun run build`
- Run repository gate before completion:
  - `make verify`
- Visual verification after implementation:
  - start the site with `cd packages/site && bun run dev`;
  - inspect the landing page at desktop and mobile widths;
  - confirm the bento is non-overlapping, responsive, text remains readable, and all five PNG assets render.

## Assumptions

- The accepted integration choice is **Adicionar Nova**, so the existing `FeaturesSection` stays intact.
- The new section should live immediately after the current features grid, before the supported agents strip.
- The assets in `packages/site/public/images/bento-illustrations` are final enough for implementation; no image regeneration or editing is part of this task.
