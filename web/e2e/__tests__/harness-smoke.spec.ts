import { readdir, readFile, stat } from "node:fs/promises";

import type { BrowserConsoleEntry, BrowserNetworkEntry } from "../fixtures/artifacts";
import { expect, test } from "../fixtures/test";

test("boots against the daemon-served onboarding shell and captures a trace plus screenshot bundle", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  await expect(appPage.getByTestId("workspace-onboarding")).toBeVisible();

  const manifest = await browserArtifacts.persist(appPage);
  expect(manifest.artifacts).toEqual(
    expect.arrayContaining([
      expect.objectContaining({ kind: "browser_trace", path: "browser_trace.zip" }),
      expect.objectContaining({ kind: "browser_screenshots", path: "browser_screenshots" }),
    ])
  );

  const traceStat = await stat(runtime.artifactCollector.artifactPath("browser_trace"));
  expect(traceStat.isFile()).toBe(true);

  const screenshots = await readdir(runtime.artifactCollector.artifactPath("browser_screenshots"));
  expect(screenshots.length).toBeGreaterThan(0);
});

test("captures console and network diagnostics after a forced failure path", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  await expect(appPage.getByTestId("workspace-onboarding")).toBeVisible();

  const failure = await appPage.evaluate(async () => {
    console.error("agh-playwright-forced-console-error");
    const response = await fetch("/api/not-found");
    return { ok: response.ok, status: response.status };
  });

  expect(failure).toEqual({ ok: false, status: 404 });

  await browserArtifacts.persist(appPage);

  const consoleEntries = JSON.parse(
    await readFile(runtime.artifactCollector.artifactPath("browser_console"), "utf8")
  ) as BrowserConsoleEntry[];
  expect(
    consoleEntries.some(entry => entry.text.includes("agh-playwright-forced-console-error"))
  ).toBe(true);

  const networkEntries = JSON.parse(
    await readFile(runtime.artifactCollector.artifactPath("browser_network"), "utf8")
  ) as BrowserNetworkEntry[];
  expect(
    networkEntries.some(
      entry =>
        entry.event === "response" && entry.status === 404 && entry.url.endsWith("/api/not-found")
    )
  ).toBe(true);
});
