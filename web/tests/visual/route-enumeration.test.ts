import path from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { enumerateRoutes, enumerateStoryFiles } from "./route-enumeration";

const thisDir = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(thisDir, "..", "..");
const routesRoot = path.join(webRoot, "src", "routes");
const srcRoot = path.join(webRoot, "src");

describe("enumerateRoutes", () => {
  const routes = enumerateRoutes(routesRoot);
  const urlPaths = routes.map(r => r.urlPath);

  it("Should expose every top-level public route under web/src/routes", () => {
    const expected = [
      "/",
      "/automation",
      "/bridges",
      "/design-system",
      "/knowledge",
      "/network",
      "/session/$id",
      "/settings",
      "/settings/automation",
      "/settings/environments",
      "/settings/general",
      "/settings/hooks-extensions",
      "/settings/mcp-servers",
      "/settings/memory",
      "/settings/network",
      "/settings/observability",
      "/settings/providers",
      "/settings/skills",
      "/skills",
      "/tasks",
      "/tasks/$id",
      "/tasks/$id/edit",
      "/tasks/$id/runs/$runId",
      "/tasks/new",
    ];
    for (const route of expected) {
      expect(urlPaths).toContain(route);
    }
  });

  it("Should skip TanStack-excluded files and folders (prefixed with '-')", () => {
    for (const entry of routes) {
      expect(entry.filePath).not.toMatch(new RegExp(`${path.sep}-[^${path.sep}]+\\.tsx$`));
    }
  });

  it("Should skip colocated .test.tsx files, .stories.tsx files, and __root.tsx", () => {
    for (const entry of routes) {
      expect(entry.filePath.endsWith(".test.tsx")).toBe(false);
      expect(entry.filePath.endsWith(".stories.tsx")).toBe(false);
      expect(path.basename(entry.filePath)).not.toBe("__root.tsx");
    }
  });

  it("Should collapse /index routes onto their parent URL", () => {
    expect(urlPaths).toContain("/");
    expect(urlPaths).toContain("/settings");
    expect(urlPaths).not.toContain("/index");
    expect(urlPaths).not.toContain("/settings/index");
  });

  it("Should treat design-system as a public route (outside the _app layout)", () => {
    const designSystem = routes.find(r => r.urlPath === "/design-system");
    expect(designSystem).toBeDefined();
    expect(designSystem?.isPublic).toBe(true);
  });
});

describe("enumerateStoryFiles", () => {
  const files = enumerateStoryFiles(srcRoot);

  it("Should discover every *.stories.tsx under web/src/**", () => {
    expect(files.length).toBeGreaterThan(0);
    for (const file of files) {
      expect(file.endsWith(".stories.tsx")).toBe(true);
    }
  });

  it("Should include primitive stories under web/src/components/stories/", () => {
    const componentStories = files.filter(f =>
      f.includes(path.join("src", "components", "stories"))
    );
    expect(componentStories.length).toBeGreaterThan(0);
  });

  it("Should include domain stories under web/src/systems/**/components/stories/", () => {
    const systemStories = files.filter(f => f.includes(path.join("src", "systems")));
    expect(systemStories.length).toBeGreaterThan(0);
  });

  it("Should include at least one route story under web/src/routes/_app/stories/", () => {
    const routeStories = files.filter(f =>
      f.includes(path.join("src", "routes", "_app", "stories"))
    );
    expect(routeStories.length).toBeGreaterThan(0);
  });

  it("Should not include snapshot directories or node_modules", () => {
    for (const file of files) {
      expect(file).not.toContain(`${path.sep}__snapshots__${path.sep}`);
      expect(file).not.toContain(`${path.sep}node_modules${path.sep}`);
    }
  });
});

describe("story ↔ route coverage", () => {
  const routes = enumerateRoutes(routesRoot);
  const stories = enumerateStoryFiles(srcRoot);
  const routeStories = stories.filter(f => f.includes(path.join("src", "routes", "_app")));

  it("Should have a route story for every _app route that currently ships with one", () => {
    // This is a "coverage inventory" — lists top-level _app routes with matching stories.
    // Future redesign tasks (Phase 3–6) cover any gaps; Phase 2 only requires
    // the existing route stories to baseline against.
    const storyFiles = routeStories.map(f => path.basename(f));
    const expectations: Array<[string, string]> = [
      ["/", "-index.stories.tsx"],
      ["/automation", "-automation.stories.tsx"],
      ["/bridges", "-bridges.stories.tsx"],
      ["/knowledge", "-knowledge.stories.tsx"],
      ["/network", "-network.stories.tsx"],
      ["/skills", "-skills.stories.tsx"],
      ["/session/$id", "-session.stories.tsx"],
    ];
    for (const [url, storyBasename] of expectations) {
      const route = routes.find(r => r.urlPath === url);
      expect(route, `missing route ${url}`).toBeDefined();
      expect(storyFiles, `missing story ${storyBasename} for ${url}`).toContain(storyBasename);
    }
  });
});
