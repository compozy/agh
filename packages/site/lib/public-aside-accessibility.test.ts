import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "./content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".ts", ".tsx"];
const nonLandmarkRoles = new Set(["note", "none", "presentation"]);

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

function hasAttribute(tag: string, attribute: string): boolean {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`\\b${escapedAttribute}(?:\\s|=|>)`).test(tag);
}

function quotedAttribute(tag: string, attribute: string): string | null {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return (
    tag.match(new RegExp(`\\b${escapedAttribute}=["']([^"']*)["']`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{["']([^"']*)["']\\}`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{\\\`([^\\\`]*)\\\`\\}`))?.[1] ??
    null
  );
}

function openingTags(content: string, tagName: string): string[] {
  return [...content.matchAll(new RegExp(`<${tagName}\\b[^>]*>`, "g"))]
    .filter(match => !isInsideStringLiteral(content, match.index ?? 0))
    .map(match => match[0] ?? "");
}

function isInsideStringLiteral(content: string, offset: number): boolean {
  let quote: "'" | '"' | "`" | null = null;
  let escaped = false;

  for (let index = 0; index < offset; index += 1) {
    const char = content[index];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (char === "\\") {
      escaped = true;
      continue;
    }
    if (quote) {
      if (char === quote) {
        quote = null;
      }
      continue;
    }
    if (char === "'" || char === '"' || char === "`") {
      quote = char;
    }
  }

  return quote !== null;
}

function isNamedComplementaryLandmark(tag: string): boolean {
  const role = quotedAttribute(tag, "role");
  if (role && nonLandmarkRoles.has(role)) {
    return true;
  }

  return (
    hasAttribute(tag, "aria-label") ||
    hasAttribute(tag, "aria-labelledby") ||
    hasAttribute(tag, "title")
  );
}

describe("public aside accessibility", () => {
  it("names complementary aside landmarks or marks note-style asides explicitly", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      return openingTags(content, "aside").flatMap(tag => {
        if (isNamedComplementaryLandmark(tag)) {
          return [];
        }

        return [
          `${relative(
            siteRoot,
            file
          )}: <aside> must have aria-label/aria-labelledby or role="note"`,
        ];
      });
    });

    expect(violations).toEqual([]);
  });
});
