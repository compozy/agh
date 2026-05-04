import { describe, expect, it } from "vitest";

import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { fencedCodeBlocks, listManualDocs, siteRoot } from "./content-test-utils";

describe("site static export determinism", () => {
  it("uses a deterministic Next build id for byte-stable static exports", async () => {
    const buildIDModule = (await import("./static-export-build-id.mjs")) as {
      STATIC_EXPORT_BUILD_ID: string;
    };
    const config = readFileSync(resolve(siteRoot, "next.config.mjs"), "utf8");

    expect(buildIDModule.STATIC_EXPORT_BUILD_ID).toBe("agh-network-static");
    expect(config).toContain('output: "export"');
    expect(config).toContain("generateBuildId: async () => STATIC_EXPORT_BUILD_ID");
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
