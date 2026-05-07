import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const coreRoot = resolve(siteRoot, "content/runtime/core");
const minimumUsefulWords = 150;

type HubPage = {
  path: string;
  content: string;
};

type CoreDirectoryEntry = {
  isDirectory(): boolean;
  name: string;
};

function runtimeCoreHubIndexPaths(entries: CoreDirectoryEntry[]): string[] {
  return entries
    .filter(entry => entry.isDirectory())
    .map(entry => resolve(coreRoot, entry.name, "index.mdx"))
    .filter(path => statSync(path, { throwIfNoEntry: false })?.isFile() === true)
    .sort((left, right) => left.localeCompare(right));
}

function runtimeCoreHubPages(): HubPage[] {
  return runtimeCoreHubIndexPaths(readdirSync(coreRoot, { withFileTypes: true })).map(path => ({
    path: relative(siteRoot, path),
    content: readFileSync(path, "utf8"),
  }));
}

function bodyText(content: string): string {
  return content
    .replace(/^---[\s\S]*?---\s*/m, "")
    .replace(/```[\s\S]*?```/g, "")
    .replace(/<[^>]+>/g, " ");
}

function wordCount(content: string): number {
  return bodyText(content).split(/\s+/).filter(Boolean).length;
}

describe("runtime core hub quality", () => {
  it("ignores top-level metadata files when finding section hubs", () => {
    const paths = runtimeCoreHubIndexPaths([
      { isDirectory: () => false, name: "meta.json" },
      { isDirectory: () => true, name: "agents" },
    ]).map(path => relative(siteRoot, path));

    expect(paths).toEqual(["content/runtime/core/agents/index.mdx"]);
  });

  it("keeps section hubs above a useful orientation floor", () => {
    const thinPages = runtimeCoreHubPages()
      .map(page => ({ ...page, words: wordCount(page.content) }))
      .filter(page => page.words < minimumUsefulWords)
      .map(page => `${page.path}: ${page.words} words`);

    expect(thinPages).toEqual([]);
  });
});
