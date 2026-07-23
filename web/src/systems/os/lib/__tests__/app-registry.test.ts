import { describe, expect, it } from "vitest";

import { OS_APPS, matchSessionInstance, resolveAppForPath } from "../app-registry";

describe("app registry", () => {
  it("Should extract the session instance key and reject non-session agent paths (UT-050)", () => {
    expect(matchSessionInstance("/agents/webgen/sessions/s1")).toBe("s1");
    expect(matchSessionInstance("/agents/webgen/settings")).toBeNull();
    expect(OS_APPS.session.matchInstance?.("/agents/webgen/sessions/s1")).toBe("s1");
  });

  it("Should resolve pathname ownership per app prefix (UT-051)", () => {
    expect(resolveAppForPath("/loop-runs/r1")?.app.id).toBe("loops");
    expect(resolveAppForPath("/marketplace/skills")?.app.id).toBe("marketplace");
    expect(resolveAppForPath("/does-not-exist")).toBeNull();
  });

  it("Should route session paths to the session app ahead of the agents prefix", () => {
    const resolved = resolveAppForPath("/agents/webgen/sessions/s1");
    expect(resolved?.app.id).toBe("session");
    expect(resolved?.instanceKey).toBe("s1");
    expect(resolveAppForPath("/agents/webgen")?.app.id).toBe("agents");
  });

  it("Should own the desktop root exactly (no prefix bleed)", () => {
    expect(resolveAppForPath("/")?.app.id).toBe("dashboard");
    expect(resolveAppForPath("/tasks")?.app.id).toBe("tasks");
  });

  it("Should open cramped dock work surfaces at enlarged default geometry", () => {
    expect(OS_APPS.agents.defaultRect).toMatchObject({ w: 920, h: 640 });
    expect(OS_APPS.loops.defaultRect).toMatchObject({ w: 920, h: 640 });
    expect(OS_APPS.jobs.defaultRect).toMatchObject({ w: 920, h: 640 });
    expect(OS_APPS.dashboard.defaultRect).toMatchObject({ w: 960, h: 680 });
    expect(OS_APPS.session.defaultRect).toMatchObject({ w: 860, h: 680 });
    expect(OS_APPS.network.defaultRect).toMatchObject({ w: 1200, h: 720 });
    expect(OS_APPS.tasks.defaultRect).toMatchObject({ w: 1160, h: 720 });
    expect(OS_APPS.settings.defaultRect).toMatchObject({ w: 1080, h: 680 });
  });
});
