Ledger Snapshot Goal:- Tech review2 RoundSpec incorporation formem `-v2`, read-only. Now/Next:’m creating I this session’s ledger and then I verify’ll the requested findings against the TechSpec line/ withRsAD evidence Open.cy Questions: usarVou ` none-spec.

-preflight` e `cy-spec-peer-review` porque isso é ver umaificação final deSpec Tech antes de gerar; tasks aplicar vou só o workflow revisão de, sem implementar nada.

A lista de ledger já tem2 umem-v `-review`; antigoá trat vou-lo como contexto read e-only criar um ledger separado para ver estaificação Round 3 para evitar colisão com. sessão caminho outra O dos skillsório está noagents/ do `. repo` diret, carregar então vou esses arquivos. dali

Vou persistir memória apenas de sessão, sem ou no Tech tocarSpec ADRs. Em seguida vou localizar as linhas ex deatas NB1/NB2, nits ADR e markers para fechar parecer o.

Agorar vou pux o primeiro Round 2 bruto para não vaga revisar contra memória. vou Depois procurarSpec no Tech os que termos deveriam ter mudid:empot `adoency`,_key `,RecordRecall queue/eventsmetrics, ADR superseded markers, docsitems sub e QA rename.

NB1/NB2 aparecem principal incorporados corpo no testes com eicas. métr Ainda estou checando o ponto perig maisoso para: task generation com que ADRs continuam texto normativo antigo, porque isso pode virar instrução errada para quem vai gerarimple/mentar tasks.

A incorp limpaoração não está o corpo: principal resolveu partes importantes, mas há contradições normativas em `memory_recall_signals`,- ADR/009/011012 e na do retenção WAL. Vou o atualizar ledger o com final estado da revisão antes de responder.

Ledger Snapshot2 Goal Round verify:-: incorporation for `mem-v2` task beforeNext. Now/ generation verification: complete final; below verdict Open. Questions: none.

 ###1. NB1 +2 NB

 verification1 — PARTIALLY- **NB-RESOLVED.** The requested additions are present: `IdempotencyKey` now includes `op|post_content_hash|_version contract and`prompt in the_ schema (` SQLtechspec.md:271`, `_techspec:.md577`), the rationale calls (` fields out those_spectech.md:609 the regression`), three and tests exist (`_techspec:`-02125.md`)., However `frontmatter` is part explicitly of theutation/m replay (` material_techspec.md:`264-`,`tech267 `_.mdspec:585`, `_techspec.md:612`) but remains outside thempot idency (` key_techspectech`,.md: `_271spec.md:577`). Same body/same op/same prompt version with different serialized header stillides coll.

-I ** — PARTNB2ALLY-OLRESVED.** The Tech addsSpec the right mechanism: live `memory_recall_signals` (`.md_techspec:615`-`617`), bounded `SignalRecorder` queue + drop/failure/events metrics (`_tech-spec.md`643:651`), canonical events in `memory_events.op` (`_spectech.md:519`-`522`),_ metrics (`tech.mdspec:1498`-`149`), and tests (`_techspec.md:1282`-`).1284 But normative stale still guidance says the migration creates recall with signals “stub data flows in slice 2” (`_techspec.md690:`), and ADR011- still says recall signals are extracted/populated in slice 2 or have zero weight in slice 1 (`:-011adr.md`,72 `adr-011.md:147`, `adr-011.md:239`). That directly contradicts NB2’s required-s liveignal behavior.

### 2. Previously-NOT-FIXED n statusits

- **N7 — FIXED.-session** is Reload consistently next/next-boot: only CLI (`_techspec.md717:`), invariant (`_spectech.md:807 unit`), test (`_techspec.md:126`9), QA proof (`_techspec.md:137`).

- **N FIX8ED —.** ADR-009 controller marks dedup/scoring config as superseded and moved/deleted (`adr-009.md:`213-223);` TechSpec puts weights under `[memory.dream.scoring]` (`_techspec.md:1132`-113`)4` lists and old controller blocks as delete (` targets_techspec.md:1582`).

- **N10 — NOT-FEDIX.** ADR-012 says still “observability (12 typed events)” in the Slice 1 table (`adr-012.md while:70`) the TechSpec CHECK enumer enumates the canonical event set (`_techspec.md:511`-`548`). ADR-012 admits the later “ typed events” wording was stale (`adr-012.md:114`), but the stale table remains.

- ** NOTN11-F —IXED.** TechSpec chooses ULID (`_techspec.md:676` and), ADR-004’s example/ ULnotesID say (`adr-004.md:44`, `adr-004.md:110`), but ADR-004 still normatively UUID saysworkspace “”_id (`adr-004.md:39`) andUUID “ stored” in the chosen alternative (`adr-004.md:71`73-).

- **N13 — FIXED.** ADR-008 now uses `on_session_started` firing `SystemPrompt says` and explicitlyBlock there is ` noOnSessionStart` method (`adr-008.md:67`).

### 3. Previously nrodu-intcedits status

 **-NN1 — NOT-FIXED.** Superseded markers to were added some ADR-009 blocks (`009adr.md-:44`-`46`, `adr-009.md:156`-`162`, `adr-009.md:`223213-`), but stale unmarked guidance remains: controller “thresholds” and mandatory embeddings in consequences/risks (`adr:009-.md137`, `adr-009.md:143`-`144`, `adr-009.md:`151), ADR- recall-011 slice2-signal claims (`adr-011.md:72`, `adr-011.md:147`, `adr-011.md:239 and`), ADR-012 “ typed events” (`adr-012.md:70`).

