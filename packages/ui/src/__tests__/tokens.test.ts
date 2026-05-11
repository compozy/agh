import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const TOKENS_PATH = join(__dirname, "../tokens.css");
const tokens = readFileSync(TOKENS_PATH, "utf-8");

function declarationValue(name: string): string | null {
  const re = new RegExp(`(?<![A-Za-z0-9_-])${name}\\s*:\\s*([^;\\n]+);`);
  const match = tokens.match(re);
  return match ? match[1].trim() : null;
}

describe("redesign-v2 token contract (PR-1, task_01)", () => {
  describe("Surface glaze ladder (ADR-001 §6)", () => {
    it("Should resolve --row-hover to rgba(255, 255, 255, 0.022)", () => {
      expect(declarationValue("--row-hover")).toBe("rgba(255, 255, 255, 0.022)");
    });

    it("Should resolve --row-selected to rgba(255, 255, 255, 0.03)", () => {
      expect(declarationValue("--row-selected")).toBe("rgba(255, 255, 255, 0.03)");
    });

    it("Should resolve --surface-glaze to rgba(255, 255, 255, 0.04)", () => {
      expect(declarationValue("--surface-glaze")).toBe("rgba(255, 255, 255, 0.04)");
    });

    it("Should resolve --bar-fill to rgba(255, 255, 255, 0.085)", () => {
      expect(declarationValue("--bar-fill")).toBe("rgba(255, 255, 255, 0.085)");
    });

    it("Should resolve --input-fill to rgba(255, 255, 255, 0.025)", () => {
      expect(declarationValue("--input-fill")).toBe("rgba(255, 255, 255, 0.025)");
    });

    it("Should resolve --btn-default-fill to rgba(255, 255, 255, 0.04)", () => {
      expect(declarationValue("--btn-default-fill")).toBe("rgba(255, 255, 255, 0.04)");
    });

    it("Should resolve --btn-default-hover to rgba(255, 255, 255, 0.07)", () => {
      expect(declarationValue("--btn-default-hover")).toBe("rgba(255, 255, 255, 0.07)");
    });

    it("Should resolve --badge-fill to rgba(255, 255, 255, 0.05)", () => {
      expect(declarationValue("--badge-fill")).toBe("rgba(255, 255, 255, 0.05)");
    });

    it("Should alias --hover to var(--row-hover) (N-45 fix)", () => {
      expect(declarationValue("--hover")).toBe("var(--row-hover)");
    });
  });

  describe("Owner avatar palette (ADR-001 §7)", () => {
    it("Should resolve --avatar-agent-0-bg / fg per the spec", () => {
      expect(declarationValue("--avatar-agent-0-bg")).toBe("rgba(232, 144, 99, 0.18)");
      expect(declarationValue("--avatar-agent-0-fg")).toBe("#f2b895");
    });

    it("Should resolve --avatar-human-0-bg / fg per the spec", () => {
      expect(declarationValue("--avatar-human-0-bg")).toBe("rgba(220, 192, 134, 0.2)");
      expect(declarationValue("--avatar-human-0-fg")).toBe("#e5cc9a");
    });

    it("Should anchor --avatar-system-bg / fg on the warm-dark ramp", () => {
      expect(declarationValue("--avatar-system-bg")).toBe("var(--elevated)");
      expect(declarationValue("--avatar-system-fg")).toBe("var(--subtle)");
    });
  });

  describe("Modal width ladder (ADR-011 §1)", () => {
    it("Should resolve --width-modal-sm to 560px", () => {
      expect(declarationValue("--width-modal-sm")).toBe("560px");
    });

    it("Should resolve --width-modal-md to 720px", () => {
      expect(declarationValue("--width-modal-md")).toBe("720px");
    });

    it("Should resolve --width-modal-lg to 880px", () => {
      expect(declarationValue("--width-modal-lg")).toBe("880px");
    });
  });

  describe("Sizing tokens (ADR-015 §2)", () => {
    it("Should resolve --size-catalog-logo to 1.5rem", () => {
      expect(declarationValue("--size-catalog-logo")).toBe("1.5rem");
    });

    it("Should resolve --size-provider-logo-well to 2.5rem", () => {
      expect(declarationValue("--size-provider-logo-well")).toBe("2.5rem");
    });
  });

  describe("Type ladder (ADR-002 §10)", () => {
    it("Should declare every type-ladder token from ADR-002 §10", () => {
      const expectations: Record<string, string> = {
        "--text-detail-h1": "1.5rem",
        "--text-empty-h1": "1.125rem",
        "--text-modal-title": "0.84375rem",
        "--text-section-head": "0.8125rem",
        "--text-form-input": "0.78125rem",
        "--text-form-label": "0.75rem",
        "--text-form-hint": "0.71875rem",
        "--text-form-required": "0.625rem",
        "--text-metric-value": "1.375rem",
        "--text-kpi-value": "1.75rem",
        "--text-agent-metric": "1rem",
        "--text-rail-avatar": "0.71875rem",
        "--text-ws-name": "0.8125rem",
        "--text-mono-id": "0.65625rem",
      };
      for (const [name, value] of Object.entries(expectations)) {
        expect(declarationValue(name), name).toBe(value);
      }
    });

    it("Should declare every tracking-ladder token from ADR-002 §10", () => {
      const expectations: Record<string, string> = {
        "--tracking-detail-h1": "-0.028em",
        "--tracking-empty-h1": "-0.022em",
        "--tracking-modal-title": "-0.012em",
        "--tracking-section-head": "-0.008em",
        "--tracking-tight": "-0.014em",
        "--tracking-eyebrow": "-0.005em",
        "--tracking-mono-id": "0",
      };
      for (const [name, value] of Object.entries(expectations)) {
        expect(declarationValue(name), name).toBe(value);
      }
    });
  });

  describe("Updates (ADR-001 §1, §3, §8 + ADR-003 §6 + ADR-015 §4)", () => {
    it("Should update --neutral-tint to rgba(150, 150, 155, 0.06)", () => {
      expect(declarationValue("--neutral-tint")).toBe("rgba(150, 150, 155, 0.06)");
    });

    it("Should update --overlay-scrim to rgba(0, 0, 0, 0.55)", () => {
      expect(declarationValue("--overlay-scrim")).toBe("rgba(0, 0, 0, 0.55)");
    });

    it("Should declare --overlay-blur 3px", () => {
      expect(declarationValue("--overlay-blur")).toBe("3px");
    });

    it("Should pin --radius-mono-badge at 4px (sharper chip)", () => {
      const all = tokens.matchAll(/--radius-mono-badge\s*:\s*([^;\n]+);/g);
      const values = Array.from(all, m => m[1].trim());
      expect(values.length).toBeGreaterThan(0);
      for (const value of values) expect(value).toBe("4px");
    });

    it("Should retune --info-tint to rgba(142, 142, 181, 0.12)", () => {
      expect(declarationValue("--info-tint")).toBe("rgba(142, 142, 181, 0.12)");
    });
  });

  describe("Pinned radii at :root (ADR-001 §9)", () => {
    it("Should declare --radius-xs 4px at :root and inside @theme inline", () => {
      const all = Array.from(tokens.matchAll(/--radius-xs\s*:\s*([^;\n]+);/g), m => m[1].trim());
      expect(all).toEqual(["4px", "4px"]);
    });

    it("Should declare --radius 6px at :root", () => {
      expect(declarationValue("--radius")).toBe("6px");
    });
  });

  describe("Font alias (ADR-001 §5)", () => {
    it("Should declare --font-display as a var(--font-sans) alias", () => {
      expect(declarationValue("--font-display")).toBe("var(--font-sans)");
    });
  });

  describe("Body baseline (ADR-002 §6 + §9)", () => {
    it("Should embed the body baseline rule inside @layer base", () => {
      const baseLayer = tokens.match(/@layer\s+base\s*\{[\s\S]*?\n\}/);
      expect(baseLayer, "expected an @layer base block").toBeTruthy();
      const baseText = baseLayer?.[0] ?? "";
      const body = baseText.match(/body\s*\{[\s\S]*?\}/);
      expect(body, "expected a body rule inside @layer base").toBeTruthy();
      const bodyText = body?.[0] ?? "";
      expect(bodyText).toMatch(/font-size\s*:\s*0\.84375rem/);
      expect(bodyText).toMatch(/line-height\s*:\s*1\.5/);
      expect(bodyText).toMatch(/letter-spacing\s*:\s*-0\.006em/);
      expect(bodyText).toMatch(/font-feature-settings\s*:\s*"cv01",\s*"ss03",\s*"cv11"/);
    });
  });

  describe("Reduced-motion guard (ADR-010 §1)", () => {
    it("Should pin every animation/transition to 0.001ms !important", () => {
      const block = tokens.match(
        /@media\s+\(prefers-reduced-motion:\s*reduce\)\s*\{[\s\S]*?\}\s*\}/
      );
      expect(block, "expected a reduced-motion @media block").toBeTruthy();
      const text = block?.[0] ?? "";
      expect(text).toMatch(/\*,\s*\*::before,\s*\*::after/);
      expect(text).toMatch(/animation-duration\s*:\s*0\.001ms\s*!important/);
      expect(text).toMatch(/animation-delay\s*:\s*0\.001ms\s*!important/);
      expect(text).toMatch(/transition-duration\s*:\s*0\.001ms\s*!important/);
    });
  });

  describe("Staged deletion comment block (TechSpec §Data Models — Deletions)", () => {
    it("Should record every staged deletion as completed (no pending entries remain)", () => {
      expect(tokens).not.toMatch(/task_02 \/ PR-1: --overlay-selection/);
      expect(tokens).not.toMatch(/task_16 \/ PR-2: --ease-in-out/);
      expect(tokens).not.toMatch(/task_29 \/ PR-4: --tracking-badge/);
      expect(tokens).toMatch(/staged token deletions are now complete/i);
    });
  });

  describe("Selection contract (ADR-001 §1, task_02 / PR-1)", () => {
    it("Should NOT declare --overlay-selection anywhere in tokens.css", () => {
      expect(tokens).not.toMatch(/--overlay-selection\s*:/);
    });

    it("Should ship a ::selection rule that resolves var(--accent-tint)", () => {
      const block = tokens.match(/::selection\s*\{[\s\S]*?\}/);
      expect(block, "expected a ::selection rule").toBeTruthy();
      const blockText = block?.[0] ?? "";
      expect(blockText).toMatch(/background\s*:\s*var\(--accent-tint\)/);
      expect(blockText).toMatch(/color\s*:\s*var\(--fg-strong\)/);
    });
  });
});
