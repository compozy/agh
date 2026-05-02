import { describe, expect, it } from "vitest";
import { listManualDocs } from "./content-test-utils";

function stripFrontmatterAndCode(content: string): string {
  return content.replace(/^---\n[\s\S]*?\n---/, "").replace(/```[\s\S]*?```/g, "");
}

function hasInternalLink(content: string): boolean {
  const stripped = stripFrontmatterAndCode(content);
  return (
    /\]\(\/(?:runtime|protocol)\//.test(stripped) ||
    /\bhref=["']\/(?:runtime|protocol)\//.test(stripped)
  );
}

describe("content related navigation", () => {
  it("keeps every manual runtime and protocol page connected to another docs page", () => {
    const isolatedDocs = listManualDocs(["runtime/", "protocol/"])
      .filter(doc => !hasInternalLink(doc.content))
      .map(doc => doc.path);

    expect(isolatedDocs).toEqual([]);
  });
});
