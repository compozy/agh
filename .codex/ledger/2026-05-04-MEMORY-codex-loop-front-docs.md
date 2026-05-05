Goal (incl. success criteria):

- Update `.agents/skills/cy-codex-loop` so frontend/docs work is always orchestrated through Compozy -> Claude Opus instead of being executed directly by the local Codex loop session.
- Success: the skill instructions clearly define when delegation applies, how to dispatch `compozy exec --ide claude --model opus`, and how completion/verify evidence gates state updates.

Constraints/Assumptions:

- Do not run destructive git commands.
- Conversation in Brazilian Portuguese; artifacts stay in English.
- Scope is the `cy-codex-loop` skill and its supporting references/checklists unless a helper is clearly required.
- Follow `compozy`, `skill-best-practices`, and `agent-md-refactor` guidance for this turn.

Key decisions:

- Treat task frontmatter `type:` as the primary classifier for `frontend` and `docs` task delegation.
- Keep backend/mixed work local unless the slice is explicitly limited to frontend/docs surfaces.
- Use documentation-first enforcement (skill + references/checklist) unless a helper script becomes necessary.

State:

- Completed.

Done:

- Read root instructions and scoped `AGENTS.md` requirements.
- Scanned `.codex/ledger/` for cross-agent awareness and read the prior `2026-05-05-MEMORY-fix-cy-codex-loop.md` ledger.
- Read `.agents/skills/cy-codex-loop/SKILL.md`, `.agents/skills/cy-codex-loop/references/checklist.md`, `.agents/skills/cy-codex-loop/references/phase-transitions.md`.
- Read `.agents/skills/compozy/SKILL.md`, `.agents/skills/skill-best-practices/SKILL.md`, and `.agents/skills/agent-md-refactor/SKILL.md`.
- Confirmed existing task files use frontmatter `type: frontend` and `type: docs`, which is suitable for deterministic delegation rules.
- Added a mandatory frontend/docs delegation lane to `.agents/skills/cy-codex-loop/SKILL.md`.
- Added `.agents/skills/cy-codex-loop/references/frontend-docs-delegation.md` with classification, dispatch contract, prompt requirements, and completion gate rules.
- Updated `references/phase-transitions.md` so Phase B task/free execution explicitly branches to `compozy exec --ide claude --model opus` for frontend/docs work.
- Updated `references/checklist.md` so self-audit enforces orchestration mode and delegated verify evidence.
- Ran `python3 .agents/skills/skill-best-practices/scripts/validate-metadata.py ...` successfully for the updated skill description.
- Ran `make verify` successfully; the full monorepo gate passed.

Now:

- Prepare final report.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.agents/skills/cy-codex-loop/SKILL.md`
- `.agents/skills/cy-codex-loop/references/checklist.md`
- `.agents/skills/cy-codex-loop/references/phase-transitions.md`
- `.agents/skills/compozy/SKILL.md`
- `.agents/skills/skill-best-practices/SKILL.md`
- `.agents/skills/agent-md-refactor/SKILL.md`
