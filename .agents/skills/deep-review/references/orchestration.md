# Orchestration

Cohort planning, the findings schema, the Workflow script template, and the fallback prompts. The prompts here are the single source for reviewer/sweep/skeptic behavior — the Workflow template and the Agent fallback both use them.

## Cohort rules (Step 3)

1. Group selected files by package/directory and domain: a source file, its tests, and its types travel together; a file pulled apart from its test loses its reviewer the cheapest evidence.
2. Size: ≤ 12 files **and** ≤ ~1,200 changed lines per cohort, whichever binds first. A single oversized file becomes its own cohort.
3. **Oversized-file split** — when one file alone exceeds ~1,200 changed lines (or ~2,500 total lines), divide the search across sibling reviewers: same file, disjoint slices of its manifest hunks (`hunk_scope`), one cohort per slice. Every slice reviewer reads the whole file for context but judges — and records coverage for — only its slice; the merged coverage must still account for every hunk exactly once.
4. Tag each cohort `risk: high|normal|low` — high when it touches storage/migrations, security/auth, public contracts, or concurrency; low for docs/config-only. Risk feeds reviewer emphasis, not selection.
5. Every selected file in exactly one cohort (or, when sliced, every hunk in exactly one slice cohort). `plan.json`:

```json
{ "cohorts": [
    { "id": "c01", "name": "store: task queue", "risk": "high",
      "files": ["internal/store/queue.go", "internal/store/queue_test.go"] },
    { "id": "c02a", "name": "loop/action.go — hunks 1-14", "risk": "high",
      "files": ["internal/loop/action.go"],
      "hunk_scope": { "internal/loop/action.go": [{"start": 12, "lines": 40}] } }
  ],
  "sweeps": ["contracts", "security", "migrations", "tests", "consistency", "config"] }
```

## Sweep lenses

Include a sweep when its trigger fires; each sweep sees the manifest + symbol map, not one cohort:

| Key | Trigger | Looks for |
| --- | --- | --- |
| `contracts` | exported/wire/API symbol changed contract | breaking changes, drift between spec/impl/clients, missing codegen co-ship |
| `security` | new endpoint/input path/authz surface/secret handling | injection, missing authn/authz, secret leakage, cross-tenant access |
| `migrations` | schema/migration files in diff | destructive ops, missing migration for model change, ordering/identity hazards |
| `tests` | any behavior change | new behavior without a failing-capable test, tests asserting mocks, weakened assertions |
| `consistency` | renames or repeated patterns in diff | incomplete renames, sibling paths not mirroring a fix, duplicated logic |
| `config` | config keys/flags/env vars changed | unwired or undocumented keys, dead flags, default mismatches |
| `spec-parity` | `--spec` provided (always included then) | deliverable vs each contract artifact, field by field — names, types, defaults, required flags, shapes, topologies, behaviors; named visual references demand parity evidence |

## Findings schema (single source)

```json
{
  "type": "object", "required": ["findings"],
  "properties": {
    "findings": { "type": "array", "items": {
      "type": "object",
      "required": ["file", "line", "in_diff", "category", "severity", "title", "body", "evidence"],
      "properties": {
        "file": {"type": "string"}, "line": {"type": "integer"},
        "end_line": {"type": ["integer", "null"]},
        "in_diff": {"type": "boolean"},
        "hunk": {"type": ["string", "null"]},
        "rule_id": {"type": ["string", "null"]},
        "category": {"enum": ["potential-issue", "refactor", "nitpick"]},
        "severity": {"enum": ["critical", "major", "minor", "trivial"]},
        "quick_win": {"type": "boolean"},
        "title": {"type": "string", "maxLength": 100},
        "body": {"type": "string"},
        "also_applies": {"type": "array", "items": {"type": "string"}},
        "guideline": {"type": ["string", "null"]},
        "suggestion": {"type": ["string", "null"]},
        "evidence": {"type": "array", "items": {"type": "string"}},
        "confidence": {"type": "number"}
      } } },
    "coverage": { "type": "array", "items": {
      "type": "object", "required": ["file", "hunk", "rules"],
      "properties": {
        "file": {"type": "string"}, "hunk": {"type": "string"},
        "rules": { "type": "array", "items": {
          "type": "object", "required": ["id", "verdict"],
          "properties": { "id": {"type": "string"},
            "verdict": {"enum": ["pass", "violation", "na"]} } } }
      } } }
  }
}
```

`hunk` = the manifest hunk the finding sits in (`"<start>-<end>"`), null for outside-diff findings. `rule_id` links a rule-derived finding to the registry. `coverage` is the per-hunk audit trail: every mapped rule gets a verdict, and every `violation` verdict must have a matching finding carrying that `rule_id`. Sweeps return findings only (no coverage).

Skeptic verdict schema:

```json
{ "type": "object", "required": ["refuted", "reason"],
  "properties": { "refuted": {"type": "boolean"}, "reason": {"type": "string"},
    "evidence": {"type": "array", "items": {"type": "string"}},
    "severity_override": {"enum": ["critical", "major", "minor", "trivial", null]} } }
```

