import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const contentRoot = resolve(siteRoot, "content");
const checkedRoots = [resolve(contentRoot, "runtime"), resolve(contentRoot, "protocol")];

type MetaFile = {
  dir: string;
  path: string;
  pages: string[];
};

function listFiles(dir: string, filename: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...listFiles(fullPath, filename));
      continue;
    }
    if (stat.isFile() && entry === filename) {
      files.push(fullPath);
    }
  }
  return files.sort();
}

function readMeta(path: string): MetaFile {
  const raw = JSON.parse(readFileSync(path, "utf8")) as { pages?: unknown[] };
  return {
    dir: dirname(path),
    path: relative(contentRoot, path),
    pages: (raw.pages ?? []).filter((entry): entry is string => typeof entry === "string"),
  };
}

function listMetaFiles(): MetaFile[] {
  return checkedRoots.flatMap(root => listFiles(root, "meta.json").map(readMeta));
}

function isSeparator(entry: string): boolean {
  return entry.startsWith("---");
}

function targetExists(dir: string, entry: string): boolean {
  return (
    existsSync(resolve(dir, `${entry}.mdx`)) ||
    existsSync(resolve(dir, entry, "meta.json")) ||
    existsSync(resolve(dir, entry, "index.mdx"))
  );
}

function childContentEntries(dir: string): string[] {
  return readdirSync(dir)
    .flatMap(entry => {
      const fullPath = resolve(dir, entry);
      const stat = statSync(fullPath);
      if (stat.isFile() && entry.endsWith(".mdx") && entry !== "index.mdx") {
        return [entry.replace(/\.mdx$/, "")];
      }
      if (stat.isDirectory() && hasContent(fullPath)) {
        return [entry];
      }
      return [];
    })
    .sort();
}

function hasContent(dir: string): boolean {
  return readdirSync(dir).some(entry => {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isFile()) {
      return entry.endsWith(".mdx") || entry === "meta.json";
    }
    return stat.isDirectory() && hasContent(fullPath);
  });
}

describe("content meta navigation", () => {
  it("points every runtime and protocol meta entry at an existing page or section", () => {
    const missingTargets = listMetaFiles().flatMap(meta =>
      meta.pages
        .filter(entry => !isSeparator(entry))
        .filter(entry => !targetExists(meta.dir, entry))
        .map(entry => `${meta.path} -> ${entry}`)
    );

    expect(missingTargets).toEqual([]);
  });

  it("keeps non-index runtime and protocol pages discoverable from their local meta", () => {
    const missingEntries = listMetaFiles().flatMap(meta => {
      const discoverable = new Set(meta.pages.filter(entry => !isSeparator(entry)));
      return childContentEntries(meta.dir)
        .filter(entry => !discoverable.has(entry))
        .map(entry => `${meta.path} missing ${entry}`);
    });

    expect(missingEntries).toEqual([]);
  });

  it("does not duplicate entries inside runtime and protocol meta pages arrays", () => {
    const duplicates = listMetaFiles().flatMap(meta => {
      const seen = new Set<string>();
      return meta.pages
        .filter(entry => !isSeparator(entry))
        .filter(entry => {
          if (seen.has(entry)) {
            return true;
          }
          seen.add(entry);
          return false;
        })
        .map(entry => `${meta.path} duplicates ${entry}`);
    });

    expect(duplicates).toEqual([]);
  });
});
