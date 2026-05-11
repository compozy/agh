import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const REPO_ROOT = join(__dirname, "../../..");
const TOKENS_PATH = join(REPO_ROOT, "packages/ui/src/tokens.css");
const OXLINTRC_PATH = join(REPO_ROOT, ".oxlintrc.json");
const COMPOZY_PLUGIN_PATH = join(REPO_ROOT, "lint-plugins/compozy-design-system.mjs");
const OBSOLETE_PLUGIN_PATH = join(REPO_ROOT, "lint-plugins/no-inline-eyebrow.mjs");
const WEB_SRC_ROOT = join(REPO_ROOT, "web/src");
const KIT_SRC_ROOT = join(REPO_ROOT, "packages/ui/src");
const KIT_SPINNER_PATH = join(KIT_SRC_ROOT, "components/spinner.tsx");
const KIT_SONNER_PATH = join(KIT_SRC_ROOT, "components/sonner.tsx");
const SHOWCASE_PATH = join(WEB_SRC_ROOT, "components/design-system-showcase.tsx");

interface CollectOptions {
  extensions: ReadonlyArray<string>;
  excludePaths: ReadonlyArray<string>;
}

function collectSourceFiles(dir: string, options: CollectOptions, into: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const stat = statSync(full);
    if (stat.isDirectory()) {
      if (entry === "node_modules" || entry === "dist" || entry === "generated") continue;
      if (entry === "__tests__") continue;
      collectSourceFiles(full, options, into);
      continue;
    }
    if (entry.endsWith(".stories.tsx") || entry.includes(".test.")) continue;
    if (!options.extensions.some(ext => entry.endsWith(ext))) continue;
    if (options.excludePaths.some(path => full.endsWith(path))) continue;
    into.push(full);
  }
  return into;
}

const RUNTIME_FILES = [
  ...collectSourceFiles(WEB_SRC_ROOT, {
    extensions: [".tsx", ".ts"],
    excludePaths: [],
  }),
  ...collectSourceFiles(KIT_SRC_ROOT, {
    extensions: [".tsx", ".ts"],
    excludePaths: [KIT_SPINNER_PATH, KIT_SONNER_PATH],
  }),
];

function findOffenders(pattern: RegExp, allowList: ReadonlyArray<string> = []): string[] {
  const offenders: string[] = [];
  for (const filePath of RUNTIME_FILES) {
    if (allowList.includes(filePath)) continue;
    const content = readFileSync(filePath, "utf-8");
    if (pattern.test(content)) offenders.push(filePath);
  }
  return offenders;
}

describe("redesign-v2 PR-4 closeout — token contract", () => {
  const tokens = readFileSync(TOKENS_PATH, "utf-8");

  it("Should not declare --tracking-badge (deleted in task_29 / PR-4)", () => {
    expect(tokens).not.toMatch(/--tracking-badge\b/);
  });
});

describe("redesign-v2 PR-4 closeout — oxlint config", () => {
  const config = JSON.parse(readFileSync(OXLINTRC_PATH, "utf-8")) as {
    jsPlugins: string[];
    rules: Record<string, unknown>;
  };

  it("Should load the consolidated compozy-design-system plugin", () => {
    expect(config.jsPlugins).toContain("./lint-plugins/compozy-design-system.mjs");
  });

  it("Should no longer reference the obsolete no-inline-eyebrow plugin file", () => {
    expect(config.jsPlugins).not.toContain("./lint-plugins/no-inline-eyebrow.mjs");
    expect(existsSync(OBSOLETE_PLUGIN_PATH)).toBe(false);
  });

  it("Should ship every redesign-v2 design-system rule at error severity", () => {
    expect(config.rules["compozy-design-system/no-inline-eyebrow"]).toBe("error");
    expect(config.rules["compozy-design-system/no-design-glaze-rgba"]).toBe("error");
    expect(config.rules["compozy-design-system/no-banned-imports"]).toBe("error");
    expect(config.rules["compozy-design-system/no-inline-design-tuples"]).toBe("error");
  });

  it("Should expose all four rules from the loaded plugin", () => {
    const plugin = readFileSync(COMPOZY_PLUGIN_PATH, "utf-8");
    expect(plugin).toMatch(/"no-inline-eyebrow":\s*noInlineEyebrow/);
    expect(plugin).toMatch(/"no-design-glaze-rgba":\s*noDesignGlazeRgba/);
    expect(plugin).toMatch(/"no-banned-imports":\s*noBannedImports/);
    expect(plugin).toMatch(/"no-inline-design-tuples":\s*noInlineDesignTuples/);
  });
});

describe("redesign-v2 PR-4 closeout — runtime sweep evidence", () => {
  it("Should find runtime source files to scan", () => {
    expect(RUNTIME_FILES.length).toBeGreaterThan(0);
  });

  it("Should remove every Loader2 / Loader2Icon import outside canonical owners", () => {
    const offenders = findOffenders(/\bLoader2(?:Icon)?\b/);
    expect(offenders).toEqual([]);
  });

  it("Should remove every inline glaze rgba literal outside the showcase allowlist", () => {
    const offenders = findOffenders(/bg-\[rgba\(255,\s*255,\s*255,\s*0\.\d+\)\]/, [SHOWCASE_PATH]);
    expect(offenders).toEqual([]);
  });

  it("Should remove the inline 22px page-h1 tuple", () => {
    const offenders = findOffenders(/text-\[22px\][\s\S]*?tracking-\[-0\.026em\]/);
    expect(offenders).toEqual([]);
  });
});
