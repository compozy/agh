import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const STYLES_PATH = join(__dirname, "../styles.css");
const TOKENS_PATH = join(__dirname, "../../../packages/ui/src/tokens.css");
const WEB_SRC_ROOT = join(__dirname, "..");
const KIT_SRC_ROOT = join(__dirname, "../../../packages/ui/src");
const KIT_BARREL_PATH = join(KIT_SRC_ROOT, "index.ts");
const BADGE_PATH = join(KIT_SRC_ROOT, "components/badge.tsx");
const PAGE_HEADER_PATH = join(KIT_SRC_ROOT, "components/custom/page-header.tsx");

function readFile(path: string): string {
  return readFileSync(path, "utf-8");
}

interface CollectOptions {
  extensions: ReadonlyArray<string>;
}

function collectSourceFiles(dir: string, options: CollectOptions, into: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const stat = statSync(full);
    if (stat.isDirectory()) {
      if (entry === "node_modules" || entry === "dist" || entry === "generated") continue;
      collectSourceFiles(full, options, into);
      continue;
    }
    if (entry.includes(".test.") || entry.includes(".stories.")) continue;
    if (!options.extensions.some(ext => entry.endsWith(ext))) continue;
    into.push(full);
  }
  return into;
}

const SOURCE_FILES = [
  ...collectSourceFiles(WEB_SRC_ROOT, { extensions: [".tsx", ".ts"] }),
  ...collectSourceFiles(KIT_SRC_ROOT, { extensions: [".tsx", ".ts"] }),
];

const NEW_CONTRACT_HEXES = [
  "#0c0b0b",
  "#131211",
  "#1a1918",
  "#1c1b1a",
  "#232220",
  "#1f1e1d",
  "#4a4847",
  "#5fbf85",
  "#d6a647",
  "#e0635a",
  "#8e8eb5",
  "#7a7a80",
  "#ececef",
  "#f6f6f8",
  "#9a9a9f",
  "#76767c",
  "#545458",
  "#e8572a",
  "#d14e25",
  "#f6874f",
  "#17110f",
] as const;

const LEGACY_HEXES = [
  "#0e0e0f",
  "#141312",
  "#1e1c1b",
  "#181716",
  "#2e2c2b",
  "#3c3a39",
  "#353332",
  "#e5e5e7",
  "#8e8e93",
  "#636366",
  "#98989d",
  "#30d158",
  "#ffd60a",
  "#ff453a",
  "#bf5af2",
  "#5ba6ff",
  "#b892ff",
  "#4fd1c5",
  "#e8572b",
] as const;

const BANNED_SHADOW_UTILITIES = [
  "shadow-md",
  "shadow-lg",
  "shadow-xl",
  "shadow-2xl",
  "shadow-xs",
  "shadow-sm",
  "shadow-inner",
  "shadow-none",
] as const;

function escapeForRegex(value: string): string {
  return value.replace(/[/\\^$*+?.()|[\]{}]/g, "\\$&");
}

function findOffenders(pattern: RegExp): string[] {
  const offenders: string[] = [];
  for (const filePath of SOURCE_FILES) {
    const content = readFileSync(filePath, "utf-8");
    if (pattern.test(content)) offenders.push(filePath);
  }
  return offenders;
}

