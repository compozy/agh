import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";

import { siteRoot } from "./content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const sourceExtensions = [".ts", ".tsx"];
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];

function listSourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const normalizedPath = fullPath.replaceAll("\\", "/");
    if (ignoredPathSegments.some(segment => normalizedPath.includes(segment))) {
      continue;
    }

    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listSourceFiles(fullPath));
      continue;
    }

    if (
      stat.isFile() &&
      sourceExtensions.some(extension => fullPath.endsWith(extension)) &&
      !/\.test\.[cm]?[tj]sx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }

  return files.sort();
}

describe("public error handling", () => {
  it("does not silence caught errors with explicit void discards", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      const content = readFileSync(file, "utf8");

      return [...content.matchAll(/catch\s*\(([^)]+)\)\s*\{[\s\S]*?\bvoid\s+\1\s*;/g)].map(
        match =>
          `${relativePath}: replace ignored ${match[1]} catch binding with catch {} or handle it`
      );
    });

    expect(violations).toEqual([]);
  });
});