- **NN2 — FIXEDocs** Web/. Impact now adds `memory-vs-provider.mdx` with AGH-memory vs provider-native boundary-memory, double-in selection modejection failure provider, guidance, and non-cordination policy (`_techspec.md118:6`-`118`).

- ** FIXNNED3 —.** QA proof is/source renamed-corrected to `Replay-decision-WAL-reconstructs`-state and explicitlyisionsmemory uses `_dec`, not `memory`_events_ (`.mdtechspec2:138`).

### 4. Cross-cuting concerns status

- **CC#1 — LM prompt fields — RESOLVED.** `llmDecide` inputs now specifies prompt: candidateattribute entity/, content, rule trace, target filename/front/snmatterippet/last-updated plus, template regression tests (`_techspec.md:-848`862`).

- **CC#2 — build order split — RESOLVED.** Schema D isDL separated from: resolver back-backedfill step 1 explicitly has no backfill (`_techspec.md:1406`), and step `7b` step resolver depends on 7 (`_techspec.md:1412`-`1414`).

- **CC#3 — retention WAL. —I PARTALLY Config** now defines `[memory.decisions] prune_after_aplied_days`,keep `_audit_summary`, and `max_bytes_post`_content (`_techspec.md:0110`-`1103`). But `keep_audit_summary` says it writes a compact audit row to `memory`,_events while the `memory_events.op enum CHECK` has no decisions-retention/a-summaryudit event (`_techspec.md511:-`548`) and the Monitoring table has no event such (`_techspec.md:1458`-`148).

- **CC#4 — greenfield wording — RESOL.VED** `AgentDef.Memory.*` no longer says “preserves behavior it existing”; says-empty default validates as the v2 routing default and is not a compat (` bridge_techspec:.md5121`6-`121).

`- **CC#5 — ADR-004 fallback RES —OLVED.** TechSpec says permission-denied is identity with creation workspace fatal no silent fallback (`_techspec.md:810`, `_techspec.md:1412` and), ADR-004 while matches explicitly fatal-only supers theding ephemeral fallback (`adr-.md004:105`).

### 5. New issues introduced

 **-3 `NB —idempot_keyency` still omits `frontmatter` from mutation. identity**  
 **:**Where `_techspec264.md`:-`267`, `_techspec.md:271`,tech `_spec.md:577`, `_techspec.md:`,585 `_tech.mdspec:612`, `_techspec.md:1250`-`1252`.  
 **Why:** The spec says `front part` ismatter of deterministic/file replay state, stores it as a separate column, andizes serial it for byte-stable replay, only but hashes the unique key workspacecope/s/agent/tier/target/op/candidate/post_content/prompt_version. header A-only mutation with the same bodyides still coll with a previous row.  
 **Fix:** Add `frontmatter_hash` serialized or-frontmatter hash to `idempot persist_key`,ency the hash if useful, and add `TestController_Didec_IdempotencyKeyDistinctOnFrontmatterChange`.

- **NB4Record — `Recall` still has stale slice-2/stub instructions in task-author inputs.**  
 **Where:** `_690spec`,tech.md: `-adr011.md:72`,adr `-011.md:147`, `adr-011.md:239`, plus’s-011 ADR post-refinement still says failures are only logged (`-adr011.md:269`-273`) while the TechSpec (` metrics requires/events_techspec.md:643`-`651`).  
 **:**Why Task authors reading migrations/ADR-011 can `_re implementmemory_signcallals` as a stub defer and live writes, exactly the NB2 failure mode.  
 **Fix:** Rewrite those/body ADR to lines live Slice 1 `SignalRecorder` semantics with queue, capacity drop/failure metrics, canonical events, and status on. CLI Do not rely a later refinement paragraph override to normative stale text.

 **-NB5 — WAL retention audit row cannot be inserted under the current. enum event**  
 Where **:** `_techspec.md:110`-`110`,3 `_techspec.md:548511`,-`tech `_spec.md:1458-`148`.  
 Why **:** `keep_audit_summary = writing true` requires a compact row audit to `_eventsmemory`, but the closed has enum no `memory.decisions.pr`uned / `memoryummar.dec.audit_sisionsized` event Implement.ing the stated config will violate the CHECK constraint or force misuse of an unrelated event.  
 ** a:** AddFix canonical decisions SQL-ret evention to + Monitoring tests +, or change the retention design to store summaries outside `memory_events###`.

 .6 Final

 verdict**NOT-READY.**

Blocking items before task generation:

- **3NB:** mutation identity still excludes`, ` persistedmatterfront a replay field.
- **NB4:** ` liveRecordRecall` semantics are contradicted by migration/ADR-011 slice2- language stub.
-NB **5:** WAL retention summary targets `memory_events` without an canonical allowed event.

Also clean before task generation10: **N, N,11 NN1**. These are stale source-of-truth issues that will mislead task generation.

### .-t7 cy-createasks recommendation

Do **not generate tasks now. The main TechSpec body moved in the direction right, but task generation in currently would bake three contradictions: incomplete decision idempotency, `memory_recall_signals` live-vs-stub drift, and an un WALableinsert-retention audit event under the closedmemory `_events.op` enum.

 `Beforecy-create-tasks`, the patch spec/ soRs the taskAD author references one coherent source: update idempotency at `_techspec.md271:`, `_techspec.md:577spec `_`,tech.md:609`, tests at `_tech.mdspec`1250:-`1252`; rewrite `_techspec.md690`: and ADR-011 lines `72`, `147`, `239`,269 `-`273`; the add decisions-retention canonicaltech eventspec `_ in.md:511`-`548 and` `_tech:spec.md145`8-148`;8 fix ADR `012- line70` and ADR-004 lines `39`, `71`-`73`.
