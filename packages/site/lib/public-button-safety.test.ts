import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "./content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".ts", ".tsx"];

type ButtonElement = {
  body: string;
  file: string;
  openingTag: string;
  tagName: "button" | "Button";
};

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

function buttonElements(file: string): ButtonElement[] {
  const content = readFileSync(file, "utf8");
  return [...content.matchAll(/<((?:button|Button))\b([^>]*)>([\s\S]*?)<\/\1>/g)].map(match => ({
    file,
    tagName: (match[1] ?? "button") as "button" | "Button",
    openingTag: `<${match[1] ?? ""}${match[2] ?? ""}>`,
    body: match[3] ?? "",
  }));
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

function hasAttribute(tag: string, attribute: string): boolean {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`\\b${escapedAttribute}(?:\\s|=|>)`).test(tag);
}

function visibleText(content: string): string {
  return content
    .replace(/\{\/\*[\s\S]*?\*\/\}/g, "")
    .replace(/<[^>]+>/g, " ")
    .replace(/\{[^}]*\}/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function hasTextLikeExpression(content: string): boolean {
  return [...content.matchAll(/\{([^{}]+)\}/g)].some(match => {
    const expression = match[1] ?? "";
    return (
      /\b(children|label|title|caption|copyState|state|tab|text|name|kind)\b/i.test(expression) ||
      /["'`][^"'`]+["'`]/.test(expression)
    );
  });
}

describe("public button safety", () => {
  it("keeps public buttons accessible by text or ARIA label", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file =>
      buttonElements(file).flatMap(button => {
        if (visibleText(button.body).length > 0 || hasTextLikeExpression(button.body)) {
          return [];
        }
        if (
          quotedAttribute(button.openingTag, "aria-label") ||
          quotedAttribute(button.openingTag, "aria-labelledby") ||
          hasAttribute(button.openingTag, "aria-label") ||
          hasAttribute(button.openingTag, "aria-labelledby")
        ) {
          return [];
        }

        return [`${relative(siteRoot, button.file)}: button is missing an accessible name`];
      })
    );

    expect(violations).toEqual([]);
  });

  it("declares native button type explicitly", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file =>
      buttonElements(file).flatMap(button => {
        if (button.tagName !== "button" || hasAttribute(button.openingTag, "type")) {
          return [];
        }

        return [`${relative(siteRoot, button.file)}: native button is missing type`];
      })
    );

    expect(violations).toEqual([]);
  });
});
