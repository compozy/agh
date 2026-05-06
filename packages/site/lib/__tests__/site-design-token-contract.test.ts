import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { publicRoot, siteRoot } from "../content-test-utils";

const repoRoot = resolve(siteRoot, "../..");
const checkedRoots = ["app", "components", "content", "lib"].map(root => resolve(siteRoot, root));
const checkedPublicFiles = ["favicon.svg", "site.webmanifest"].map(file =>
  resolve(publicRoot, file)
);
const generatedPathSegments = [`content${"/"}runtime${"/"}cli-reference${"/"}`];
const hexColorPattern = /#[0-9a-fA-F]{6}(?:[0-9a-fA-F]{2})?\b/g;

function listFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      if (entry === "__tests__") {
        continue;
      }
      files.push(...listFiles(fullPath));
      continue;
    }
    if (
      stat.isFile() &&
      /\.(?:css|mdx?|tsx?|json|svg)$/.test(fullPath) &&
      !/\.test\.[cm]?[tj]sx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }
  return files;
}

function sourceFiles(): string[] {
  return [...checkedRoots.flatMap(root => listFiles(root)), ...checkedPublicFiles]
    .filter(file => !generatedPathSegments.some(segment => file.includes(segment)))
    .sort((left, right) => left.localeCompare(right));
}

function designTokenColors(): Set<string> {
  const tokens = readFileSync(resolve(repoRoot, "packages/ui/src/tokens.css"), "utf8");
  return new Set([...tokens.matchAll(hexColorPattern)].map(match => match[0].toLowerCase()));
}

describe("site design token contract", () => {
  it("keeps hardcoded hex colors inside the canonical AGH token palette", () => {
    const allowedColors = designTokenColors();
    const violations = sourceFiles().flatMap(file => {
      const content = readFileSync(file, "utf8");
      const colors = [...new Set([...content.matchAll(hexColorPattern)].map(match => match[0]))];

      return colors
        .filter(color => !allowedColors.has(color.toLowerCase()))
        .map(color => `${relative(siteRoot, file)} -> ${color}`);
    });

    expect(violations).toEqual([]);
  });
});
