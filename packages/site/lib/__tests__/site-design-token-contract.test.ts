import { readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { publicRoot, siteRoot } from "../content-test-utils";

const checkedRoots = ["app", "components", "content", "lib"].map(root => resolve(siteRoot, root));
const checkedPublicFiles = ["favicon.svg", "site.webmanifest"].map(file =>
  resolve(publicRoot, file)
);
const generatedPathSegments = [`content${"/"}runtime${"/"}cli-reference${"/"}`];
const hexColorPattern = /#[0-9a-fA-F]{6}(?:[0-9a-fA-F]{2})?\b/g;

/*
 * Site-local palette allowlist. The runtime kit moved to a new warm-dark
 * contract under `.compozy/tasks/redesign` (P1 token foundation); the site
 * keeps the legacy palette below until its own redesign TechSpec runs. The
 * shared accent (`#E8572A`) and tints stay valid because they were not
 * deleted by the runtime cut.
 */
const SITE_PALETTE: ReadonlyArray<string> = [
  "#0e0e0f",
  "#141312",
  "#181716",
  "#1e1c1b",
  "#2e2c2b",
  "#353332",
  "#3c3a39",
  "#4a4847",
  "#e5e5e7",
  "#8e8e93",
  "#636366",
  "#98989d",
  "#e8572a",
  "#d14e25",
  "#f6874f",
  "#17110f",
  "#e8572a59",
  "#e8572a3d",
  "#e8572a26",
  "#5ba6ff",
  "#b892ff",
  "#4fd1c5",
  "#30d158",
  "#30d15826",
  "#ff453a",
  "#ff453a26",
  "#ffd60a",
  "#ffd60a26",
  "#bf5af2",
  "#bf5af226",
  "#63636626",
  "#000000",
  "#ffffff",
];

const allowedColors = new Set(SITE_PALETTE.map(color => color.toLowerCase()));

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
    if (
      stat.isFile() &&
      /\.(?:css|mdx?|tsx?|json|svg)$/.test(fullPath) &&
      !/\.test\.[cm]?[tj]sx?$/.test(fullPath)
    ) {
      files.push(fullPath);
    }
  }
  return files;
}

function sourceFiles(): string[] {
  return [...checkedRoots.flatMap(root => listFiles(root)), ...checkedPublicFiles]
    .filter(file => !generatedPathSegments.some(segment => file.includes(segment)))
    .sort((left, right) => left.localeCompare(right));
}

describe("site design token contract", () => {
  it("Should keep hardcoded hex colors inside the site palette allowlist", () => {
    const violations = sourceFiles().flatMap(file => {
      const content = readFileSync(file, "utf8");
      const colors = [...new Set([...content.matchAll(hexColorPattern)].map(match => match[0]))];

      return colors
        .filter(color => !allowedColors.has(color.toLowerCase()))
        .map(color => `${relative(siteRoot, file)} -> ${color}`);
    });

    expect(violations).toEqual([]);
  });
});
