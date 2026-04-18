import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, resolve } from "node:path";

import { describe, expect, it } from "vitest";

const PACKAGE_ROOT = resolve(__dirname, "..");
const README_PATH = resolve(PACKAGE_ROOT, "README.md");
const INDEX_PATH = resolve(PACKAGE_ROOT, "src/index.ts");

const README_CONTENT = readFileSync(README_PATH, "utf8");
const INDEX_CONTENT = readFileSync(INDEX_PATH, "utf8");

function collectMarkdownLinks(markdown: string): Array<{ label: string; target: string }> {
  const results: Array<{ label: string; target: string }> = [];
  // Markdown link syntax: [label](target) — skip images, code fences, and inline code.
  const linkPattern = /(?<!!)\[([^\]\n]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
  let insideCodeFence = false;
  for (const rawLine of markdown.split("\n")) {
    if (rawLine.startsWith("```")) {
      insideCodeFence = !insideCodeFence;
      continue;
    }
    if (insideCodeFence) continue;
    const stripped = rawLine.replace(/`[^`\n]*`/g, "");
    let match: RegExpExecArray | null;
    linkPattern.lastIndex = 0;
    while ((match = linkPattern.exec(stripped)) !== null) {
      results.push({ label: match[1], target: match[2] });
    }
  }
  return results;
}

function collectExportedNames(indexSource: string): Set<string> {
  const names = new Set<string>();
  const blockPattern = /export\s+(?:type\s+)?\{([^}]+)\}\s+from\s+["'][^"']+["'];?/g;
  let match: RegExpExecArray | null;
  while ((match = blockPattern.exec(indexSource)) !== null) {
    const inside = match[1];
    const parts = inside.split(",");
    for (const part of parts) {
      // Strip `type` modifier and whitespace, keep the exported identifier (before `as`).
      const trimmed = part
        .replace(/\/\/.*$/g, "")
        .replace(/\/\*[\s\S]*?\*\//g, "")
        .trim();
      if (!trimmed) continue;
      const withoutType = trimmed.replace(/^type\s+/, "");
      const identifier = withoutType.split(/\s+as\s+/)[0].trim();
      if (identifier) names.add(identifier);
    }
  }
  return names;
}

describe("packages/ui README", () => {
  it("exists under the package root", () => {
    expect(existsSync(README_PATH)).toBe(true);
    expect(statSync(README_PATH).isFile()).toBe(true);
  });

  it("stays under 500 lines", () => {
    const lineCount = README_CONTENT.split("\n").length;
    expect(lineCount).toBeLessThanOrEqual(500);
  });

  it("resolves every relative markdown link", () => {
    const readmeDir = dirname(README_PATH);
    const links = collectMarkdownLinks(README_CONTENT);
    expect(links.length).toBeGreaterThan(0);

    const broken: string[] = [];
    for (const { target } of links) {
      // Skip http(s), mailto, and pure anchor links.
      if (/^(https?:|mailto:|tel:)/.test(target)) continue;
      if (target.startsWith("#")) continue;
      const [rawPath] = target.split("#");
      if (!rawPath) continue;
      const absolute = resolve(readmeDir, rawPath);
      if (!existsSync(absolute)) {
        broken.push(`${target} → ${absolute}`);
      }
    }
    expect(broken).toEqual([]);
  });

  it("mentions every exported identifier from src/index.ts", () => {
    const exportedNames = collectExportedNames(INDEX_CONTENT);
    expect(exportedNames.size).toBeGreaterThan(0);

    const missing: string[] = [];
    for (const name of exportedNames) {
      const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      const pattern = new RegExp(`\\b${escaped}\\b`);
      if (!pattern.test(README_CONTENT)) {
        missing.push(name);
      }
    }
    expect(missing).toEqual([]);
  });

  it("keeps the section heading contract stable", () => {
    const headings: string[] = [];
    let insideCodeFence = false;
    for (const rawLine of README_CONTENT.split("\n")) {
      const line = rawLine.replace(/\s+$/, "");
      if (line.startsWith("```")) {
        insideCodeFence = !insideCodeFence;
        continue;
      }
      if (insideCodeFence) continue;
      if (/^#{1,3}\s+/.test(line)) headings.push(line);
    }

    expect(headings).toMatchInlineSnapshot(`
      [
        "# @agh/ui",
        "## Canonical references",
        "## Architecture decisions",
        "## When to add a primitive here vs. in \`web/\`",
        "## Primitive inventory",
        "### Foundations",
        "### Structural",
        "### Form",
        "### Feedback",
        "### Chat",
        "## UIProvider wiring",
        "### Reduced-motion gotchas",
        "## Motion vs. CSS decision rules",
        "## Story contribution rules",
        "## Playwright snapshot workflow",
        "### Generating baselines",
        "### Updating baselines (intentional drift)",
        "### Reviewing a failing snapshot",
        "### Per-platform baselines + CI",
        "### CI gate expectations",
        "## Anti-patterns",
        "## Quick reference",
      ]
    `);
  });
});
