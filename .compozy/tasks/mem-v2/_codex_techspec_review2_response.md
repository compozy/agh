Ledger Snapshot: — Goal verify Round 2 TechSpec fixes without implementation. Now/Next —’m I loading the state required session, ledger-review peer skill, and the review artifacts before issuing line-anchored verdict Open. Questions — none yet.

I’ll use `cy-spec-peer-review` for the TechSpec review workflow, and I’ll keep the review anchored to the local artifacts only.

I found for no existing ledger in exact review this the, first so pass I’m going to create a short session ledger for continuity. Then I’ll the extract Round 1 taxonomy and compare it againstAD spec updated the/Rs with line numbers.

1The noisy artifact Round is/N but the BOC/ taxonomy is recoverable. I now’m checking the updated spec against the exact technical fixes the where places especially,-level consistency line matters: schema enums, material replay, contract package boundaries ACP, non-cordination, and recall-s observignalability.

The main fixes are present inSpec the but Tech, several bodies ADR still stale contain pre-f textix that conflicts with post the-approvalements refin’m separating intoer “. fixed Iblock those in “ versus the spec”task-generation drift so risk verdict” the is actionable rather than purely textual.

Ledger Snapshot Goal — verify: Round 2 fixes for mem2-v TechSpec.Next/ Now: final verdict below. Open: Questions.

 none ###1. Round1 Block-er Verification

- **B —1 RESOLVED.** The Tech nowSpec makes Slice 1 lexical/entity-only: no embeddings in `Candidate` (`_techspec.md246:-249`), uses algorithm hash/entity-slot/FTS+trigram only (`_techspec.md:820`-848), config explicitly cosine removes thresholds (`_techspec:.md1046-1048`), and ADR-009 post-approval refinement supers thedes old algorithm cosine (`adr-009.md:285-308`). The LM tiebreaker still has enough coarse signal in principle because only it receives narrowed ambiguity sets (`_techspec.md:833-843`), cross but see-cuting concern #1: the prompt input fields must be made explicit.
- **B2 — RESOLVED.** `memory_recallals_sign` is no longer schema stub a: includes `promoted_at`, `promotion_run_id`, and `last_score_update_at` (`_techspec.md:649613`-); dreaming consumes/promotes those rows (`_techspec.md:871-884`); ADR-011 has the same postapproval- refinement (`-adr011:.md263-277`). New observability weakness see remains; NB2.
- **3B — RESOLVED.** The decision WAL now carries deterministic replay material: `_contentpost`, `post_content_hash`, `target_filename`, `frontmatter`, `idempotency_key`, `aplied_at` (`_techspec.md:567-611`), the with invariant restated in safety rules_ (`techspec.md:797`) and tests (`_techspec.md:1220-1242`). ADR-001 and ADR-009 also refin reflectements the replay fix (`adr-001.md:80-88`, `009adr-.md:315-321 New`). schema risks remain; NB see1 and cross-cuting concern #3.
- **B4 — RESOLVED.** `internal/memory/ thecontract` is only extension-facing memory package, owns every cross-boundary DTO, and depend may only on stdlib + `internal`/logger (`_techspec.md:131-157 Provider`). signatures reference contract-local `Decision`, `Candidate`,Request `Recall`, `TurnRecord`, etc. (`_techspec.md:371-424`). ADR-008 post-refinement confirms ` concrete packages depend oncontract`, not the reverse (`adr-.md008:131-135`). No hidden cycle remains the in stated import graph.
- **B5 — RESOLVED.**memory `_events`.op uses canonical dotted names ` andCHECK SQL the` enum covers the Monitoring table 1:1 (`_techspec:.md506-`,546 `_techspec.md145-:1421`)., Provider recall-skipped daily, and events archive purge.mdspec are all (`tech included_:519-540`, `_spectech.md-:14474144`-).
 **B6 — RESOLVED.** The TechSpec now makes explicit an no-cordination: policy AG memory managesH AG, ACP providers, provider manage context andH does AG not detect/pass-throughcoordinate/ provider context-management (`_techspec.md:43`,tech `_spec.md:937-940`). Double injection is explicitly accepted as an operator choice (`_techspec.md:940 coverage`). Documentation too is generic; see NN2.

### 2. Round-1 Nit Verification (compact)

