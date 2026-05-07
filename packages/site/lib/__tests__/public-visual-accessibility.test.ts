import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "../content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".ts", ".tsx"];

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

function hasAccessibleName(tag: string): boolean {
  return (
    hasAttribute(tag, "aria-label") ||
    hasAttribute(tag, "aria-labelledby") ||
    hasAttribute(tag, "title")
  );
}

describe("public visual accessibility", () => {
  it("marks inline SVGs as decorative or intentionally named", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      return openingTags(content, "svg").flatMap(tag => {
        if (hasAttribute(tag, "aria-hidden")) {
          return quotedAttribute(tag, "focusable") === "false"
            ? []
            : [`${relative(siteRoot, file)}: decorative inline <svg> must set focusable="false"`];
        }

        if (quotedAttribute(tag, "role") === "img" && hasAccessibleName(tag)) {
          return [];
        }

        return [
          `${relative(siteRoot, file)}: inline <svg> must be aria-hidden or role="img" with a label`,
        ];
      });
    });

    expect(violations).toEqual([]);
  });

  it("does not put aria-label on generic divs without a role", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const content = readFileSync(file, "utf8");
      return openingTags(content, "div").flatMap(tag => {
        if (!hasAttribute(tag, "aria-label") || hasAttribute(tag, "role")) {
          return [];
        }

        return [`${relative(siteRoot, file)}: <div> with aria-label must declare a role`];
      });
    });

    expect(violations).toEqual([]);
  });
});
