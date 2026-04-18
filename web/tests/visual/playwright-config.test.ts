import { describe, expect, it } from "vitest";

import playwrightConfig, { STORYBOOK_URL } from "../../playwright.visual.config";

describe("web/ Playwright visual config", () => {
  it("Should force prefers-reduced-motion: reduce at the browser context level", () => {
    expect(playwrightConfig.use?.contextOptions?.reducedMotion).toBe("reduce");
  });

  it("Should tighten the visual diff threshold to 0.1%", () => {
    expect(playwrightConfig.expect?.toHaveScreenshot?.maxDiffPixelRatio).toBe(0.001);
  });

  it("Should store snapshots under web/tests/visual/__snapshots__/", () => {
    expect(playwrightConfig.snapshotDir ?? "").toMatch(/tests\/visual\/__snapshots__$/);
  });

  it("Should emit per-platform snapshot paths so CI (linux) and local (darwin) do not clash", () => {
    expect(playwrightConfig.snapshotPathTemplate).toContain("{platform}");
    expect(playwrightConfig.snapshotPathTemplate).toContain("{projectName}");
  });

  it("Should pin the web Storybook dev server to port 6008 by default (distinct from @agh/ui 6007)", () => {
    expect(STORYBOOK_URL).toBe("http://127.0.0.1:6008");
  });

  it("Should force the dark color scheme to match the production theme", () => {
    expect(playwrightConfig.use?.colorScheme).toBe("dark");
  });

  it("Should pin the viewport so Linux and macOS baselines agree on content layout", () => {
    expect(playwrightConfig.use?.viewport).toEqual({ width: 1280, height: 800 });
  });

  it("Should boot the Storybook static server before running Playwright", () => {
    const webServer = Array.isArray(playwrightConfig.webServer)
      ? playwrightConfig.webServer[0]
      : playwrightConfig.webServer;
    expect(webServer?.command).toContain("scripts/serve-storybook.ts");
    expect(webServer?.url).toBe(`${STORYBOOK_URL}/index.json`);
    expect(webServer?.env?.AGH_WEB_STORYBOOK_PORT).toBe("6008");
  });

  it("Should run from the tests/visual directory with *.spec.ts matcher", () => {
    expect(playwrightConfig.testDir).toBe("./tests/visual");
    expect(playwrightConfig.testMatch).toEqual(["**/*.spec.ts"]);
  });
});