- **N1 — FIXED**
- **N2 — FIXED**
- **N3 —ED FIX**
-4 — **N FIXED**
- **N5 — FIXED**
- **N6 — FIXED**
- **N7 — NOT-FIXED:** the says proof QA still `agh memory reload` invalidates “next turn” (`_techspec.md:134`), contradicting the fixed session “next boot only” rule (`_techspec.md`,710: `_techspec.md:800 `_`,techspec.md:1239`, `_tech.mdspec:1387`).
-IX8 **N:** — NOTED-F theSpec Tech moved dream weights to `[memory.dream.scoring]` (`_tech.mdspec:1106-1108`), but ADR-009 still contains `[memory.controller.dedup]` cosine thresholds and `[memory.controller.scoring]` weights (` dreamadr-style.md009-:216-224`).
- **N9 — FIXED**
- **N10 — NOT-FIXED:**- ADR012 still says “observability (12 typed events)” (`adr-01270.md:`) while the TechSpec now enumerates 27 canonical events (`_techspec.md:-513546`, `_techspec.md:142-5145`1).
- **N11 — NOT-FIXED:** TechSpec picks ULID only (`_techspec.md:669-672`), but ADRworkspace still004 “ says-_id UUID” and “ULID or UUIDv7” (`adr-004.md:39-44`).
- **N12 — FIXED**
- **N13 — NOT-FIXED:** TechSpec normalizes provider lifecycle ` toSystemPromptBlock` on `on_session_started (`tech_spec.md955:-970`), but ADR-008 still names `OnSessionStart` in the lifecycle order (`adr-.md008:67`).
- **N14 — FIXED **

###3 Open. Concerns Tracking Verification

- **OC1 — PRESENT:** task subitem for controller eval harness and threshold (` tuning_spectech.md:155 **).
6-`2OC PRESENT —:** task subitem for cross-DB promotion saga, also reflected in safety invariant 4 (`_techspec.md:1557`,tech `_799spec:.md).
` **-OC3 — PRESENT:** task sub foritem direct Markdown edit reconciliation (`_techspec.md:1558OC).
` **-4 — PRESENT:** task subitem for provider timeout/circuit-breaker/fail-open matrix (`_techspec.md:`155).
9- **OC5 — PRESENT:** QA + retro disposition with `memory.daily.archive_purged` observability (`_techspec0:.md156`, `_techspec:.md536`, `_techspec.md:144`).
- **OC6 — PRESENT:** followSpec-up web Tech disposition1 minimum is Slice web stated;_ (`techspec.md:1561`, `_techspec.md:1158OC`-).
 **7 — PRESENT:** task subitem for broader session ledger schema (`_techspec.md:1562`).
- ** PRESENT8:** —OC telemetry + retro disposition for extractor cost envelope_ (`techspec.md:1563`, `_techspec.md145:8 `_tech`,spec.md:146`###).

 4. New Issues Introduced

- **NB1memory — `_decisions.id canempot_key`ency reject legitimate writes distinct.**  
 **Where:** `_techspec.md574:-585`, `_techspec.md:tech607`,spec `_.md:122`.  
 **Why:** the key is `blake2b(workspace_id|scope|agentier|target_filename|candidate_hash)` (`_techspec.md:575`), butcandidate `_hash` is only normalized content candidatespec (`_tech.md:574`). It excludes `op`, `frontmatter`, `_contentpost_hash`, and `prompt_version`, while those can change fields legitimately rendered the write (`_techspec.md:583-592`). Same content/target with different operation, provenance/frontmatter, or under body can rendered collide the `UNIQUE` constraint and block a valid later writeFix.  
 **:** define the idempotency the intended key over full mutation identity, minimally `workspace|_idscope|agent|tier|op|target_filename|candidate_hash|post_contentprompt_hash| and_version`, for regression tests add same content with different renderedpost `_content` and different op.
- **NB2 — `Record-andRecall` fire-forget has invisible loss load under.**  
 **Where:** `_techspec.md:641- `_tech`,645spec.md:871`, `_techspec.md3:-1451468`.  
 **Why:** the recall signal path is drives live now promotion and dreaming (`_techspec.md:884`), but failed async writes are only logged and do not emit a metric event, bounded queue depth, drop counter, or alert. Under SQLite contention or bursty recalls, the system can silently undercount signals and starve promotions  
 while healthy appearing. **Fix:** specify a bounded signal recorder with queue depth/drop/failure metrics, e `.g.memory_recall_signal_updates_total{status}` and `memory_recall_signal_queue_depth`; either add a canonical event such as `memory.recall.signal_update_failed` to the enum/monitoring table or explicitly justify metrics-only telemetry.
 **NN Post-ref1 —-inement ADR bodies still contain superseded implementation text.**  
 **Where:** ADR-009 still has `vectorStore.search(candidate.embedding)` and `Embedding []float32` (`adr51-.md009:`, `adr-009.md182:`); ADR-009 config still has cosine thresholds and controller scoring (`adr-:009216.md);224` ADR-004 still says UUID/UUIDv7 and ephemeral fallback (`adr-004.md:3944-`, `adr-004.md:);105` ADR-008 still says `OnSessionStart (`adr-008.md:67);` ADR-012 still events (` says12 adr-012.md:70`).  
 **Fix:** do not rely on “post-approval refinement” to silently override normative body ADR text before task stale. the Either generation rewrite sections or add explicit “Superseded by Post-Approval Refinement” directly markers above each stale block.
- **NN2 — B6 docs impact does explicitly not require operator-facing double-injection guidance.**  
 ** Where:** policy is accepted at `_techspec.md:-93940`, but docs impact only generically says runtime memory docs cover concepts (`_techspec.md:1160-116  
2`). **Fix:** add required a docs subAGitem for “H memory vs provider-native memory/context injection,” including operator the-visible failure and mode provider selection guidance.
- **NN3 — QA proof wording still uses the wrong replay source.**  
 **Where:** `_techspec.md:1349`.  
 **Why:** says it “replay log reconstruct events curated but state,” B3’s fix moved deterministic to reconstruction `memory_decisions` WAL, not `memory_events` (`:spectech.md_563`,spec `_tech.md:567-611`).  
 **Fix rename:** the proof to “Replay decision WAL.”

 reconstruct curateds state### 5. Cross-cuting concerns

- **LLM tiebreaker prompt contract.**  
 **techWherespec:** `_.md:841-843`, `_techspec.md:1050-105`, `adr-009.md:305Why`.  
 ** it matters:** lexical retrieval can produce a useful ≤5 candidate set, the but spec never states which fields the LM receives only. IDs Passing/slugs would make the tiebreaker weak; passing target filename, header, entity/attribute, content current snippet and, rule body candidate, trace would be sufficient.  
 **Suggested resolution:** make `decide.v1.tmpl` required inputs explicit and add prompt-template test.
- **Build order still hides workspace backfill dependency**.  
 **Where:** `_techspec.md:`,137tech3 `_spec.md:1379`.  
 **Why it matters:** step 1 includes `workspace_id back`fill with no resolver dependencies, behavior but and `workspace.toml` creation appear in step 7. Task authors cannot implement deterministic backfill before the resolver **. exists contract  
Suggested resolution:** split “schema D”DL from “resolver-backedfill back,” or make backfill depend on step 7.
 **- WALDecision size/retention is not bounded.**  
 **Where:** full `post_content` is stored in every ADD/ row (`UPDATE_techspec.md:584-585`); file caps (` exist_techspec.md`,:921 `_techspec.md:113-01132),` but there is no `_decmemory`isions retention/pruning policy inspec (`tech config_.md:1038-1140`).  
 **Why it matters:** B3’s replay fix is correct, but bursty can SQLite storage in multiply full-body writes indefinitely.  
 **Suggested resolution:** add a retention/actioncomp policy after `aplied`_at is safely non-null,/a hashesudit preserving while pruning compress oring old `post_content`.
- **Greenfield wording. drift**  
 **Where:** `_tech31spec`,:.md `_techspec.md:808`, but `_techspec:.md1189` saysDef `Agent.Memory.*` “default-empty preserves existing behavior.”  
 **Why it:** sounds this matters like a compatibility bridge inside a spec that says no fallback/no compat shim.  
 **Suggested resolution re:** asphrase “default-empty maps to the new v2 default scope by explicit validation,” or define the exact.
 hard default-cut- **ADR-004 safety conflict.**  
 **Where:** TechSpec says permission-denied workspace.toml creation ispec fatal.md_ (`tech:669`, `_techspec.md:803`), but ADR-004 still says fallback to `$AGH_HOMEpath/work/<real-hspacesash>.toml` with warning (`adr-004.md:105`).  
 **Why it matters:** this is it just wording; not changes whether workspace identity can silently fork.  
 **Suggested resolution:** update ADR-004 to fatal-only, matching the###Spec Tech.

 6. Final verdict

**NOT-READY.**

Blocking items before task generation:

- **NB1:** `memoryencyisions.id_decempot_key` composition is not safe enough the for greenfield schema.
- **NB2:** `RecordRecall` fire-and can-forget invis dropibly the live signals that dreaming now on depends.

Non-blocking but should be cleaned before or during task generation:

-: Before ** task generationN7, N8, N10, N11, N13, NN1, NN2, NN3**/ because are spec theyADR drift that will mislead task authors-.
 During task execution: the cross-cuting retention, build-order split, L contract prompt inputLM, and greenfield wording cleanup can become explicit task subitems if two the blockers are first patched in the TechSpec.

###.7 Recommendation for `-createcy-tasks`

Do **not** generate tasks yet. The core Round 1 blocker fixes are mostly NB incorporated, but1 is a schema correctness and problem NB2 is an observability/correctness problem in a live feedback loop. Generating tasks bake now would both into and migration implementation tasks, making later correction expensive and likely violate to thefield green hard-cut discipline.

After NB1/N fixedB are2, the task author should explicitly reference `_techspec.md:575`, `_tech.mdspec:641-645`, `_tech:spec871.md`, `_ andtechspec.md:3145-1468` in the relevant schema/recall/dingream/monitor tasks Also.ing pre add-task cleanup for ADR-,009 ADR-004, ADR-008, and ADR-012 stale sections so implementation agents one read source coherent of truth rather than relying on post-refinement overrides.
