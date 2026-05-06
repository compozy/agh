import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const checkedRoots = [
  resolve(siteRoot, "app/changelog"),
  resolve(siteRoot, "components/landing"),
  resolve(siteRoot, "content/blog/posts"),
];

function listFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      if (entry === "__tests__") {
        continue;
      }
      files.push(...listFiles(fullPath));
      continue;
    }
    if (stat.isFile() && /\.(?:mdx?|tsx?)$/.test(fullPath)) {
      files.push(fullPath);
    }
  }
  return files.sort((left, right) => left.localeCompare(right));
}

describe("site copy contract", () => {
  it("keeps public marketing and release body copy out of first-person plural voice", () => {
    const violations = checkedRoots.flatMap(root =>
      listFiles(root).flatMap(file => {
        const content = readFileSync(file, "utf8");
        return [...content.matchAll(/\b(?:we|our)\b/gi)].map(
          match => `${relative(siteRoot, file)}: ${match[0]}`
        );
      })
    );

    expect(violations).toEqual([]);
  });
});