describe("Token contract — packages/ui/src/tokens.css", () => {
  const tokens = readFile(TOKENS_PATH);
  const tokensNormalized = tokens.replace(/\s+/g, " ");

  it.each(NEW_CONTRACT_HEXES)("Should declare the canonical hex %s in the token sheet", hex => {
    expect(tokens).toMatch(new RegExp(escapeForRegex(hex), "i"));
  });

  it("Should pin the --shadow-overlay whitelist value (ADR-003)", () => {
    expect(tokensNormalized).toContain(
      "--shadow-overlay: 0 24px 48px -12px rgba(0, 0, 0, 0.65), 0 0 0 1px rgba(255, 255, 255, 0.045);"
    );
  });

  it("Should pin the --highlight whitelist value (ADR-003)", () => {
    expect(tokensNormalized).toContain("--highlight: inset 0 1px 0 rgba(255, 255, 255, 0.035);");
  });

  it("Should not redeclare --color-line (collapsed into --line)", () => {
    expect(tokens).not.toMatch(/--color-line\s*:/);
  });

  it("Should not declare --radius-diagram literal 12px (consolidated into --radius-lg)", () => {
    expect(tokens).not.toMatch(/--radius-diagram\s*:\s*12px/i);
  });

  it("Should not declare --font-display in the kit token sheet", () => {
    expect(tokens).not.toMatch(/--font-display\b/);
  });

  it("Should not declare --font-wordmark in the kit token sheet", () => {
    expect(tokens).not.toMatch(/--font-wordmark\b/);
  });

  it("Should map shadcn --background to var(--canvas)", () => {
    expect(tokens).toMatch(/--background\s*:\s*var\(--canvas\)/i);
  });

  it("Should map shadcn --primary to var(--accent)", () => {
    expect(tokens).toMatch(/--primary\s*:\s*var\(--accent\)/i);
  });

  it("Should declare --font-sans with Inter Variable", () => {
    expect(tokens).toMatch(/--font-sans:.*"Inter Variable"/);
  });

  it("Should declare --font-mono with JetBrains Mono", () => {
    expect(tokens).toMatch(/--font-mono:.*"JetBrains Mono"/);
  });

  it("Should keep the shared token sheet free of concrete font imports", () => {
    expect(tokens).not.toMatch(/@fontsource-variable\/inter/);
    expect(tokens).not.toMatch(/@fontsource\/jetbrains-mono/);
    expect(tokens).not.toMatch(/@fontsource-variable\/geist/);
    expect(tokens).not.toMatch(/@fontsource\/bricolage-grotesque/);
  });

  it("Should embed the universal-selector reduced-motion guard", () => {
    const block = tokens.match(/@media\s+\(prefers-reduced-motion:\s*reduce\)\s*\{[\s\S]*?\}\s*\}/);
    expect(block, "expected the reduced-motion block to exist").toBeTruthy();
    const blockText = block?.[0] ?? "";
    expect(blockText).toMatch(/\*,\s*\*::before,\s*\*::after/);
    expect(blockText).toMatch(/animation-duration\s*:\s*0\.01ms\s*!important/i);
    expect(blockText).toMatch(/animation-iteration-count\s*:\s*1\s*!important/i);
    expect(blockText).toMatch(/transition-duration\s*:\s*0\.01ms\s*!important/i);
    expect(blockText).toMatch(/scroll-behavior\s*:\s*auto\s*!important/i);
  });

  it("Should not contain oklch() values", () => {
    expect(tokens).not.toMatch(/oklch\(/i);
  });

  it("Should not contain color-mix() expressions", () => {
    expect(tokens).not.toMatch(/color-mix\(/i);
  });

  it("Should not contain linear-gradient or radial-gradient declarations", () => {
    expect(tokens).not.toMatch(/linear-gradient/i);
    expect(tokens).not.toMatch(/radial-gradient/i);
  });

  it("Should not contain --ds-* custom properties", () => {
    expect(tokens).not.toMatch(/--ds-/);
  });
});

describe("Web stylesheet — web/src/styles.css", () => {
  const css = readFile(STYLES_PATH);

  it("Should import tailwindcss before composing app styles", () => {
    expect(css).toMatch(/@import\s+["']tailwindcss["']/);
  });

  it("Should import @agh/ui/tokens.css", () => {
    expect(css).toMatch(/@import\s+["']@agh\/ui\/tokens\.css["']/);
  });

  it("Should import @fontsource/jetbrains-mono weight 400", () => {
    expect(css).toMatch(/@import\s+["']@fontsource\/jetbrains-mono\/400\.css["']/);
  });

  it("Should import @fontsource-variable/inter", () => {
    expect(css).toMatch(/@import\s+["']@fontsource-variable\/inter["']/);
  });

  it("Should register workspace UI sources for Tailwind emission", () => {
    expect(css).toMatch(/@source\s+["']\.\.\/\.\.\/packages\/ui\/src\/\*\*\/\*\.\{ts,tsx\}["']/);
  });

  it("Should not contain a stray @theme inline block (consolidated into tokens.css)", () => {
    expect(css).not.toMatch(/@theme\s+inline/);
  });

  it("Should not contain oklch() values", () => {
    expect(css).not.toMatch(/oklch\(/i);
  });

  it("Should not contain color-mix() expressions", () => {
    expect(css).not.toMatch(/color-mix\(/i);
  });

  it("Should not contain linear-gradient declarations", () => {
    expect(css).not.toMatch(/linear-gradient/i);
  });
});

describe("Source files — no legacy hex or banned utilities", () => {
  it("Should find component files to scan", () => {
    expect(SOURCE_FILES.length).toBeGreaterThan(0);
  });

  it.each(LEGACY_HEXES)(
    "Should not reference the legacy hex %s under web/src or packages/ui/src",
    hex => {
      const offenders = findOffenders(new RegExp(escapeForRegex(hex), "i"));
      expect(offenders).toEqual([]);
    }
  );

  it.each(BANNED_SHADOW_UTILITIES)("Should not use the banned Tailwind utility %s", utility => {
    const offenders = findOffenders(new RegExp(`\\b${utility}\\b`));
    expect(offenders).toEqual([]);
  });

  it("Should not use any backdrop-blur utility class", () => {
    const offenders = findOffenders(/\bbackdrop-blur/);
    expect(offenders).toEqual([]);
  });

  it("Should not use any mix-blend utility class", () => {
    const offenders = findOffenders(/\bmix-blend-/);
    expect(offenders).toEqual([]);
  });

  it("Should not use the font-display utility class", () => {
    const offenders = findOffenders(/\bfont-display\b/);
    expect(offenders).toEqual([]);
  });

  it("Should not reference --ds-* tokens", () => {
    const offenders = findOffenders(/--ds-/);
    expect(offenders).toEqual([]);
  });

  it("Should not reference --font-display variable", () => {
    const offenders = findOffenders(/--font-display\b/);
    expect(offenders).toEqual([]);
  });

  it("Should not reference --font-wordmark variable", () => {
    const offenders = findOffenders(/--font-wordmark\b/);
    expect(offenders).toEqual([]);
  });

  it("Should not contain min(var(--radius-md), Npx) literals", () => {
    const offenders = findOffenders(/min\(var\(--radius-md\)/);
    expect(offenders).toEqual([]);
  });
});

describe("Primitive surface — badge.tsx hard-deleted", () => {
  it("Should not exist on disk", () => {
    expect(existsSync(BADGE_PATH)).toBe(false);
  });

  it("Should not be exported from the @agh/ui barrel", () => {
    const barrel = readFileSync(KIT_BARREL_PATH, "utf-8");
    expect(barrel).not.toMatch(/from\s+["']\.\/components\/badge["']/);
    expect(barrel).not.toMatch(/\bBadge,\s*badgeVariants\b/);
  });
});

describe("Composite surface — page-header.tsx hard-deleted (P4)", () => {
  it("Should not exist on disk", () => {
    expect(existsSync(PAGE_HEADER_PATH)).toBe(false);
  });

  it("Should not be exported from the @agh/ui barrel", () => {
    const barrel = readFileSync(KIT_BARREL_PATH, "utf-8");
    expect(barrel).not.toMatch(/page-header/);
    expect(barrel).not.toMatch(/\bPageHeader\b/);
  });

  it("Should not be imported by any web/ or kit source file", () => {
    const offenders = findOffenders(/from\s+["'@/][^"']*custom\/page-header["']/);
    expect(offenders).toEqual([]);
  });
});
