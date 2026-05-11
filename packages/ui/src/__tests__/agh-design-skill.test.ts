import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const SKILL_PATH = join(__dirname, "../../../../.agents/skills/agh-design/SKILL.md");
const skill = readFileSync(SKILL_PATH, "utf-8");

function rulesBlock(): string {
  const start = skill.indexOf("**Rules that matter most for this brand");
  const end = skill.indexOf("**Site profile");
  if (start === -1 || end === -1) throw new Error("Rules block markers missing in SKILL.md");
  return skill.slice(start, end);
}

function ruleBullets(): string[] {
  return rulesBlock()
    .split(/\r?\n/)
    .filter(line => line.startsWith("- **"));
}

const PR1_RULES: { name: string; needle: RegExp; citation: RegExp }[] = [
  {
    name: "Surface glaze tokens are mandatory",
    needle: /Surface glaze tokens are mandatory/,
    citation: /ADR-001 §6/,
  },
  {
    name: "Owner / kind palettes route through --avatar tokens",
    needle: /Owner \/ kind palettes route through `--avatar-/,
    citation: /ADR-001 §7/,
  },
  {
    name: "Modal-scrim alpha 0.55 + --overlay-blur",
    needle: /Modal-scrim alpha is `0\.55` and applies `var\(--overlay-blur\)`/,
    citation: /ADR-001 §2/,
  },
  {
    name: "::selection resolves to var(--accent-tint)",
    needle: /`::selection` resolves to `var\(--accent-tint\)`/,
    citation: /ADR-001 §1/,
  },
  {
    name: "Shadow vocabulary is two tokens only",
    needle: /Shadow vocabulary is two tokens only/,
    citation: /ADR-001 §4/,
  },
  {
    name: "--neutral-tint pinned at rgba(150, 150, 155, 0.06)",
    needle: /`--neutral-tint` is pinned at `rgba\(150, 150, 155, 0\.06\)`/,
    citation: /ADR-001 §8/,
  },
  {
    name: "--font-display is an Inter alias reserved for tokens.css",
    needle: /`--font-display` is an Inter alias reserved for `tokens\.css`/,
    citation: /ADR-001 §5/,
  },
  {
    name: "Tailwind rounded-md / rounded-lg / rounded-xl forbidden",
    needle: /Tailwind `rounded-md` \/ `rounded-lg` \/ `rounded-xl` utilities are forbidden/,
    citation: /ADR-001 §9/,
  },
  {
    name: "Eyebrow tuple inline ban",
    needle: /Inlining `font-mono uppercase tracking-mono`/,
    citation: /ADR-002 §1/,
  },
  {
    name: "Body baseline pinned",
    needle: /Body baseline is pinned to Inter 13\.5 px/,
    citation: /ADR-002 §6/,
  },
  {
    name: "Mono identifiers bare, never <Pill mono>",
    needle: /Mono identifiers in row contexts.*render as bare mono text, never as `<Pill mono>`/,
    citation: /ADR-002 §1/,
  },
  {
    name: "Lucide stroke-width ramp (1.75 default, 2 at xs)",
    needle:
      /Lucide icons use `strokeWidth=\{1\.75\}` by default; only at the 11 px `xs` size does `strokeWidth=\{2\}` apply/,
    citation: /ADR-010 §9/,
  },
  {
    name: "Loader2 banned in production routes",
    needle: /`Loader2` from `lucide-react` is banned in production routes/,
    citation: /ADR-009 §5/,
  },
];

const PR2_RULES: { name: string; needle: RegExp; citation: RegExp }[] = [
  // §5.3 Layout & Shell
  {
    name: "Shell grid 56 / 244 / 1fr ladder",
    needle: /Shell grid is `56 px rail \+ 244 px sidebar \+ 1 fr content`/,
    citation: /ADR-003 §4|ADR-005 §1/,
  },
  {
    name: "Active workspace nub + selected rails use --fg-strong",
    needle:
      /Active workspace nub \+ selected nav rail \+ selected list-row rail all paint `var\(--fg-strong\)`/,
    citation: /ADR-003 §3/,
  },
  {
    name: "<Topbar> canonical route-identity surface + useNavCounts integration",
    needle: /`<Topbar>` is the canonical route-identity surface/,
    citation: /ADR-005 §[38]/,
  },
  {
    name: "Detail-mode topbar swaps to back + title + meta + overflow",
    needle:
      /Detail-mode topbar swaps to `back \+ title \+ id-chip \+ sep \+ meta \+ spacer \+ actions \+ overflow`/,
    citation: /ADR-005 §5/,
  },
  {
    name: "<RuntimeConnectionIndicator> single daemon LED owner",
    needle: /`<RuntimeConnectionIndicator>` is the single owner of the daemon LED/,
    citation: /ADR-005 §1/,
  },
  // §5.4 Components
  {
    name: "<Button> 9 additive variants (no rename, no drops)",
    needle:
      /`<Button>` runtime variants are additive: `default \| primary \| outline \| secondary \| ghost \| destructive \| success \| link \| neutral`/,
    citation: /ADR-004 §1/,
  },
  {
    name: "<Pill> 4 px radius across sizes + uppercase prop gone",
    needle: /`<Pill>` ships 4 px radius across every size/,
    citation: /ADR-004 §3|ADR-002 §1/,
  },
  {
    name: "<PillGroup> Inter sentence-case 12/510 + neutral count badge",
    needle: /`<PillGroup>` track is borderless `var\(--canvas-soft\)`/,
    citation: /ADR-004 §4/,
  },
  {
    name: "<Tabs> --fg-strong 1.5 px underline + lane variant",
    needle: /`<Tabs variant="line">` underline is `var\(--fg-strong\)` 1\.5 px/,
    citation: /ADR-003 §7|ADR-004 §2/,
  },
  {
    name: "<Empty> top-padded layout + borderless icon-well",
    needle: /`<Empty>` icon-well is borderless `var\(--canvas-soft\)`/,
    citation: /ADR-004 §6/,
  },
  {
    name: "<Metric> + <KpiCard> flat, sentence-case Inter labels",
    needle: /`<Metric>` and `<KpiCard>` are flat \(no border\)/,
    citation: /ADR-004 §9|DESIGN\.md §11/,
  },
  {
    name: "<Dialog> title 13.5 + modal width ladder",
    needle: /`<Dialog>` title is 13\.5 px \/ 510 \/ -0\.012em/,
    citation: /ADR-008 §1|ADR-011 T-1/,
  },
  {
    name: "<RadioCard> --surface-glaze + inset --line-strong ring",
    needle:
      /`<RadioCard>` selected state paints `var\(--surface-glaze\)` \+ `box-shadow: inset 0 0 0 1px var\(--line-strong\)`/,
    citation: /ADR-004 §8/,
  },
  {
    name: "<CatalogCard> flat, padding 16, logo well 24 or 40",
    needle: /`<CatalogCard>` is flat, padding 16 px/,
    citation: /ADR-004 §7|ADR-011 §8/,
  },
  {
    name: "Form inputs use --input-fill, no h-9/h-10 override",
    needle: /Form inputs ship `bg-\(--input-fill\)` \+ transparent ring at rest/,
    citation: /ADR-008 §9/,
  },
  // §5.5 Primitive scaffolding
  {
    name: "Use <DetailHeader> for detail hero — 6-row anatomy",
    needle: /Use `<DetailHeader>` for the detail hero/,
    citation: /ADR-003 §2|DESIGN\.md §11/,
  },
  {
    name: "Use <FormSection> inside modals + editable forms",
    needle: /Use `<FormSection>` \(not `<Section>`\) inside modals and editable forms/,
    citation: /ADR-008 §9|ADR-015 §7/,
  },
  {
    name: "Use <MonoId> for row-context identifiers",
    needle: /Use `<MonoId>` for identifiers in row contexts/,
    citation: /ADR-002 §1|ADR-014 §3/,
  },
  {
    name: "Use <RunCard> for active-run + standalone run-detail",
    needle: /Use `<RunCard>` for active-run \+ standalone run-detail rendering/,
    citation: /ADR-007 §9|ADR-008 §3/,
  },
  {
    name: "Use <DescriptionCard> + STREAMDOWN_SAFE_CONFIG",
    needle: /Use `<DescriptionCard>` for markdown bodies/,
    citation: /ADR-007 §2/,
  },
  {
    name: "Use <OwnerAvatar> for owner identity surfaces",
    needle: /Use `<OwnerAvatar>` for owner identity surfaces/,
    citation: /ADR-010 §6|ADR-013 §2/,
  },
  // §5.6 Status & motion
  {
    name: "Motion: three named keyframes only, animate-pulse banned",
    needle: /Motion uses exactly three named keyframes/,
    citation: /ADR-010 §[78]/,
  },
  {
    name: "STATUS_TONE single declarative typed dictionary",
    needle: /Status → tone is one declarative typed dictionary at `web\/src\/lib\/status-tone\.ts`/,
    citation: /ADR-010 §5|ADR-016 §3/,
  },
  {
    name: "<PriorityBars> three bars, color shifts not bar count",
    needle: /`<PriorityBars>` always renders three bars/,
    citation: /ADR-006 §4/,
  },
  {
    name: "<QueueHealthSparkline> recharts adapter, --bar-fill default",
    needle: /`<QueueHealthSparkline>` consumes `recharts` via the `@agh\/ui` adapter/,
    citation: /ADR-010 §3|ADR-016 §7/,
  },
  {
    name: "<StatusDot> 4-tone × 2-mode × 2-size grammar",
    needle: /`<StatusDot>` ships 4 tones/,
    citation: /ADR-009 §5/,
  },
  {
    name: "<Time> canonical timestamp + @agh/ui re-export shim",
    needle: /`<Time>` is the canonical timestamp primitive/,
    citation: /ADR-007 §3|ADR-016 §10/,
  },
  // §5.8 Anti-patterns (compile-fail rules)
  {
    name: "Anti: font-mono uppercase tracking-mono inline tuple",
    needle: /Anti-pattern \(compile-fail\): `font-mono uppercase tracking-mono`/,
    citation: /ADR-016 §5|DESIGN\.md §11/,
  },
  {
    name: "Anti: text-[22px] page-h1 tuple inline",
    needle:
      /Anti-pattern \(compile-fail\): `text-\[22px\] font-medium tracking-\[-0\.026em\]` page-h1 tuple inline/,
    citation: /ADR-003 §2|DESIGN\.md §11/,
  },
  {
    name: "Anti: border-l-* border-l-(--accent) on cards / rows",
    needle: /Anti-pattern \(compile-fail\): `border-l-\* border-l-\(--accent\)`/,
    citation: /ADR-003 §3|DESIGN\.md §11/,
  },
  {
    name: "Anti: hover:border-(--accent) / accent-on-hover",
    needle: /Anti-pattern \(compile-fail\): `hover:border-\(--accent\)`/,
    citation: /ADR-004 §1|ADR-016 §5/,
  },
  {
    name: "Anti: bg-[rgba(255,255,255,0.0NN)] glaze literals",
    needle: /Anti-pattern \(compile-fail\): `bg-\[rgba\(255,255,255,0\.0NN\)\]` literals/,
    citation: /ADR-001 §6|DESIGN\.md §10/,
  },
  {
    name: "Anti: Loader2 import under web/src/routes/**",
    needle:
      /Anti-pattern \(compile-fail\): `import \{ Loader2 \} from "lucide-react"` under `web\/src\/routes\/\*\*`/,
    citation: /ADR-009 §5|ADR-010 §8/,
  },
  {
    name: "Anti: Pill tone=accent co-located with Button variant=primary",
    needle:
      /Anti-pattern \(review fail\): `<Pill tone="accent">` co-located with `<Button variant="primary">`/,
    citation: /ADR-001 §10|DESIGN\.md §11/,
  },
];

describe("agh-design skill rules block (PR-1, task_03; PR-2, task_17)", () => {
  describe("YAML frontmatter + markdown structure", () => {
    it("Should expose YAML frontmatter with name=agh-design and user-invocable=true", () => {
      const fm = skill.match(/^---\n([\s\S]*?)\n---/);
      expect(fm, "expected SKILL.md frontmatter").toBeTruthy();
      const body = fm?.[1] ?? "";
      expect(body).toMatch(/^name:\s*agh-design$/m);
      expect(body).toMatch(/^user-invocable:\s*true$/m);
    });

    it("Should keep the rules-block heading + the Site profile boundary", () => {
      expect(skill).toMatch(
        /\*\*Rules that matter most for this brand \(full reasoning in `DESIGN\.md`\):\*\*/
      );
      expect(skill).toMatch(/\*\*Site profile \(when working in `packages\/site\/`\):\*\*/);
    });

    it("Should render every rules-block entry as a `- **Headline.**` bullet", () => {
      const bullets = ruleBullets();
      expect(bullets.length).toBeGreaterThanOrEqual(PR1_RULES.length + PR2_RULES.length);
      for (const bullet of bullets) {
        expect(bullet, `bullet must open with "- **": ${bullet}`).toMatch(/^- \*\*/);
        const afterOpen = bullet.slice(4);
        expect(afterOpen, `bullet must carry a closing "**": ${bullet}`).toContain("**");
      }
    });
  });

  describe("Rule count (PR-1 + PR-2)", () => {
    it("Should carry the PR-1 baseline + every PR-2 addition", () => {
      expect(ruleBullets().length).toBeGreaterThanOrEqual(PR1_RULES.length + PR2_RULES.length);
    });
  });

  describe("§5.1 token discipline rules (ADR-001)", () => {
    const slice = PR1_RULES.slice(0, 8);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("§5.2 typography subset rules (ADR-002)", () => {
    const slice = PR1_RULES.slice(8, 11);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("§5.7 iconography rules (ADR-009 + ADR-010)", () => {
    const slice = PR1_RULES.slice(11);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("ADR / DESIGN.md citation discipline (PR-1 + PR-2)", () => {
    it("Should cite at least one ADR-NNN or DESIGN.md anchor in every PR-1 + PR-2 rule bullet", () => {
      const block = rulesBlock();
      for (const rule of [...PR1_RULES, ...PR2_RULES]) {
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} bullet must cite an ADR or DESIGN.md anchor`).toMatch(
          /ADR-\d{3}\s§\d+|`DESIGN\.md` §\d+|`analysis\.md` §\d+(?:\.\d+)?/
        );
        expect(block, rule.name).toContain(bullet);
      }
    });
  });

  describe("§5.3 layout + shell rules (task_17)", () => {
    const slice = PR2_RULES.slice(0, 5);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("§5.4 component rules (task_17)", () => {
    const slice = PR2_RULES.slice(5, 15);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("§5.5 primitive scaffolding rules (task_17)", () => {
    const slice = PR2_RULES.slice(15, 21);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("§5.6 status & motion rules (task_17)", () => {
    const slice = PR2_RULES.slice(21, 27);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("§5.8 anti-pattern compile-fail rules (task_17)", () => {
    const slice = PR2_RULES.slice(27);
    for (const rule of slice) {
      it(`Should land "${rule.name}" with ${rule.citation} citation`, () => {
        const block = rulesBlock();
        expect(block, rule.name).toMatch(rule.needle);
        const bullet = ruleBullets().find(b => rule.needle.test(b));
        expect(bullet, `missing bullet: ${rule.name}`).toBeTruthy();
        expect(bullet!, `${rule.name} must cite ${rule.citation}`).toMatch(rule.citation);
      });
    }
  });

  describe("Snapshot lock — rules block (task_17 subtask 17.9)", () => {
    it("Should match the rules-block snapshot (locks future drift)", () => {
      expect(rulesBlock()).toMatchSnapshot();
    });
  });
});
