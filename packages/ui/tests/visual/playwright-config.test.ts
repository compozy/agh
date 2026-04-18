import { describe, expect, it } from "vitest";

import playwrightConfig, { STORYBOOK_URL } from "../../playwright.config";

describe("packages/ui Playwright config", () => {
  it("Should force prefers-reduced-motion: reduce at the browser context level", () => {
    expect(playwrightConfig.use?.reducedMotion).toBe("reduce");
  });

  it("Should tighten the visual diff threshold to 0.1%", () => {
    expect(playwrightConfig.expect?.toHaveScreenshot?.maxDiffPixelRatio).toBe(0.001);
  });

  it("Should store snapshots under packages/ui/src/components/stories/__snapshots__/", () => {
    expect(playwrightConfig.snapshotDir ?? "").toMatch(/components\/stories\/__snapshots__$/);
  });

  it("Should emit per-platform snapshot paths so CI (linux) and local (darwin) do not clash", () => {
    expect(playwrightConfig.snapshotPathTemplate).toContain("{platform}");
    expect(playwrightConfig.snapshotPathTemplate).toContain("{projectName}");
  });

  it("Should pin the Storybook dev server to port 6007 by default", () => {
    expect(STORYBOOK_URL).toBe("http://127.0.0.1:6007");
  });

  it("Should boot the Storybook static server before running Playwright", () => {
    const webServer = Array.isArray(playwrightConfig.webServer)
      ? playwrightConfig.webServer[0]
      : playwrightConfig.webServer;
    expect(webServer?.command).toContain("scripts/serve-storybook.ts");
    expect(webServer?.url).toBe(`${STORYBOOK_URL}/index.json`);
  });
});
