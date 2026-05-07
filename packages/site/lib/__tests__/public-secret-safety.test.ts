import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { siteRoot } from "../content-test-utils";

const checkedRoots = ["app", "components", "content", "lib"].map(root => resolve(siteRoot, root));
const skippedDirectories = new Set([".next", ".source", ".velite", "dist", "node_modules"]);
const checkedFilePattern = /\.(?:mdx?|tsx?|json|ya?ml)$/;
const skippedFilePattern = /\.test\.(?:ts|tsx)$/;
const secretPatterns: Array<[label: string, pattern: RegExp]> = [
  ["private key", /-----BEGIN [A-Z ]*PRIVATE KEY-----/],
  ["OpenAI-style API key", /sk-[A-Za-z0-9]{20,}/],
  ["GitHub token", /gh[pousr]_[A-Za-z0-9_]{20,}/],
  ["Slack token", /xox[baprs]-[A-Za-z0-9-]{20,}/],
  ["AWS access key", /AKIA[0-9A-Z]{16}/],
  ["JWT", /eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}/],
];

function listCheckedFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = resolve(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      if (skippedDirectories.has(entry)) {
        continue;
      }
      files.push(...listCheckedFiles(fullPath));
      continue;
    }
    if (stat.isFile() && checkedFilePattern.test(fullPath) && !skippedFilePattern.test(fullPath)) {
      files.push(fullPath);
    }
  }
  return files.sort();
}

describe("public site secret safety", () => {
  it("does not publish real-looking secret material in site source or manual content", () => {
    const violations = checkedRoots.flatMap(root =>
      listCheckedFiles(root).flatMap(file => {
        const content = readFileSync(file, "utf8");
        return secretPatterns.flatMap(([label, pattern]) =>
          pattern.test(content) ? [`${relative(siteRoot, file)}: ${label}`] : []
        );
      })
    );

    expect(violations).toEqual([]);
  });
});
