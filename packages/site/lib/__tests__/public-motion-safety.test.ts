import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "../content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".css", ".ts", ".tsx"];

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
      !/\.test\.[cm]?tsx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }

  return files.sort();
}

function lineNumber(content: string, offset: number): number {
  return content.slice(0, offset).split("\n").length;
}

describe("public motion safety", () => {
  it("cleans up browser timers used by public components", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      const relativePath = relative(siteRoot, file);
      const results: string[] = [];

      for (const match of content.matchAll(/\b(?:window\.)?setTimeout\b/g)) {
        if (!/\bclearTimeout\b/.test(content)) {
          results.push(
            `${relativePath}:${lineNumber(content, match.index ?? 0)} setTimeout without clearTimeout`
          );
        }
      }

      for (const match of content.matchAll(/\b(?:window\.)?setInterval\b/g)) {
        if (!/\bclearInterval\b/.test(content)) {
          results.push(
            `${relativePath}:${lineNumber(content, match.index ?? 0)} setInterval without clearInterval`
          );
        }
        if (!/\breducedMotion\b|\buseReducedMotion\b/.test(content)) {
          results.push(
            `${relativePath}:${lineNumber(content, match.index ?? 0)} setInterval without reduced-motion gating`
          );
        }
      }

      return results;
    });

    expect(violations).toEqual([]);
  });

  it("guards IntersectionObserver usage for unsupported browsers and cleanup", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      const relativePath = relative(siteRoot, file);
      return [...content.matchAll(/\bnew\s+IntersectionObserver\b/g)].flatMap(match => {
        const results: string[] = [];
        if (!/typeof\s+IntersectionObserver\s*===\s*["']undefined["']/.test(content)) {
          results.push(
            `${relativePath}:${lineNumber(content, match.index ?? 0)} IntersectionObserver without feature guard`
          );
        }
        if (!/\.disconnect\(\)/.test(content)) {
          results.push(
            `${relativePath}:${lineNumber(content, match.index ?? 0)} IntersectionObserver without disconnect cleanup`
          );
        }
        return results;
      });
    });

    expect(violations).toEqual([]);
  });

  it("pairs custom keyframe animations with reduced-motion handling", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      const relativePath = relative(siteRoot, file);
      return [...content.matchAll(/@keyframes\s+([A-Za-z0-9_-]+)/g)].flatMap(match => {
        if (content.includes("prefers-reduced-motion") || relativePath === "app/global.css") {
          return [];
        }

        return [
          `${relativePath}:${lineNumber(content, match.index ?? 0)} @keyframes without reduced-motion handling`,
        ];
      });
    });

    expect(violations).toEqual([]);
  });
});
