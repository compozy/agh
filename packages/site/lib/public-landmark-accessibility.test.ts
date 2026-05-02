import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const appRoot = resolve(siteRoot, "app");

const mainContentTargets = [
  "app/(home)/page.tsx",
  "app/blog/layout.tsx",
  "app/changelog/layout.tsx",
  "app/error.tsx",
  "app/not-found.tsx",
  "app/protocol/[[...slug]]/page.tsx",
  "app/runtime/[[...slug]]/page.tsx",
];

function readSiteFile(relativePath: string): string {
  return readFileSync(resolve(siteRoot, relativePath), "utf8");
}

function listTsxFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listTsxFiles(fullPath));
      continue;
    }
    if (stat.isFile() && fullPath.endsWith(".tsx")) {
      files.push(fullPath);
    }
  }
  return files.sort((left, right) => left.localeCompare(right));
}

function openingMainTags(source: string): string[] {
  return Array.from(source.matchAll(/<main\b[^>]*>/g), match => match[0] ?? "");
}

describe("public landmark accessibility", () => {
  it("exposes a keyboard skip link to the primary content", () => {
    const layout = readSiteFile("app/layout.tsx");

    expect(layout).toContain('href="#main-content"');
    expect(layout).toContain("Skip to content");
    expect(layout).toContain("sr-only");
    expect(layout).toContain("focus:not-sr-only");
    expect(layout).toContain("focus:bg-(--color-surface-elevated)");
  });

  it("keeps every public route family reachable from the skip link", () => {
    const missingTargets = mainContentTargets
      .filter(path => !readSiteFile(path).includes('id="main-content"'))
      .map(path => `${path} is missing id="main-content"`);

    expect(missingTargets).toEqual([]);
  });

  it("does not add unnamed main landmarks in app routes", () => {
    const unnamedMainTags = listTsxFiles(appRoot).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      return openingMainTags(readFileSync(file, "utf8"))
        .filter(tag => !/\bid=["']main-content["']/.test(tag))
        .map(tag => `${relativePath}: ${tag}`);
    });

    expect(unnamedMainTags).toEqual([]);
  });
});
