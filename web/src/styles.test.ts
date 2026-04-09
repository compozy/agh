import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const STYLES_PATH = join(__dirname, "styles.css");
const SRC_ROOT = __dirname;

function readStyles(): string {
  return readFileSync(STYLES_PATH, "utf-8");
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

  it("contains no oklch() values", () => {
    expect(css).not.toMatch(/oklch\(/i);
  });

  it("contains no box-shadow declarations", () => {
    expect(css).not.toMatch(/box-shadow/i);
  });

  it("contains no color-mix() expressions", () => {
    expect(css).not.toMatch(/color-mix\(/i);
  });

  it("contains no .ds-texture- class definitions", () => {
    expect(css).not.toMatch(/\.ds-texture-/);
  });

  it("contains no .ds-panel class definitions", () => {
    expect(css).not.toMatch(/\.ds-panel/);
  });

  it("contains no .ds-toolbar-field class definitions", () => {
    expect(css).not.toMatch(/\.ds-toolbar-field/);
  });

  it("contains no --ds- custom properties", () => {
    expect(css).not.toMatch(/--ds-/);
  });

  it("contains no gradient declarations", () => {
    expect(css).not.toMatch(/linear-gradient/);
    expect(css).not.toMatch(/radial-gradient/);
  });

  it("defines --color-canvas: #121212", () => {
    expect(css).toMatch(/--color-canvas:\s*#121212/);
  });

  it("defines --color-accent: #E8572A", () => {
    expect(css).toMatch(/--color-accent:\s*#E8572A/i);
  });

  it("defines --color-surface: #1C1C1E", () => {
    expect(css).toMatch(/--color-surface:\s*#1C1C1E/i);
  });

  it("defines --color-surface-elevated: #2C2C2E", () => {
    expect(css).toMatch(/--color-surface-elevated:\s*#2C2C2E/i);
  });

  it("defines --color-text-primary: #E5E5E7", () => {
    expect(css).toMatch(/--color-text-primary:\s*#E5E5E7/i);
  });

  it("defines --color-text-secondary: #8E8E93", () => {
    expect(css).toMatch(/--color-text-secondary:\s*#8E8E93/i);
  });

  it("defines --color-text-tertiary: #636366", () => {
    expect(css).toMatch(/--color-text-tertiary:\s*#636366/i);
  });

  it("defines --color-text-label: #98989D", () => {
    expect(css).toMatch(/--color-text-label:\s*#98989D/i);
  });

  it("defines --color-success: #30D158", () => {
    expect(css).toMatch(/--color-success:\s*#30D158/i);
  });

  it("defines --color-danger: #FF453A", () => {
    expect(css).toMatch(/--color-danger:\s*#FF453A/i);
  });

  it("defines --color-warning: #FFD60A", () => {
    expect(css).toMatch(/--color-warning:\s*#FFD60A/i);
  });

  it("defines --color-info: #BF5AF2", () => {
    expect(css).toMatch(/--color-info:\s*#BF5AF2/i);
  });

  it("declares --font-sans with Inter Variable", () => {
    expect(css).toMatch(/--font-sans:.*"Inter Variable"/);
  });

  it("declares --font-mono with JetBrains Mono", () => {
    expect(css).toMatch(/--font-mono:.*"JetBrains Mono"/);
  });

  it("does NOT declare --font-display", () => {
    expect(css).not.toMatch(/--font-display/);
  });

  it("imports @fontsource-variable/inter", () => {
    expect(css).toMatch(/@import\s+["']@fontsource-variable\/inter["']/);
  });

  it("imports @fontsource/jetbrains-mono", () => {
    expect(css).toMatch(/@import\s+["']@fontsource\/jetbrains-mono/);
  });

  it("does NOT import @fontsource-variable/geist", () => {
    expect(css).not.toMatch(/@fontsource-variable\/geist/);
  });

  it("does NOT import @fontsource/bricolage-grotesque", () => {
    expect(css).not.toMatch(/@fontsource\/bricolage-grotesque/);
  });

  it("maps shadcn --primary to the accent token", () => {
    expect(css).toMatch(/--primary:\s*var\(--color-accent\)/i);
  });

  it("maps shadcn --background to the canvas token", () => {
    expect(css).toMatch(/--background:\s*var\(--color-canvas\)/i);
  });

  it("maps shadcn --border to the divider token", () => {
    expect(css).toMatch(/--border:\s*var\(--color-divider\)/i);
  });

  it("sets --radius to 0.5rem", () => {
    expect(css).toMatch(/--radius:\s*0\.5rem/);
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
