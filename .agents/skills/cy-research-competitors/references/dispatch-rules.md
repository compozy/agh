# Dispatch Rules

Subagents launched by this skill (`cy-researcher`, defined at `.claude/agents/cy-researcher.md`) operate under a strict **scoped-write** contract — exactly one `Write` to the named target path, every other action read-only. The rules below MUST be embedded in every subagent prompt verbatim.

## Scoped-Write Contract

1. The parent prompt MUST name two things:
   - The competitor source directory: `.resources/<name>/`.
   - The exact target analysis file path: `.compozy/tasks/<slug>/analysis/analysis_<name>.md`.
   If either is missing or ambiguous, the subagent returns a clarification request and writes nothing.
2. The subagent MAY call `Write` exactly once, and only at the target path the parent named.
3. The subagent MUST NOT call `Edit`. MUST NOT call `Write` against any other path. MUST NOT create directories outside the named analysis directory.
4. The subagent reads only under `.resources/<name>/` and `~/dev/knowledge/<name>/` if it exists.
5. The subagent MUST NOT run state-mutating shell commands: no `git`, `make`, `bun`, `npm`, `pnpm`, `mv`, `rm`, `cp` of non-trivial trees, `>`, `>>`, or any command that touches the working tree outside `.compozy/tasks/<slug>/analysis/`.
6. If the subagent encounters a file that requires interpretation by another tool (compiled binary, encrypted blob), it records a note in the **Open Questions** section and continues.

## Tool Restrictions

- **Allowed:** `Read`, `Grep`, `Glob`, `Bash` for read-only operations (e.g., `wc -l`, `find`, `head`, `cat`, `ls`, `file`, `rg`), `Write` (exactly once, only at the named target path).
- **Forbidden:** `Edit` anywhere; `Write` to any path other than the named target; `Bash` commands that mutate state (`rm`, `mv`, `>`, `>>`, `git`, `make`, package managers).

## Parent Responsibilities

- The parent agent MUST ensure `.compozy/tasks/<slug>/analysis/` exists before dispatch (the subagent will refuse to write into a missing directory rather than creating it).
- The parent agent MUST set `subagent_type: cy-researcher` on every Agent dispatch in the research round.
- The parent agent MUST embed both names — competitor directory and target file path — explicitly in the subagent prompt.

## Model Selection

- Omit explicit model selection unless the user explicitly requests it.
- If explicit model selection is requested and supported, use `gpt-5.4-mini` with `reasoning_effort=high` for breadth across many files.
- Use `gpt-5.4` with `reasoning_effort=xhigh` for architecturally complex competitors (Hermes process registry, OpenClaw daemon kernel, AGH-network research) where shallow reading would miss invariants.

## Parallelism

- All subagents in a research round dispatch in the same parallel batch. Do not stagger.
- Wait for every subagent to complete before verification. A partial set is unacceptable.

## Output Validation

Each subagent writes a file containing all seven sections from `assets/analysis-template.md` (Overview, Mechanisms/Patterns, Relevant Code Paths, Transferable Patterns, Risks/Mismatches, Open Questions, Evidence). After dispatch the parent:

1. Lists `.compozy/tasks/<slug>/analysis/` and confirms one file per dispatched competitor.
2. Re-reads each file to confirm all seven sections are present.
3. Sample-checks at least one cited path per file with `Read` to confirm evidence is real, not fabricated.
4. If any section is empty or any cited path is fake, re-dispatches the offending subagent with the schema and a request to fill the gap. The parent never authors the missing content — the subagent owns the write.

## Failure Handling

- If a subagent crashes or returns malformed output, retry once with a stricter prompt restating the scoped-write contract.
- If a subagent reports the competitor directory is empty or missing, the subagent returns a clarification request and writes nothing. The parent decides whether to author a one-paragraph stub documenting the absence. The stub is parent-authored — it is not a `cy-researcher` write.
- If a subagent violates the scoped-write contract (writes outside the named path, calls `Edit`, runs `git`/`make`/etc.), treat it as a contract violation: stop, re-read this file, and re-dispatch with the contract restated verbatim in the subagent prompt.
- Do not synthesize a missing competitor as if its analysis succeeded.
