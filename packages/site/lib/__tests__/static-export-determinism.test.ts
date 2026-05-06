import { describe, expect, it } from "vitest";

import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { fencedCodeBlocks, listManualDocs, siteRoot } from "../content-test-utils";

describe("site runtime search configuration", () => {
  it("keeps the site on standard Next.js runtime output instead of static export mode", () => {
    const config = readFileSync(resolve(siteRoot, "next.config.mjs"), "utf8");

    expect(config).not.toContain('output: "export"');
    expect(config).not.toContain("generateBuildId:");
    expect(config).toContain("reactStrictMode: true");
  });

  it("keeps frontmatter-bearing Markdown examples as plain text for stable static output", () => {
    const unstableBlocks = listManualDocs()
      .flatMap(doc =>
        fencedCodeBlocks(doc.content)
          .filter(block => ["markdown", "md"].includes(block.language))
          .filter(block => block.body.trimStart().startsWith("---"))
          .map(block => `${doc.path}: ${block.info}`)
      )
      .sort();

    expect(unstableBlocks).toEqual([]);
  });
});
