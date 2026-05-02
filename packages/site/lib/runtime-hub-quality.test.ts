import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const coreRoot = resolve(siteRoot, "content/runtime/core");
const minimumUsefulWords = 150;

type HubPage = {
  path: string;
  content: string;
};

function runtimeCoreHubPages(): HubPage[] {
  return readdirSync(coreRoot)
    .map(entry => resolve(coreRoot, entry, "index.mdx"))
    .filter(path => statSync(path, { throwIfNoEntry: false })?.isFile() === true)
    .map(path => ({
      path: relative(siteRoot, path),
      content: readFileSync(path, "utf8"),
    }))
    .sort((left, right) => left.path.localeCompare(right.path));
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
  it("keeps section hubs above a useful orientation floor", () => {
    const thinPages = runtimeCoreHubPages()
      .map(page => ({ ...page, words: wordCount(page.content) }))
      .filter(page => page.words < minimumUsefulWords)
      .map(page => `${page.path}: ${page.words} words`);

    expect(thinPages).toEqual([]);
  });
});
