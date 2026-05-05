# Goal Header Template

`cy-codex-loop` is invoked manually. The most common invocation pattern
is to paste the codex-loop activation header at the top of the prompt
plus an explicit reference to the skill. The plugin itself is **not**
modified; nothing in `~/.codex/codex-loop/config.toml` needs to change.

## Canonical header (copy-paste)

For a feature with slug `<slug>` whose techspec lives at
`.compozy/tasks/<slug>/_techspec.md`:

```text
[[CODEX_LOOP name="<slug>" goal="ship <slug> end-to-end via cy-codex-loop: every iteration runs .agents/skills/cy-codex-loop/scripts/detect-phase.py and executes the printed action; finish only when 3 consecutive coderabbit rounds are clean and make verify is PASS"]]

Use the cy-codex-loop skill at .agents/skills/cy-codex-loop/SKILL.md.
The skill is a state machine — one iteration per Stop. Slug: <slug>.
```

The `goal=` text becomes `state.yaml.goal_signature` and is shown to the
goal-check confirmation prompt as the success criterion. Keep it
specific (mentions slug, mentions the 3-clean-rounds + verify gate) so
the verdict is grounded.

## Tunable confirm/interpret models

The plugin's confirm/interpret defaults work for this skill, but for
long features (>30 iterations) raising the confirm reasoning effort
yields better verdict quality:

```text
[[CODEX_LOOP name="<slug>" goal="..." confirm_model="gpt-5.5" confirm_reasoning_effort="xhigh"]]
```

This does not affect the skill itself — only the plugin's verdict
quality.

## Invoking without the plugin (manual run)

The skill can also be exercised manually inside a single Claude Code or
Codex session — the codex-loop-plugin restart flow is convenient but
optional. Manual invocation:

```
Activate the cy-codex-loop skill at .agents/skills/cy-codex-loop/SKILL.md
for slug <slug>. Run one iteration, then stop.
```

The agent will run `.agents/skills/cy-codex-loop/scripts/detect-phase.py`, take one action, update state +
memory, and stop. The user re-invokes for each subsequent iteration.

## When NOT to use the goal header

Do not paste the `[[CODEX_LOOP ...]]` header for the bootstrap iteration
if `_techspec.md` does not exist yet — `cy-codex-loop` will refuse and
record a blocker. Author the techspec first via `cy-create-techspec`,
then start the loop.
