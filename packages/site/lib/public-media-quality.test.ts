import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { publicRoot, siteRoot } from "./content-test-utils";

const checkedRoots = ["app", "components", "lib"].map(root => resolve(siteRoot, root));
const ignoredPathSegments = ["/.source/", "/.velite/", "/node_modules/"];
const sourceExtensions = [".ts", ".tsx"];

type MediaTag = {
  file: string;
  kind: "img" | "Image";
  tag: string;
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

function mediaTags(file: string): MediaTag[] {
  return [...readFileSync(file, "utf8").matchAll(/<(img|Image)\b[\s\S]*?>/g)].map(match => ({
    file,
    kind: (match[1] ?? "img") as "img" | "Image",
    tag: match[0],
  }));
}

function attributeValue(tag: string, attribute: string): string | null {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return (
    tag.match(new RegExp(`\\b${escapedAttribute}=["']([^"']*)["']`))?.[1] ??
    tag.match(new RegExp(`\\b${escapedAttribute}=\\{["']([^"']*)["']\\}`))?.[1] ??
    null
  );
}

function hasAttribute(tag: string, attribute: string): boolean {
  const escapedAttribute = attribute.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`\\b${escapedAttribute}(?:\\s|=|>)`).test(tag);
}

function publicAssetExists(src: string): boolean {
  return existsSync(resolve(publicRoot, src.replace(/^\//, "")));
}

function staticSrcIssue(src: string | null): string | null {
  if (!src || src.includes("{")) {
    return null;
  }
  if (!src.startsWith("/")) {
    return `non-local src ${src}`;
  }
  if (!publicAssetExists(src)) {
    return `missing public asset ${src}`;
  }
  return null;
}

function staticAltIssue(alt: string | null): string | null {
  if (alt === null) {
    return null;
  }
  if (alt.length === 0) {
    return "empty alt";
  }
  if (alt.length < 32) {
    return "alt too short";
  }
  if (alt.length > 220) {
    return "alt too long";
  }
  return null;
}

describe("public media quality", () => {
  it("keeps plain img tags accessible, local, and stable during rendering", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file =>
      mediaTags(file)
        .filter(media => media.kind === "img")
        .flatMap(media => {
          const issues: string[] = [];
          const src = attributeValue(media.tag, "src");
          const alt = attributeValue(media.tag, "alt");
          const loading = attributeValue(media.tag, "loading");
          const decoding = attributeValue(media.tag, "decoding");
          const srcIssue = staticSrcIssue(src);
          const altIssue = staticAltIssue(alt);

          if (!hasAttribute(media.tag, "src")) {
            issues.push("missing src");
          } else if (srcIssue) {
            issues.push(srcIssue);
          }
          if (!hasAttribute(media.tag, "alt")) {
            issues.push("missing alt");
          } else if (altIssue) {
            issues.push(altIssue);
          }
          if (!loading) {
            issues.push("missing loading strategy");
          }
          if (decoding !== "async") {
            issues.push("missing async decoding");
          }

          return issues.map(issue => `${relative(siteRoot, media.file)}: ${issue}`);
        })
    );

    expect(violations).toEqual([]);
  });

  it("keeps Next Image usage dimensioned and named", () => {
    const violations = checkedRoots.flatMap(listSourceFiles).flatMap(file =>
      mediaTags(file)
        .filter(media => media.kind === "Image")
        .flatMap(media => {
          const issues: string[] = [];
          const src = attributeValue(media.tag, "src");
          const alt = attributeValue(media.tag, "alt");
          const srcIssue = staticSrcIssue(src);
          const altIssue = staticAltIssue(alt);
          const hasDimensions =
            hasAttribute(media.tag, "fill") ||
            (hasAttribute(media.tag, "width") && hasAttribute(media.tag, "height"));

          if (!hasAttribute(media.tag, "src")) {
            issues.push("missing src");
          } else if (srcIssue) {
            issues.push(srcIssue);
          }
          if (!hasAttribute(media.tag, "alt")) {
            issues.push("missing alt");
          } else if (altIssue) {
            issues.push(altIssue);
          }
          if (!hasDimensions) {
            issues.push("missing dimensions");
          }
          if (!hasAttribute(media.tag, "sizes")) {
            issues.push("missing responsive sizes");
          }

          return issues.map(issue => `${relative(siteRoot, media.file)}: ${issue}`);
        })
    );

    expect(violations).toEqual([]);
  });
});
