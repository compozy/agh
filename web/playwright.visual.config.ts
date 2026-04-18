import { fileURLToPath } from "node:url";
import path from "node:path";

import { defineConfig, devices } from "@playwright/test";

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.join(rootDir, ".tmp", "playwright-visual");
const snapshotDir = path.join(rootDir, "tests", "visual", "__snapshots__");

const STORYBOOK_PORT = Number(process.env.AGH_WEB_STORYBOOK_PORT ?? 6008);
export const STORYBOOK_URL = `http://127.0.0.1:${STORYBOOK_PORT}`;

export default defineConfig({
  testDir: "./tests/visual",
  testMatch: ["**/*.spec.ts"],
  snapshotDir,
  snapshotPathTemplate: "{snapshotDir}/{arg}-{projectName}-{platform}{ext}",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  workers: process.env.CI ? 2 : undefined,
  timeout: 60_000,
  outputDir: path.join(outDir, "test-results"),
  reporter: [["list"], ["html", { open: "never", outputFolder: path.join(outDir, "report") }]],
  expect: {
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.001,
      animations: "disabled",
      caret: "hide",
    },
  },
  use: {
    ...devices["Desktop Chrome"],
    baseURL: STORYBOOK_URL,
    headless: true,
    colorScheme: "dark",
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 1,
    ignoreHTTPSErrors: true,
    trace: "off",
    screenshot: "off",
    video: "off",
    contextOptions: {
      // `reducedMotion` is not declared on top-level `use` in
      // @playwright/test@1.59 types — set it via `contextOptions` instead.
      reducedMotion: "reduce",
    },
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: `bun run scripts/serve-storybook.ts .tmp/storybook-static ${STORYBOOK_PORT} 127.0.0.1`,
    url: `${STORYBOOK_URL}/index.json`,
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    stdout: "pipe",
    stderr: "pipe",
    env: {
      AGH_WEB_STORYBOOK_PORT: String(STORYBOOK_PORT),
    },
  },
});
