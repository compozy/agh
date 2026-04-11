import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";
import {
  disableSkill,
  enableSkill,
  getSkill,
  getSkillContent,
  listSkills,
  SkillApiError,
} from "@/systems/skill/adapters/skill-api";

const validSkill = {
  name: "test-skill",
  description: "A test skill",
  source: "bundled",
  enabled: true,
  dir: "/path/to/skill",
};

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("listSkills", () => {
  const validResponse = { skills: [validSkill] };

  it("calls GET /api/skills?workspace=:id and returns typed array", async () => {
    mockJsonResponse(validResponse);

    const result = await listSkills("ws_123");

    expect(result).toEqual([validSkill]);
    await expectFetchRequest({ path: "/api/skills?workspace=ws_123" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse(validResponse);

    const controller = new AbortController();
    await listSkills("ws_123", controller.signal);

    await expectFetchRequest({
      path: "/api/skills?workspace=ws_123",
      signal: controller.signal,
    });
  });

  it("returns empty array when server returns empty list", async () => {
    mockJsonResponse({ skills: [] });

    const result = await listSkills("ws_123");

    expect(result).toEqual([]);
  });

  it("throws SkillApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(listSkills("ws_123")).rejects.toThrow(SkillApiError);
    await expect(listSkills("ws_123")).rejects.toThrow("Failed to fetch skills: 500");
  });

  it("encodes workspace in URL", async () => {
    mockJsonResponse(validResponse);

    await listSkills("/home/user/project");

    await expectFetchRequest({ path: "/api/skills?workspace=%2Fhome%2Fuser%2Fproject" });
  });
});

describe("getSkill", () => {
  const validResponse = { skill: validSkill };

  it("calls GET /api/skills/:name?workspace=:id and returns typed object", async () => {
    mockJsonResponse(validResponse);

    const result = await getSkill("test-skill", "ws_123");

    expect(result).toEqual(validSkill);
    await expectFetchRequest({ path: "/api/skills/test-skill?workspace=ws_123" });
  });

  it("throws SkillApiError with 404 for unknown skill", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(getSkill("unknown", "ws_123")).rejects.toThrow("Skill not found: unknown");

    try {
      await getSkill("unknown", "ws_123");
    } catch (error) {
      expect(error).toBeInstanceOf(SkillApiError);
      expect((error as SkillApiError).status).toBe(404);
    }
  });

  it("throws SkillApiError for other failures", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 503 }));

    await expect(getSkill("test-skill", "ws_123")).rejects.toThrow(
      'Failed to fetch skill "test-skill": 503'
    );
  });

  it("encodes skill name in URL", async () => {
    mockJsonResponse(validResponse);

    await getSkill("my skill", "ws_123");

    await expectFetchRequest({ path: "/api/skills/my%20skill?workspace=ws_123" });
  });
});

describe("getSkillContent", () => {
  it("calls GET /api/skills/:name/content?workspace=:id and returns content string", async () => {
    mockJsonResponse({ content: "full skill content" });

    const result = await getSkillContent("test-skill", "ws_123");

    expect(result).toBe("full skill content");
    await expectFetchRequest({ path: "/api/skills/test-skill/content?workspace=ws_123" });
  });

  it("encodes skill name in content URL", async () => {
    mockJsonResponse({ content: "full skill content" });

    await getSkillContent("my skill", "ws_123");

    await expectFetchRequest({ path: "/api/skills/my%20skill/content?workspace=ws_123" });
  });
});

describe("enableSkill", () => {
  it("calls POST /api/skills/:name/enable and returns {ok: true}", async () => {
    mockJsonResponse({ ok: true });

    const result = await enableSkill("test-skill", "ws_123");

    expect(result).toEqual({ ok: true });
    await expectFetchRequest({
      method: "POST",
      path: "/api/skills/test-skill/enable?workspace=ws_123",
    });
  });

  it("throws SkillApiError on 404", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(enableSkill("unknown", "ws_123")).rejects.toThrow("Skill not found: unknown");
  });

  it("throws SkillApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(enableSkill("test-skill", "ws_123")).rejects.toThrow(SkillApiError);
  });
});

describe("disableSkill", () => {
  it("calls POST /api/skills/:name/disable and returns {ok: true}", async () => {
    mockJsonResponse({ ok: true });

    const result = await disableSkill("test-skill", "ws_123");

    expect(result).toEqual({ ok: true });
    await expectFetchRequest({
      method: "POST",
      path: "/api/skills/test-skill/disable?workspace=ws_123",
    });
  });

  it("throws SkillApiError on 404", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 404 }));

    await expect(disableSkill("unknown", "ws_123")).rejects.toThrow("Skill not found: unknown");
  });

  it("throws SkillApiError on non-2xx response", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(disableSkill("test-skill", "ws_123")).rejects.toThrow(SkillApiError);
  });
});

describe("SkillApiError", () => {
  it("has correct name and status properties", () => {
    const error = new SkillApiError("test error", 404);

    expect(error.name).toBe("SkillApiError");
    expect(error.status).toBe(404);
    expect(error.message).toBe("test error");
    expect(error).toBeInstanceOf(Error);
  });
});
