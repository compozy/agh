import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const DESIGN_PATH = join(__dirname, "../../../../DESIGN.md");
const SKILL_PATH = join(__dirname, "../../../../.agents/skills/agh-design/SKILL.md");
const design = readFileSync(DESIGN_PATH, "utf-8");
const skill = readFileSync(SKILL_PATH, "utf-8");

function section(heading: string): string {
  const start = design.indexOf(`\n## ${heading}\n`);
  if (start === -1) {
    throw new Error(`DESIGN.md section "${heading}" not found`);
  }
  const after = design.indexOf("\n## ", start + 1);
  return after === -1 ? design.slice(start) : design.slice(start, after);
}

function rulesBlock(): string {
  const start = skill.indexOf("**Rules that matter most for this brand");
  const end = skill.indexOf("**Site profile");
  if (start === -1 || end === -1) {
    throw new Error("SKILL.md rules block markers missing");
  }
  return skill.slice(start, end);
}

describe("DESIGN.md PR-1 partial patches (task_05)", () => {
  describe("Section headings", () => {
    it("Should keep §6 as 'Depth & Elevation'", () => {
      expect(design).toMatch(/^## 6\. Depth & Elevation$/m);
    });

    it("Should keep §10 as 'Do's and Don'ts'", () => {
      expect(design).toMatch(/^## 10\. Do's and Don'ts$/m);
    });

    it("Should introduce §11 'Anti-patterns'", () => {
      expect(design).toMatch(/^## 11\. Anti-patterns$/m);
    });

    it("Should renumber Responsive Behavior to §12", () => {
      expect(design).toMatch(/^## 12\. Responsive Behavior$/m);
    });

    it("Should renumber Site Profile to §13", () => {
      expect(design).toMatch(/^## 13\. Site Profile \(`packages\/site` extensions\)$/m);
    });

    it("Should renumber Agent Prompt Guide to §14", () => {
      expect(design).toMatch(/^## 14\. Agent Prompt Guide$/m);
    });
  });

  describe("§6 Depth & Elevation — shadow vocabulary reaffirmation", () => {
    const depth = section("6. Depth & Elevation");

    it("Should reaffirm the two-token shadow whitelist", () => {
      expect(depth).toMatch(/`--shadow-overlay`/);
      expect(depth).toMatch(/`--highlight`/);
    });

    it("Should explicitly ban --shadow-card and --shadow-pop", () => {
      expect(depth).toMatch(/`--shadow-card`/);
      expect(depth).toMatch(/`--shadow-pop`/);
    });

    it("Should cross-link to ADR-001 §4", () => {
      expect(depth).toMatch(/ADR-001 §4/);
    });

    it("Should reference the §11 Anti-patterns section for side-stripe and shadow bans", () => {
      expect(depth).toMatch(/§11 "Anti-patterns"/);
    });

    it("Should swap the selected-rail color to --fg-strong (ADR-003 §3)", () => {
      expect(depth).toMatch(/--fg-strong.*indicator rail.*ADR-003 §3/);
    });
  });

  describe("§10 Do's and Don'ts — PR-1 ban additions + route-identity rule", () => {
    const dosDonts = section("10. Do's and Don'ts");

    it("Should add the side-stripe accent rail ban", () => {
      expect(dosDonts).toMatch(/border-l-\* border-l-\(--accent\)/);
      expect(dosDonts).toMatch(/no-side-stripe-accent/);
    });

    it("Should add the inline glaze rgba ban", () => {
      expect(dosDonts).toMatch(/Don't inline glaze rgba literals/);
      expect(dosDonts).toMatch(/no-design-glaze-rgba/);
    });

    it("Should add the --shadow-card / --shadow-pop ban", () => {
      expect(dosDonts).toMatch(/Don't introduce `--shadow-card` or `--shadow-pop`/);
    });

    it("Should add the backdrop-blur carve-out", () => {
      expect(dosDonts).toMatch(/Don't apply `backdrop-filter: blur\(\.\.\.\)`/);
      expect(dosDonts).toMatch(/\.dialog-scrim/);
      expect(dosDonts).toMatch(/\.sheet-scrim/);
    });

    it("Should add the accent-on-hover ban", () => {
      expect(dosDonts).toMatch(/Don't trigger accent on hover/);
      expect(dosDonts).toMatch(/hover:border-\(--accent\)/);
    });

    it("Should add the outline button variant ban", () => {
      expect(dosDonts).toMatch(/Don't use `<Button variant="outline">`/);
    });

    it("Should add the Loader2 import ban", () => {
      expect(dosDonts).toMatch(/Don't import `Loader2` from `lucide-react`/);
    });

    it("Should add the route-identity Do (Topbar carries page identity)", () => {
      expect(dosDonts).toMatch(/Route identity lives in the global `<Topbar>`/);
      expect(dosDonts).toMatch(/body-side 22 px H1/);
      expect(dosDonts).toMatch(/ADR-003 §2/);
    });
  });

  describe("§11 Anti-patterns — formalized bans", () => {
    const anti = section("11. Anti-patterns");

    it("Should formalize the side-stripe ban", () => {
      expect(anti).toMatch(/### Side-stripe accent rail on cards or rows/);
      expect(anti).toMatch(/border-l-2 border-l-\(--accent\)/);
      expect(anti).toMatch(/ADR-003 §3/);
    });

    it("Should formalize the Eyebrow misuse ban (inline tuple + structural-vs-metric)", () => {
      expect(anti).toMatch(/### Eyebrow misuse/);
      expect(anti).toMatch(/Pattern A — inline tuple/);
      expect(anti).toMatch(/Pattern B — wrong register/);
      expect(anti).toMatch(/KpiCard.*Metric/);
      expect(anti).toMatch(/ADR-002 §1/);
    });

    it("Should formalize the accent-overload ban", () => {
      expect(anti).toMatch(/### Accent overload/);
      expect(anti).toMatch(/one accent target per viewport/);
    });

    it("Should formalize the Section-as-page-head ban", () => {
      expect(anti).toMatch(/### `Section` as page-head/);
      expect(anti).toMatch(/22 px.*H1/);
      expect(anti).toMatch(/ADR-003 §2/);
    });

    it("Should cite ADR-016 §5 enforcement on every anti-pattern", () => {
      const enforcementCount = (anti.match(/ADR-016 §5/g) ?? []).length;
      expect(enforcementCount).toBeGreaterThanOrEqual(3);
    });
  });

  describe("Smoke grep — required strings exist in DESIGN.md after the patch", () => {
    const required = [
      "--shadow-card",
      "border-l-",
      "accent-tint",
      "Topbar",
      "outline",
      "Loader2",
      "--shadow-overlay",
      "--highlight",
    ] as const;

    for (const needle of required) {
      it(`Should contain the string "${needle}" somewhere in DESIGN.md`, () => {
        expect(design).toContain(needle);
      });
    }
  });

  describe("Cross-check — agh-design skill §5.1 bans appear in DESIGN.md §10 or §11", () => {
    const dosDonts = section("10. Do's and Don'ts");
    const anti = section("11. Anti-patterns");
    const combined = `${dosDonts}\n${anti}`;
    const block = rulesBlock();

    const crossChecks: { name: string; skill: RegExp; design: RegExp }[] = [
      {
        name: "Surface glaze rgba ban",
        skill: /Surface glaze tokens are mandatory/,
        design: /Don't inline glaze rgba literals/,
      },
      {
        name: "Modal-scrim alpha + backdrop-blur carve-out",
        skill: /Modal-scrim alpha is `0\.55`/,
        design: /Don't apply `backdrop-filter: blur\(\.\.\.\)`/,
      },
      {
        name: "Two-token shadow vocabulary",
        skill: /Shadow vocabulary is two tokens only/,
        design: /Don't introduce `--shadow-card` or `--shadow-pop`/,
      },
      {
        name: "Loader2 banned in production routes",
        skill: /`Loader2` from `lucide-react` is banned in production routes/,
        design: /Don't import `Loader2` from `lucide-react`/,
      },
    ];

    for (const rule of crossChecks) {
      it(`Should mirror "${rule.name}" between SKILL.md and DESIGN.md §10/§11`, () => {
        expect(block, `${rule.name}: missing in SKILL.md rules block`).toMatch(rule.skill);
        expect(combined, `${rule.name}: missing in DESIGN.md §10/§11`).toMatch(rule.design);
      });
    }
  });

  describe("Snapshot lock — §6 / §10 / §11 ranges", () => {
    it("Should match the §6 snapshot", () => {
      expect(section("6. Depth & Elevation")).toMatchSnapshot();
    });

    it("Should match the §10 snapshot", () => {
      expect(section("10. Do's and Don'ts")).toMatchSnapshot();
    });

    it("Should match the §11 snapshot", () => {
      expect(section("11. Anti-patterns")).toMatchSnapshot();
    });
  });
});

describe("DESIGN.md PR-2 §4 + §5.8 cross-check patches (task_17)", () => {
  const component = section("4. Component Stylings");
  const block = rulesBlock();
  const dosDonts = section("10. Do's and Don'ts");
  const anti = section("11. Anti-patterns");
  const combined = `${dosDonts}\n${anti}`;

  describe("§4 Component Stylings — hex literals removed", () => {
    it("Should contain zero `#RRGGBB` / `#RGB` hex literals in §4", () => {
      const hexMatches = component.match(/#[0-9A-Fa-f]{3}(?:[0-9A-Fa-f]{3})?\b/g) ?? [];
      expect(hexMatches, `unexpected hex literals in §4: ${hexMatches.join(", ")}`).toEqual([]);
    });
  });

  describe("§4 Component Stylings — PR-2 primitive references", () => {
    const required = [
      "<DetailHeader>",
      "<FormSection>",
      "<RunCard>",
      "<KpiCard>",
      "<MonoId>",
      "<Time>",
      "<DescriptionCard>",
      "<ChatToolCard>",
      "<RadioCard>",
      "<CatalogCard>",
      "<PillGroup>",
      "<StatusDot>",
      "<ContextBox>",
      "<QueueHealthSparkline>",
      "<OwnerAvatar>",
    ] as const;

    for (const needle of required) {
      it(`Should reference the PR-2 primitive "${needle}" in §4`, () => {
        expect(component, `§4 must reference ${needle}`).toContain(needle);
      });
    }
  });

  describe("§4 Component Stylings — token-only color references", () => {
    const requiredTokens = [
      "var(--canvas-soft)",
      "var(--accent)",
      "var(--accent-tint)",
      "var(--fg-strong)",
      "var(--muted)",
      "var(--line-soft)",
      "var(--input-fill)",
      "var(--surface-glaze)",
      "var(--btn-default-fill)",
      "var(--badge-fill)",
      "var(--overlay-scrim)",
      "var(--overlay-blur)",
      "var(--highlight)",
      "var(--radius-md)",
      "var(--radius-lg)",
    ] as const;

    for (const needle of requiredTokens) {
      it(`Should anchor the contract through "${needle}" in §4`, () => {
        expect(component, `§4 must consume ${needle}`).toContain(needle);
      });
    }
  });

  describe("Cross-check — agh-design skill §5.8 bans mirror DESIGN.md §10 / §11", () => {
    const crossChecks: { name: string; skill: RegExp; design: RegExp }[] = [
      {
        name: "Anti-pattern: font-mono uppercase tracking-mono inline tuple",
        skill: /Anti-pattern \(compile-fail\): `font-mono uppercase tracking-mono`/,
        design: /font-mono.*uppercase.*tracking-\[0\.06em\]|Eyebrow misuse/,
      },
      {
        name: "Anti-pattern: text-[22px] page-h1 tuple inline",
        skill:
          /Anti-pattern \(compile-fail\): `text-\[22px\] font-medium tracking-\[-0\.026em\]` page-h1 tuple inline/,
        design: /text-\[22px\].*tracking-\[-0\.026em\]/,
      },
      {
        name: "Anti-pattern: border-l-* border-l-(--accent)",
        skill: /Anti-pattern \(compile-fail\): `border-l-\* border-l-\(--accent\)`/,
        design: /border-l-\* border-l-\(--accent\)/,
      },
      {
        name: "Anti-pattern: hover:border-(--accent)",
        skill: /Anti-pattern \(compile-fail\): `hover:border-\(--accent\)`/,
        design: /hover:border-\(--accent\)/,
      },
      {
        name: "Anti-pattern: bg-[rgba(255,255,255,0.0NN)] glaze literals",
        skill: /Anti-pattern \(compile-fail\): `bg-\[rgba\(255,255,255,0\.0NN\)\]` literals/,
        design: /bg-\[rgba\(255,255,255,0\.0NN\)\]/,
      },
      {
        name: "Anti-pattern: Loader2 import under web/src/routes/**",
        skill:
          /Anti-pattern \(compile-fail\): `import \{ Loader2 \} from "lucide-react"` under `web\/src\/routes\/\*\*`/,
        design: /Don't import `Loader2` from `lucide-react`/,
      },
      {
        name: "Anti-pattern: Pill tone=accent next to Button variant=primary",
        skill:
          /Anti-pattern \(review fail\): `<Pill tone="accent">` co-located with `<Button variant="primary">`/,
        design: /Accent overload|one accent target per viewport/,
      },
    ];

    for (const rule of crossChecks) {
      it(`Should mirror "${rule.name}" between SKILL.md §5.8 and DESIGN.md §10 / §11`, () => {
        expect(block, `${rule.name}: missing in SKILL.md rules block`).toMatch(rule.skill);
        expect(combined, `${rule.name}: missing in DESIGN.md §10/§11`).toMatch(rule.design);
      });
    }
  });

  describe("§13 / §14 eyebrow utility list cleanup (PR-2)", () => {
    const siteProfile = section("13. Site Profile (`packages/site` extensions)");

    it("Should reflect the prop-less `<Eyebrow>` + single `eyebrow` `@utility` contract", () => {
      const sharedList = siteProfile.match(
        /What site shares with runtime[\s\S]*?### What site adds/
      );
      expect(sharedList, "§13 runtime-shared section missing").toBeTruthy();
      const sharedText = sharedList?.[0] ?? "";
      expect(sharedText, "§13 must mention the canonical single eyebrow utility").toMatch(
        /single `eyebrow` `@utility`/
      );
      expect(sharedText, "§13 must call out that the legacy tiers are deleted").toMatch(
        /legacy `eyebrow-badge` \/ `eyebrow-micro` tiers are deleted/
      );
    });
  });
});
