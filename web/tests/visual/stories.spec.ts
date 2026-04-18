import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

import { expect, test } from "@playwright/test";

import {
  assertStorybookIndex,
  collectVisualTargets,
  type StorybookIndex,
} from "@agh/ui/testing/visual";

import { STORYBOOK_URL } from "../../playwright.visual.config";

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");
const indexPath = path.join(rootDir, ".tmp", "storybook-static", "index.json");

function loadIndex(): StorybookIndex {
  let payload: unknown;
  try {
    payload = JSON.parse(readFileSync(indexPath, "utf8"));
  } catch (err) {
    throw new Error(
      `Unable to read Storybook index at ${indexPath}. Run 'bun run build:visual' before 'bun run test:visual'. Underlying: ${(err as Error).message}`
    );
  }
  assertStorybookIndex(payload);
  return payload;
}

const index = loadIndex();
const targets = collectVisualTargets(index, STORYBOOK_URL, { excludeTags: ["play-fn"] });

if (targets.length === 0) {
  throw new Error(
    `Storybook index at ${indexPath} returned zero snapshot targets — rebuild storybook bundle.`
  );
}

test.describe("web story snapshots", () => {
  for (const target of targets) {
    test(target.id, async ({ page }) => {
      await page.emulateMedia({ reducedMotion: "reduce", colorScheme: "dark" });
      await page.goto(target.storyUrl, { waitUntil: "load" });
      // Wait for the Storybook root to attach. Stories that open a Base UI
      // Dialog mark `#storybook-root` as `aria-hidden` + `data-base-ui-inert`,
      // which Playwright treats as "hidden", so we can't wait for visibility.
      await page.locator("#storybook-root").waitFor({ state: "attached" });
      await waitForFonts(page);
      // Snapshot the entire viewport so portaled content (Dialog, Sheet,
      // Popover, Tooltip) is captured alongside `#storybook-root`.
      await expect(page).toHaveScreenshot(target.snapshotName, {
        maxDiffPixelRatio: 0.001,
        animations: "disabled",
        caret: "hide",
        fullPage: false,
      });
    });
  }
});

const fontProbeFamilies = [
  "400 14px 'Inter Variable'",
  "500 14px 'Inter Variable'",
  "600 14px 'Inter Variable'",
  "700 14px 'Inter Variable'",
  "500 14px 'JetBrains Mono'",
  "600 14px 'JetBrains Mono'",
];

async function waitForFonts(page: import("@playwright/test").Page): Promise<void> {
  await page.evaluate(async (families: string[]) => {
    if (!document.fonts) return;
    await Promise.all(families.map(fam => document.fonts.load(fam)));
    await document.fonts.ready;
  }, fontProbeFamilies);
}
