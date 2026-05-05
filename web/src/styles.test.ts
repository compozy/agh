import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const STYLES_PATH = join(__dirname, "styles.css");
const TOKENS_PATH = join(__dirname, "../../packages/ui/src/tokens.css");
const SRC_ROOT = __dirname;

function readStyles(): string {
  return readFileSync(STYLES_PATH, "utf-8");
}

function readTokens(): string {
  return readFileSync(TOKENS_PATH, "utf-8");
}

/**
 * Recursively collect all .tsx files under a directory,
 * excluding node_modules, dist, and test files.
 */
function collectTsxFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const stat = statSync(full);
    if (stat.isDirectory()) {
      if (entry === "node_modules" || entry === "dist") continue;
      results.push(...collectTsxFiles(full));
    } else if (
      entry.endsWith(".tsx") &&
      !entry.includes(".test.") &&
      !entry.includes(".stories.")
    ) {
      results.push(full);
    }
  }
  return results;
}

describe("Design Token System — styles.css", () => {
  const css = readStyles();
  const tokens = readTokens();

  it("imports @agh/ui/tokens.css", () => {
    expect(css).toMatch(/@import\s+["']@agh\/ui\/tokens\.css["']/);
  });

  it("imports tailwindcss before composing app styles", () => {
    expect(css).toMatch(/@import\s+["']tailwindcss["']/);
  });

  it("imports the runtime fontsource faces in the web app stylesheet", () => {
    expect(css).toMatch(/@import\s+["']@fontsource-variable\/inter["']/);
    expect(css).toMatch(/@import\s+["']@fontsource\/jetbrains-mono\/500\.css["']/);
    expect(css).toMatch(/@import\s+["']@fontsource\/jetbrains-mono\/600\.css["']/);
  });

  it("registers workspace UI sources for Tailwind emission", () => {
    expect(css).toMatch(/@source\s+["']\.\.\/\.\.\/packages\/ui\/src\/\*\*\/\*\.\{ts,tsx\}["']/);
  });

  it("contains no oklch() values", () => {
    expect(css).not.toMatch(/oklch\(/i);
    expect(tokens).not.toMatch(/oklch\(/i);
  });

  it("contains no box-shadow declarations", () => {
    expect(css).not.toMatch(/box-shadow/i);
    expect(tokens).not.toMatch(/box-shadow/i);
  });

  it("contains no color-mix() expressions", () => {
    expect(css).not.toMatch(/color-mix\(/i);
    expect(tokens).not.toMatch(/color-mix\(/i);
  });

  it("contains no .ds-texture- class definitions", () => {
    expect(tokens).not.toMatch(/\.ds-texture-/);
  });

  it("contains no .ds-panel class definitions", () => {
    expect(tokens).not.toMatch(/\.ds-panel/);
  });

  it("contains no .ds-toolbar-field class definitions", () => {
    expect(tokens).not.toMatch(/\.ds-toolbar-field/);
  });

  it("contains no --ds- custom properties", () => {
    expect(tokens).not.toMatch(/--ds-/);
  });

  it("contains no gradient declarations", () => {
    expect(tokens).not.toMatch(/linear-gradient/);
    expect(tokens).not.toMatch(/radial-gradient/);
  });

  it("defines --color-canvas: #141312", () => {
    expect(tokens).toMatch(/--color-canvas:\s*#141312/i);
  });

  it("defines --color-accent: #E8572A", () => {
    expect(tokens).toMatch(/--color-accent:\s*#E8572A/i);
  });

  it("defines --color-surface: #1E1C1B", () => {
    expect(tokens).toMatch(/--color-surface:\s*#1E1C1B/i);
  });

  it("defines --color-surface-elevated: #2E2C2B", () => {
    expect(tokens).toMatch(/--color-surface-elevated:\s*#2E2C2B/i);
  });

  it("defines --color-text-primary: #E5E5E7", () => {
    expect(tokens).toMatch(/--color-text-primary:\s*#E5E5E7/i);
  });

  it("defines --color-text-secondary: #8E8E93", () => {
    expect(tokens).toMatch(/--color-text-secondary:\s*#8E8E93/i);
  });

  it("defines --color-text-tertiary: #636366", () => {
    expect(tokens).toMatch(/--color-text-tertiary:\s*#636366/i);
  });

  it("defines --color-text-label: #98989D", () => {
    expect(tokens).toMatch(/--color-text-label:\s*#98989D/i);
  });

  it("defines --color-success: #30D158", () => {
    expect(tokens).toMatch(/--color-success:\s*#30D158/i);
  });

  it("defines --color-danger: #FF453A", () => {
    expect(tokens).toMatch(/--color-danger:\s*#FF453A/i);
  });

  it("defines --color-warning: #FFD60A", () => {
    expect(tokens).toMatch(/--color-warning:\s*#FFD60A/i);
  });

  it("defines --color-info: #BF5AF2", () => {
    expect(tokens).toMatch(/--color-info:\s*#BF5AF2/i);
  });

  it("declares --font-sans with Inter Variable", () => {
    expect(tokens).toMatch(/--font-sans:.*"Inter Variable"/);
  });

  it("declares --font-mono with JetBrains Mono", () => {
    expect(tokens).toMatch(/--font-mono:.*"JetBrains Mono"/);
  });

  it("declares --font-display with Playfair Display", () => {
    expect(tokens).toMatch(/--font-display:.*"Playfair Display"/);
  });

  it("declares --font-wordmark with NuixyberNext", () => {
    expect(tokens).toMatch(/--font-wordmark:[\s\S]*?"NuixyberNext"/);
  });

  it("keeps the shared token stylesheet free of concrete font imports", () => {
    expect(tokens).not.toMatch(/@fontsource-variable\/inter/);
    expect(tokens).not.toMatch(/@fontsource\/jetbrains-mono/);
  });

  it("does NOT import @fontsource-variable/geist", () => {
    expect(tokens).not.toMatch(/@fontsource-variable\/geist/);
  });

  it("does NOT import @fontsource/bricolage-grotesque", () => {
    expect(tokens).not.toMatch(/@fontsource\/bricolage-grotesque/);
  });

  it("maps shadcn --primary to the accent token", () => {
    expect(tokens).toMatch(/--primary:\s*var\(--color-accent\)/i);
  });

  it("maps shadcn --background to the canvas token", () => {
    expect(tokens).toMatch(/--background:\s*var\(--color-canvas\)/i);
  });

  it("maps shadcn --border to the divider token", () => {
    expect(tokens).toMatch(/--border:\s*var\(--color-divider\)/i);
  });

  it("sets --radius to 0.5rem", () => {
    expect(tokens).toMatch(/--radius:\s*0\.5rem/);
  });
});

describe("Component files — no legacy token references", () => {
  const componentFiles = collectTsxFiles(SRC_ROOT);

  it("finds component files to check", () => {
    expect(componentFiles.length).toBeGreaterThan(0);
  });

  it("no component file contains var(--ds- references", () => {
    const violations: string[] = [];
    for (const filePath of componentFiles) {
      const content = readFileSync(filePath, "utf-8");
      if (content.includes("var(--ds-")) {
        const relativePath = filePath.replace(SRC_ROOT, "");
        violations.push(relativePath);
      }
    }
    expect(violations).toEqual([]);
  });

  it("no component file uses font-display class", () => {
    const violations: string[] = [];
    for (const filePath of componentFiles) {
      const content = readFileSync(filePath, "utf-8");
      if (/\bfont-display\b/.test(content)) {
        const relativePath = filePath.replace(SRC_ROOT, "");
        violations.push(relativePath);
      }
    }
    expect(violations).toEqual([]);
  });

  it("no component file uses ds-texture-canvas class", () => {
    const violations: string[] = [];
    for (const filePath of componentFiles) {
      const content = readFileSync(filePath, "utf-8");
      if (/\bds-texture-canvas\b/.test(content)) {
        const relativePath = filePath.replace(SRC_ROOT, "");
        violations.push(relativePath);
      }
    }
    expect(violations).toEqual([]);
  });
});
