import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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

describe("listSkills", () => {
  const validResponse = { skills: [validSkill] };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("calls GET /api/skills?workspace=:id and returns typed array", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const result = await listSkills("ws_123");
    expect(result).toEqual([validSkill]);
    expect(fetch).toHaveBeenCalledWith("/api/skills?workspace=ws_123", { signal: undefined });
  });

  it("passes abort signal to fetch", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const controller = new AbortController();
    await listSkills("ws_123", controller.signal);
    expect(fetch).toHaveBeenCalledWith("/api/skills?workspace=ws_123", {
      signal: controller.signal,
    });
  });

  it("returns empty array when server returns empty list", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ skills: [] }),
    } as Response);

    const result = await listSkills("ws_123");
    expect(result).toEqual([]);
  });

  it("throws SkillApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(listSkills("ws_123")).rejects.toThrow(SkillApiError);
    await expect(listSkills("ws_123")).rejects.toThrow("Failed to fetch skills: 500");
  });

  it("encodes workspace in URL", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    await listSkills("/home/user/project");
    expect(fetch).toHaveBeenCalledWith(
      "/api/skills?workspace=%2Fhome%2Fuser%2Fproject",
      expect.any(Object)
    );
  });
});

describe("getSkill", () => {
  const validResponse = { skill: validSkill };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("calls GET /api/skills/:name?workspace=:id and returns typed object", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    const result = await getSkill("test-skill", "ws_123");
    expect(result).toEqual(validSkill);
    expect(fetch).toHaveBeenCalledWith("/api/skills/test-skill?workspace=ws_123", {
      signal: undefined,
    });
  });

  it("throws SkillApiError with 404 for unknown skill", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 404,
    } as Response);

    await expect(getSkill("unknown", "ws_123")).rejects.toThrow("Skill not found: unknown");
    try {
      await getSkill("unknown", "ws_123");
    } catch (err) {
      expect(err).toBeInstanceOf(SkillApiError);
      expect((err as SkillApiError).status).toBe(404);
    }
  });

  it("throws SkillApiError for other failures", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 503,
    } as Response);

    await expect(getSkill("test-skill", "ws_123")).rejects.toThrow(
      'Failed to fetch skill "test-skill": 503'
    );
  });

  it("encodes skill name in URL", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(validResponse),
    } as Response);

    await getSkill("my skill", "ws_123");
    expect(fetch).toHaveBeenCalledWith("/api/skills/my%20skill?workspace=ws_123", {
      signal: undefined,
    });
  });
});

describe("getSkillContent", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("calls GET /api/skills/:name/content?workspace=:id and returns content string", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ content: "full skill content" }),
    } as Response);

    const result = await getSkillContent("test-skill", "ws_123");
    expect(result).toBe("full skill content");
    expect(fetch).toHaveBeenCalledWith("/api/skills/test-skill/content?workspace=ws_123", {
      signal: undefined,
    });
  });

  it("encodes skill name in content URL", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ content: "full skill content" }),
    } as Response);

    await getSkillContent("my skill", "ws_123");
    expect(fetch).toHaveBeenCalledWith("/api/skills/my%20skill/content?workspace=ws_123", {
      signal: undefined,
    });
  });
});

describe("enableSkill", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("calls POST /api/skills/:name/enable and returns {ok: true}", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    } as Response);

    const result = await enableSkill("test-skill", "ws_123");
    expect(result).toEqual({ ok: true });
    expect(fetch).toHaveBeenCalledWith("/api/skills/test-skill/enable?workspace=ws_123", {
      method: "POST",
      signal: undefined,
    });
  });

  it("throws SkillApiError on 404", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 404,
    } as Response);

    await expect(enableSkill("unknown", "ws_123")).rejects.toThrow("Skill not found: unknown");
  });

  it("throws SkillApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(enableSkill("test-skill", "ws_123")).rejects.toThrow(SkillApiError);
  });
});

describe("disableSkill", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("calls POST /api/skills/:name/disable and returns {ok: true}", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    } as Response);

    const result = await disableSkill("test-skill", "ws_123");
    expect(result).toEqual({ ok: true });
    expect(fetch).toHaveBeenCalledWith("/api/skills/test-skill/disable?workspace=ws_123", {
      method: "POST",
      signal: undefined,
    });
  });

  it("throws SkillApiError on 404", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 404,
    } as Response);

    await expect(disableSkill("unknown", "ws_123")).rejects.toThrow("Skill not found: unknown");
  });

  it("throws SkillApiError on non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    await expect(disableSkill("test-skill", "ws_123")).rejects.toThrow(SkillApiError);
  });
});

describe("SkillApiError", () => {
  it("has correct name and status properties", () => {
    const err = new SkillApiError("test error", 404);
    expect(err.name).toBe("SkillApiError");
    expect(err.status).toBe(404);
    expect(err.message).toBe("test error");
    expect(err).toBeInstanceOf(Error);
  });
});
