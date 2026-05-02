import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { listManualDocs, mdxAttribute, publicRoot } from "./content-test-utils";

function imgTags(content: string): string[] {
  return [...content.matchAll(/<img\s+[\s\S]*?\/>/g)].map(match => match[0]);
}

function markdownImages(content: string): string[] {
  return [...content.matchAll(/!\[[^\]]*]\([^)]*\)/g)].map(match => match[0]);
}

function publicAssetExists(src: string): boolean {
  return existsSync(resolve(publicRoot, src.replace(/^\//, "")));
}

describe("manual content media quality", () => {
  it("uses explicit img tags instead of Markdown image shorthand", () => {
    const shorthandImages = listManualDocs().flatMap(doc =>
      markdownImages(doc.content).map(image => `${doc.path}: replace Markdown image ${image}`)
    );

    expect(shorthandImages).toEqual([]);
  });

  it("keeps manual images accessible, local, and stable during docs rendering", () => {
    const violations = listManualDocs().flatMap(doc =>
      imgTags(doc.content).flatMap(tag => {
        const issues: string[] = [];
        const src = mdxAttribute(tag, "src");
        const alt = mdxAttribute(tag, "alt");
        const loading = mdxAttribute(tag, "loading");
        const decoding = mdxAttribute(tag, "decoding");

        if (!src) {
          issues.push("missing src");
        } else if (!src.startsWith("/")) {
          issues.push(`non-local src ${src}`);
        } else if (!publicAssetExists(src)) {
          issues.push(`missing public asset ${src}`);
        }

        if (!alt) {
          issues.push("missing alt");
        } else if (alt.length < 40) {
          issues.push("alt too short");
        } else if (alt.length > 220) {
          issues.push("alt too long");
        }

        if (!loading) {
          issues.push("missing loading strategy");
        }
        if (decoding !== "async") {
          issues.push("missing async decoding");
        }

        return issues.map(issue => `${doc.path}: ${issue}`);
      })
    );

    expect(violations).toEqual([]);
  });
});
