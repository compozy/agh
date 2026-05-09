import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "../content-test-utils";

const checkedRoots = ["app", "components", "content", "lib"].map(root => resolve(siteRoot, root));
const urlCheckedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".md", ".mdx", ".ts", ".tsx"];
const localHttpHosts = new Set(["localhost", "127.0.0.1", "::1"]);
const httpIdentifierAllowlist = new Set(["http://www.w3.org/2005/Atom"]);

type LinkElement = {
  openingTag: string;
  body: string;
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
      !/\.test\.[cm]?[tj]sx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }

  return files.sort();
}

function quotedAttribute(tag: string, attribute: string): string | null {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return (
    tag.match(new RegExp(`\\b${escapedAttribute}=["']([^"']+)["']`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{["']([^"']+)["']\\}`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{\\\`([^\\\`]+)\\\`\\}`))?.[1] ??
    null
  );
}

function openingLinkTags(content: string): string[] {
  return [...content.matchAll(/<(?:a|Link)\b[\s\S]*?>/g)].map(match => match[0]);
}

function externalUrls(content: string): string[] {
  return [...content.matchAll(/\bhttps?:\/\/[^\s<>"'`)]+/g)]
    .map(match => match[0].replace(/[.,;:]+$/g, ""))
    .sort();
}

function parseConcreteUrl(url: string): URL | null {
  if (url.includes("...") || url.includes("...")) {
    return null;
  }
  return new URL(url);
}

function linkElements(content: string): LinkElement[] {
  return [...content.matchAll(/<((?:a|Link))\b([^>]*)>([\s\S]*?)<\/\1>/g)].map(match => ({
    openingTag: `<${match[1] ?? ""}${match[2] ?? ""}>`,
    body: match[3] ?? "",
  }));
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
      /\b(children|label|title|version|name|text|date|category)\b/i.test(expression) ||
      /["'`][^"'`]+["'`]/.test(expression)
    );
  });
}

describe("public link safety", () => {
  it("uses HTTPS for external URLs in public source files", () => {
    const violations = urlCheckedRoots.flatMap(listSourceFiles).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      return externalUrls(readFileSync(file, "utf8")).flatMap(url => {
        if (httpIdentifierAllowlist.has(url)) {
          return [];
        }

        const parsed = parseConcreteUrl(url);
        if (!parsed) {
          return [];
        }
        if (parsed.protocol !== "http:") {
          return [];
        }
        if (localHttpHosts.has(parsed.hostname)) {
          return [];
        }

        return [`${relativePath}: ${url} uses http:// outside a local example`];
      });
    });

    expect(violations).toEqual([]);
  });

  it("pairs links that open new tabs with noopener and noreferrer", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      return openingLinkTags(readFileSync(file, "utf8")).flatMap(tag => {
        if (quotedAttribute(tag, "target") !== "_blank") {
          return [];
        }

        const relTokens = new Set((quotedAttribute(tag, "rel") ?? "").split(/\s+/).filter(Boolean));
        const missing = ["noopener", "noreferrer"].filter(token => !relTokens.has(token));
        return missing.length > 0
          ? [`${relativePath}: target="_blank" missing rel token(s): ${missing.join(", ")}`]
          : [];
      });
    });

    expect(violations).toEqual([]);
  });

  it("keeps icon-only links accessible", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file => {
      const relativePath = relative(siteRoot, file);
      return linkElements(readFileSync(file, "utf8")).flatMap(link => {
        if (visibleText(link.body).length > 0 || hasTextLikeExpression(link.body)) {
          return [];
        }
        if (
          quotedAttribute(link.openingTag, "aria-label") ||
          quotedAttribute(link.openingTag, "aria-labelledby")
        ) {
          return [];
        }

        return [`${relativePath}: icon-only link is missing an accessible name`];
      });
    });

    expect(violations).toEqual([]);
  });
});