## Prompt templates

Substitute `{...}` placeholders. All agents work read-only in the repo checkout.

**Cohort reviewer** — one per cohort:

```
Review cohort "{name}" (risk: {risk}) of {target}. Read-only; never modify files.
The unit of judgment is the HUNK; whole files and the symbol map are context.
1. Read {out}/context-pack.md and {skill-dir}/references/taxonomy.md in full — the
   rule registry, rule map, and taxonomy bind every finding.
2. Read EVERY cohort file in full (hunks lie without their surroundings): {files}.
   The manifest lists each file's hunks (new-side ranges); see the change itself with
   git diff {base}..{head} -- <file>.
   {when hunk_scope is set: "Judge ONLY these hunks — sibling reviewers own the
   rest of this file: {hunk_scope}. Read beyond them freely; report inside them."}
3. RULE PASS — for each hunk of each file, walk the file's mapped rules (rule map)
   and record a verdict per rule in coverage[]: "pass" (hunk complies), "violation"
   (emit a finding carrying that rule_id + the verbatim rule in guideline), or "na"
   (rule's subject does not occur in this hunk). Every hunk × mapped rule gets a
   verdict — coverage is the audit that the rubric was actually applied.
4. HUNT PASS — judge each hunk against intent and the symbol map beyond the rules:
   correctness, security, concurrency, error handling, contract drift, resource
   leaks, test gaps, sibling paths that should mirror a fix in this diff. Set hunk
   on every in-diff finding; outside-diff findings (taxonomy clauses only) set
   in_diff: false and hunk: null.
5. Before recording any finding, verify it against the checkout: read the callers/
   callees, run targeted rg/grep, trace the failing input. Record each check in
   evidence[] as "command or file:line → what it showed". A finding without concrete
   failure mode (or concrete improvement) is not a finding.
6. Apply the taxonomy suppression rules; linter-covered findings are void
   (lanes that ran are listed in the context pack).
7. Severity per taxonomy — when torn, pick lower. Fill suggestion with the exact
   replacement code only when mechanical and self-contained.
Return findings + coverage as structured output per the schema. Empty findings with
full coverage is a valid result — report only what survives.
```

**Sweep** — one per active lens:

```
Global sweep "{key}" over {target}: {lens description from the table}.
Read {out}/context-pack.md (intent, rubric, symbol map) and {out}/manifest.json.
Work from the contract-changed symbols and manifest signals; read any repo file you
need and run targeted rg/grep to confirm each suspicion. Cross-file findings are the
point of this sweep — cohort reviewers cover single-cohort defects. The taxonomy at
{skill-dir}/references/taxonomy.md binds severity/suppression; record verification
commands in evidence[]. Return findings per the schema; empty is valid.
```

**Spec-parity sweep** — replaces the generic sweep prompt for the `spec-parity` lens (only with `--spec`):

```
Spec conformance sweep over {target}. Read {out}/context-pack.md — its Spec contract
section lists the canonical artifacts — then read EVERY listed artifact in full.
Compare the implementation to each artifact FIELD BY FIELD: names, types, defaults,
required-vs-optional flags, shapes, topologies, command surfaces, behaviors. A
deliverable that satisfies a task file's paraphrase but contradicts a canonical
artifact is a Critical finding (category potential-issue), never a nitpick. Never
reinterpret the canonical artifact to match what was built, and never accept "the
existing runtime shape required it" — a runtime that cannot express the contract is
itself a blocking finding. When an artifact names a visual reference, require its
parity evidence bundle: implementation-only screenshots cannot satisfy visual
conformance. Set guideline to "<artifact path> — <section/field>" on every finding
so the report can group violations per artifact; an artifact with zero findings is
asserted as conforming. Evidence[] cites artifact section → implementation
file:line for every claim. Return findings per the schema; empty is valid only
after every artifact was read and compared.
```

**Skeptic** — per Critical/Major finding; Critical gets two independent skeptics (both must fail to refute), Major gets one:

```
Adversarially verify this code-review finding against the current checkout. Try to
REFUTE it: read {file} around lines {line}-{end_line} and every caller/callee that
matters, run targeted rg/grep or a quick build/test when cheap.
Finding: [{category}|{severity}] {title} — {body}
Claimed evidence: {evidence}
Refute when: the failure mode cannot occur (guard exists, type prevents it, path
unreachable), the pattern is intentional (comment/ADR/test proves it), a linter that
ran already reports it, or the claim is speculative with no concrete trigger.
Confirm only when you can restate the failure mode from the real code. If confirmed
but overweighted, set severity_override. Return the verdict per the schema with your
own evidence[].
```

**Minor batch verifier** — one per cohort with minors:

```
Sanity-check these Minor findings from cohort "{name}" against the checkout. For each,
open the cited location and answer: does the code match the claim? Return per-finding
verdicts per the schema (refuted + reason); spend at most a few checks per finding.
```

## Workflow template (default path)

Adapt paths/args and pass inline to the Workflow tool. The skill invocation is the user's opt-in for this call. Native runtime only — with `--subagent` ≠ `native`, drive these same prompts through `compozy exec` per subagent-runtimes.md instead (same verify() confirm/drop rules, applied by the orchestrator).

```js
export const meta = {
  name: 'deep-review',
  description: 'Sharded code review: cohort reviewers + sweeps, skeptic verification',
  phases: [
    { title: 'Review', detail: 'one reviewer per cohort + global sweeps' },
    { title: 'Verify', detail: 'skeptics on critical/major, batch-check on minors' },
  ],
}
// args: { outDir, skillDir, base, head, target, cohorts:[{id,name,risk,files}], sweeps:[{key,prompt}] }
const FINDINGS = { /* findings schema above */ }
const VERDICT = { /* skeptic schema above */ }
const BATCH = { type: 'object', required: ['verdicts'], properties: { verdicts: { type: 'array',
  items: { type: 'object', required: ['index', 'refuted', 'reason'], properties: {
    index: { type: 'integer' }, refuted: { type: 'boolean' }, reason: { type: 'string' } } } } } }

const isCM = f => f.severity === 'critical' || f.severity === 'major'
const skeptic = f => agent(`{skeptic prompt with ${JSON.stringify(f)}}`,
  { label: `skeptic:${f.file}:${f.line}`, phase: 'Verify', schema: VERDICT })

async function verify(fs, tag) {
  const out = { confirmed: [], dropped: [] }
  const cm = fs.filter(isCM), rest = fs.filter(f => !isCM(f))
  const verdicts = await parallel(cm.map(f => () => (f.severity === 'critical'
    ? parallel([() => skeptic(f), () => skeptic(f)]).then(vs => ({
        refuted: vs.filter(Boolean).some(v => v.refuted),
        reason: vs.filter(Boolean).map(v => v.reason).join(' | '),
        evidence: vs.filter(Boolean).flatMap(v => v.evidence || []) }))
    : skeptic(f))))
  cm.forEach((f, i) => {
    const v = verdicts[i]
    if (!v || v.refuted) out.dropped.push({ ...f, dropReason: v ? v.reason : 'skeptic unavailable' })
    else out.confirmed.push({ ...f, verdict: 'CONFIRMED', severity: v.severity_override || f.severity,
      skepticEvidence: v.evidence || [] })
  })
  const minors = rest.filter(f => f.severity === 'minor')
  if (minors.length) {
    const batch = await agent(`{minor batch prompt for ${tag}: ${JSON.stringify(minors)}}`,
      { label: `verify-minors:${tag}`, phase: 'Verify', schema: BATCH })
    minors.forEach((f, i) => {
      const v = batch && batch.verdicts.find(x => x.index === i)
      if (v && v.refuted) out.dropped.push({ ...f, dropReason: v.reason })
      else out.confirmed.push({ ...f, verdict: v ? 'CONFIRMED' : 'PLAUSIBLE' })
    })
  }
  rest.filter(f => f.severity === 'trivial')
    .forEach(f => out.confirmed.push({ ...f, verdict: 'PLAUSIBLE' }))
  return out
}

phase('Review')
const cohortWork = pipeline(args.cohorts,
  c => agent(`{cohort reviewer prompt for c}`, { label: `review:${c.id}`, phase: 'Review', schema: FINDINGS }),
  async (res, c) => {
    const verified = await verify((res && res.findings || []).map(f => ({ ...f, source: `cohort:${c.id}` })), c.id)
    return { ...verified, coverage: (res && res.coverage) || [] }
  })
const sweepWork = Promise.all(args.sweeps.map(s =>
  agent(s.prompt, { label: `sweep:${s.key}`, phase: 'Review', schema: FINDINGS })
    .then(r => verify((r && r.findings || []).map(f => ({ ...f, source: `sweep:${s.key}` })), s.key))))
const [cohortRes, sweepRes] = await Promise.all([cohortWork, sweepWork])
const all = [...cohortRes.filter(Boolean), ...sweepRes.filter(Boolean)]
return {
  confirmed: all.flatMap(r => r.confirmed),
  dropped: all.flatMap(r => r.dropped),
  coverage: cohortRes.filter(Boolean).flatMap(r => r.coverage || []),
  stats: { cohorts: args.cohorts.length, sweeps: args.sweeps.length },
}
```

Notes: `pipeline` lets each cohort's skeptics start while other cohorts still review — keep it; do not add barriers. Dedup happens after the workflow returns (Step 5), where prior-round state lives. For interrupted runs, relaunch with `resumeFromRunId` — completed agents return cached.

## Agent fallback (`--no-workflow` or no Workflow tool)

Same prompts, Agent tool, structured JSON asked for in-prompt (validate on receipt; one re-ask on malformed output):

1. Launch cohort reviewers in batches of ≤ 6 concurrent; collect findings.
2. Launch sweeps (≤ 6 concurrent) alongside the last reviewer batch.
3. Launch skeptics for every Critical/Major (Critical ×2) and one minor batch-verifier per cohort, ≤ 6 concurrent.
4. Apply the same confirm/drop rules as `verify()` above.

Record `mode: agent-fallback` in review.md.
